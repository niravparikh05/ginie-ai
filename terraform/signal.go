package terraform

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Adapted from https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/manager/signals/signal.go

var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

var onlyOneSignalHandler = make(chan struct{})

// SetupSignalHandler registers for SIGTERM and SIGINT. A context is returned
// which is canceled on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func setupSignalHandler() context.Context {
	close(onlyOneSignalHandler) // panics when called twice

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		abort(ctx)
		cancel()

		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return ctx
}

func abort(ctx context.Context) {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		logger.Error("failed to get processes", "error", err)
	}

	for _, p := range processes {
		name, err := p.NameWithContext(ctx)
		if err != nil {
			logger.Error("failed to get process name", "error", err)
		}

		if name != "terraform" {
			continue
		}

		pid := int(p.Pid)
		pgid, err := syscall.Getpgid(pid)
		if err != nil {
			logger.Error("failed to get terraform process group id", "error", err)
		}

		logger.Debug("killing terraform process")

		if err = syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
			logger.Error("failed to kill terraform process", "error", err)
		}

		// wait for terraform to terminate
		for pgid != -1 {
			pgid, _ = syscall.Getpgid(pid)
			time.Sleep(time.Second * 1)
		}

		break
	}
}
