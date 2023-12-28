package terraform

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/hashicorp/terraform-exec/tfexec"
)

const (
	Init        = "init"
	Plan        = "plan"
	Apply       = "apply"
	Destroy     = "destroy"
	Output      = "output"
	Show        = "show"
	ForceUnlock = "force-unlock"

	defaultAttempts = 3
	defaultDelay    = 5 * time.Second
)

var (
	AvailableActions = map[string]bool{
		Init:        true,
		Plan:        true,
		Apply:       true,
		Destroy:     true,
		Output:      true,
		Show:        true,
		ForceUnlock: true,
	}
)

var zipMimeTypes = []string{
	"application/x-zip-compressed",
	"application/zip",
}

func isContentTypeZip(contentType string) bool {
	for _, mt := range zipMimeTypes {
		if mt == contentType {
			return true
		}
	}
	return false
}

func setTerraformMultiStdout(tf *tfexec.Terraform, file string, debug bool) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}

	filesToBeDeleted = append(filesToBeDeleted, file)

	writers := []io.Writer{f}
	if debug {
		writers = append(writers, os.Stdout)
	}

	multi := io.MultiWriter(writers...)
	tf.SetStdout(multi)
	return nil
}

var onRetryFunc = func(n uint, err error) {
	logger.Debug("retrying...",
		slog.Uint64("count", uint64(n)),
		slog.Any("error", err),
	)
}

func setHeaders(request *http.Request, headers map[string]string) {
	if headers == nil {
		return
	}

	for k, v := range headers {
		request.Header.Set(k, v)
	}
}

func cleanUp() error {
	defer timer("cleanUp")()
	logger.Debug("cleaning up files...")
	for _, f := range filesToBeDeleted {
		if err := os.RemoveAll(f); err != nil {
			return err
		}
	}
	return nil
}

func errorCheck(f func() error) {
	if err := f(); err != nil {
		logger.Error("error while cleaning up", err)
	}
}

func timer(name string) func() {
	now := time.Now()
	return func() {
		logger.Debug("time taken to execute",
			slog.Duration(name, time.Since(now)),
		)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirSize(path string) (float32, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return float32(size) / 1024 / 1024, err
}

func retryOnError(f func() error) error {
	return retry.Do(
		f,
		retry.OnRetry(func(n uint, err error) {
			logger.Debug("retrying...",
				slog.String("error", err.Error()),
				slog.Uint64("attempt", uint64(n)+1),
			)
		}),
		retry.Attempts(defaultAttempts),
		retry.Delay(defaultDelay),
		retry.DelayType(retry.FixedDelay),
	)
}

// copyTFBinary copies the terraform binary if it's a custom driver configuration
func copyTFBinary(src, dst string) error {
	if !fileExists(src) {
		return nil
	}

	if err := copyFile(src, dst); err != nil {
		return err
	}

	return os.Chmod(dst, 0o700)
}
