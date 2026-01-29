//go:build !windows

package internal

import (
	"os"
	"os/signal"
	"syscall"
)

func setupSignalReload(reloadFunc func()) chan os.Signal {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1)
	go func() {
		for range sigCh {
			reloadFunc()
		}
	}()
	return sigCh
}
