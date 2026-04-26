//go:build !windows

// Package signal handles OS signal management.
// See AI.md PART 8 for signal handling specifications.
package signal

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalHandlers configures signal handlers for graceful shutdown and log rotation.
// SIGTERM/SIGINT/SIGQUIT → shutdown callback
// SIGUSR1 → reopen log files (for rotation)
// SIGUSR2 → dump status to log
// SIGHUP → ignored (config auto-reloads via file watcher)
func SetupSignalHandlers(onShutdown func(), onReopenLogs func(), onDumpStatus func()) {
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	usr1Ch := make(chan os.Signal, 1)
	signal.Notify(usr1Ch, syscall.SIGUSR1)

	usr2Ch := make(chan os.Signal, 1)
	signal.Notify(usr2Ch, syscall.SIGUSR2)

	// Ignore SIGHUP (config auto-reloads)
	signal.Ignore(syscall.SIGHUP)

	go func() {
		for {
			select {
			case sig := <-shutdownCh:
				log.Printf("Received %s, initiating graceful shutdown...", sig)
				if onShutdown != nil {
					onShutdown()
				}
				return

			case <-usr1Ch:
				log.Println("Received SIGUSR1, reopening log files...")
				if onReopenLogs != nil {
					onReopenLogs()
				}

			case <-usr2Ch:
				log.Println("Received SIGUSR2, dumping status...")
				if onDumpStatus != nil {
					onDumpStatus()
				}
			}
		}
	}()
}
