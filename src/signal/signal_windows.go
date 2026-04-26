//go:build windows

// Package signal handles OS signal management on Windows.
// Windows only supports SIGTERM and SIGINT.
package signal

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalHandlers configures signal handlers for graceful shutdown.
// On Windows, only SIGTERM and SIGINT are supported.
// SIGUSR1/SIGUSR2 are not available on Windows.
func SetupSignalHandlers(onShutdown func(), onReopenLogs func(), onDumpStatus func()) {
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-shutdownCh
		log.Printf("Received %s, initiating graceful shutdown...", sig)
		if onShutdown != nil {
			onShutdown()
		}
	}()
}
