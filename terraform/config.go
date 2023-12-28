package terraform

import (
	"fmt"
	"log/slog"
	"strings"
)

const (
	defaultLockTimeout = "0s"
	defaultParallelism = 10
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type DriverConfig struct {
	Actions                    arrayFlags
	Version                    string
	WorkDir                    string
	PlanFile                   string
	Backend                    bool
	BackendConfig              arrayFlags
	ForceCopy                  bool
	FromModule                 string
	Get                        bool
	GetPlugins                 bool
	Lock                       bool
	LockTimeout                string
	Reconfigure                bool
	Upgrade                    bool
	VerifyPlugins              bool
	Var                        arrayFlags
	VarFile                    arrayFlags
	Target                     arrayFlags
	Replace                    arrayFlags
	Refresh                    bool
	Destroy                    bool
	Parallelism                int
	Backup                     string
	StateOut                   string
	DownloadUrl                string
	DownloadToken              string
	UploadUrl                  string
	UploadToken                string
	Debug                      bool
	LockID                     string
	OverrideTfDownloadEndpoint string
	SkipTLSVerify              bool

	logLvl *slog.LevelVar
}

func NewDriverConfig(actions arrayFlags, version, workDir string) *DriverConfig {
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelDebug)
	return &DriverConfig{
		Actions: actions,
		Version: version,
		WorkDir: workDir,
		logLvl:  logLevel,
	}
}

func (d *DriverConfig) GetLogLvl() string {
	return d.logLvl.Level().String()
}

func (d *DriverConfig) Validate() error {
	if len(d.Actions) == 0 {
		return fmt.Errorf("-action flag is required")
	}

	for _, action := range d.Actions {
		if !AvailableActions[action] {
			return fmt.Errorf("invalid action: %s", action)
		}

		if action == ForceUnlock && d.LockID == "" {
			return fmt.Errorf("-lock-id flag is required when -force-unlock is used")
		}
	}

	if d.DownloadUrl == "" {
		return fmt.Errorf("-download-url flag is required")
	}

	return nil
}
