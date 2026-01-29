//go:build windows

package internal

import (
	"os"
)

func setupSignalReload(reloadFunc func()) chan os.Signal {
	// SIGUSR1 is not supported on Windows
	// Return nil channel that will never receive
	return nil
}
