package terraform

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
)

var (
	filesToBeDeleted []string
	logger           *slog.Logger
)

const (
	downloadArchiveName = "workdir.tar.zst"
	uploadArchiveName   = "job.tar.zst"
	planFile            = "plan.json"
	outputFile          = "output.json"
	appPath             = "app/"
	basePath            = "app/scratch/"
	terraformWorkDir    = "gen-ai-tf/"
	installDir          = "gen-ai-tf/app"
	overrideDir         = "tmp/overrides"
	secretMountPath     = "tmp/contextdata"
	binaryName          = "terraform"
)

type Installer interface {
	Install(context.Context) (string, error)
	Remove(context.Context) error
}

type tfLogger struct {
	logger *slog.Logger
}

func newTfLogger(slogger *slog.Logger) *tfLogger {
	return &tfLogger{logger: slogger}
}

func (l tfLogger) Printf(format string, v ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, v...))
}

type TerraformRunner struct {
	*DriverConfig
	workDir    string
	planPath   string
	outputPath string

	logger    *slog.Logger
	tfLog     *tfLogger
	installer Installer
}

func NewTerraformRunner(_logger *slog.Logger, driverConfig *DriverConfig) *TerraformRunner {
	t := &TerraformRunner{
		DriverConfig: driverConfig,
		workDir:      driverConfig.WorkDir,
		logger:       _logger,
		tfLog:        newTfLogger(_logger),
	}

	t.installer = &releases.ExactVersion{
		Product: product.Terraform,
		Version: version.Must(version.NewVersion(t.Version)),
		//InstallDir: installDir,
	}

	logger = _logger

	return t
}

func (t *TerraformRunner) install(ctx context.Context) (*tfexec.Terraform, error) {
	now := time.Now()

	var execPath string
	var err error
	err = retryOnError(func() error {
		execPath, err = t.installer.Install(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("error installing Terraform: %s", err)
	}

	tf, err := tfexec.NewTerraform(t.workDir, execPath)
	if err != nil {
		return nil, fmt.Errorf("error running NewTerraform: %s", err)
	}

	v, _, err := tf.Version(ctx, true)
	if err != nil {
		return nil, err
	}
	t.Version = v.String()

	t.logger.Debug("time taken to install terraform binary",
		slog.Duration("duration", time.Since(now)),
		slog.String("version", v.String()),
	)

	return tf, nil
}

func (t *TerraformRunner) runCommand(ctx context.Context, tf *tfexec.Terraform, action string) error {
	switch action {
	case Init:
		err := retryOnError(func() error {
			return tf.Init(ctx, t.GetInitOptions()...)
		})
		if err != nil {
			return fmt.Errorf("error running Init: %s", err)
		}
	case Plan:
		if _, err := tf.Plan(ctx, t.GetPlanOptions()...); err != nil {
			return fmt.Errorf("error running Plan: %s", err)
		}
		// if plan file is provided, execute show command and upload terraform json output
		if t.PlanFile != "" {
			goto showCase
		}
		break
	showCase:
		fallthrough
	case Show:
		if t.PlanFile == "" {
			return fmt.Errorf("please provide -plan-file flag  to show the terraform plan")
		}

		if err := setTerraformMultiStdout(tf, t.planPath, t.Debug); err != nil {
			return fmt.Errorf("error setting multi stdout to terraform: %s", err)
		}
		if _, err := tf.ShowPlanFile(ctx, t.PlanFile); err != nil {
			return fmt.Errorf("error running Show: %s", err)
		}
	case Apply:
		// do not write the output of apply to the plan file if both are in single activity
		tf.SetStdout(os.Stdout)
		if err := tf.Apply(ctx, t.GetApplyOptions()...); err != nil {
			return fmt.Errorf("error running Apply: %s", err)
		}
	case Destroy:
		tf.SetStdout(os.Stdout)
		if err := tf.Destroy(ctx, t.GetDestroyOptions()...); err != nil {
			return fmt.Errorf("error running Destroy: %s", err)
		}
	case Output:
		if err := setTerraformMultiStdout(tf, t.outputPath, t.Debug); err != nil {
			return fmt.Errorf("error setting multi stdout to terraform: %s", err)
		}
		if _, err := tf.Output(ctx); err != nil {
			return fmt.Errorf("error running Output: %s", err)
		}
	case ForceUnlock:
		tf.SetStdout(os.Stdout)
		if err := tf.ForceUnlock(ctx, t.LockID, t.GetForceUnlockOptions()...); err != nil {
			return fmt.Errorf("error running ForceUnlock: %s", err)
		}
	}
	return nil
}

func (t *TerraformRunner) run(ctx context.Context) error {
	defer timer("runTerraform")()

	tf, err := t.install(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_ = t.installer.Remove(ctx)
	}()

	if err = t.setTerraformLogger(tf); err != nil {
		return fmt.Errorf("error setting terraform logger: %s", err)
	}

	for _, action := range t.Actions {
		t.logger.Info("running terraform command",
			slog.String("action", action),
			slog.String("version", t.Version),
			slog.String("workdir", t.workDir),
		)

		if err = t.runCommand(ctx, tf, action); err != nil {
			return err
		}
	}

	return nil
}

func (t *TerraformRunner) setTerraformLogger(tf *tfexec.Terraform) error {
	// To print the terraform commands
	tf.SetLogger(t.tfLog)

	// display the output of terraform commands to the terminal
	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	// For terraform logs
	logLvl := t.GetLogLvl()
	err := tf.SetLogProvider(logLvl)
	if err != nil {
		return err
	}

	err = tf.SetLog(logLvl)
	if err != nil {
		return err
	}

	return tf.SetLogPath("./job.log")
}

func (t *TerraformRunner) Execute() error {
	defer timer("main")()

	defer errorCheck(cleanUp)

	if err := os.MkdirAll(terraformWorkDir, 0755); err != nil {
		logger.Error("unable to create job dir", "error", err)
		return err
	}

	t.planPath = path.Join(basePath, planFile)
	t.outputPath = path.Join(basePath, outputFile)

	// install terraform binary and run the terraform commands
	if err := t.run(context.Background()); err != nil {
		logger.Error("failed to run terraform job", "error", err)
		return err
	}

	return nil
}
