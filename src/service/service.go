// Package service handles system service management (install, uninstall, start, stop).
// See AI.md PART 24, 25 for service specifications.
// Supports systemd (Linux), launchd (macOS), rc.d (BSD), Windows Service Manager.
package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	projectOrg  = "casapps"
	projectName = "ytdl"
)

// HandleServiceCommand processes --service flag arguments
func HandleServiceCommand(binaryName string, args []string) {
	if len(args) == 0 || args[0] == "--help" {
		printServiceHelp(binaryName)
		os.Exit(0)
	}

	switch args[0] {
	case "start":
		startService()
	case "stop":
		stopService()
	case "restart":
		stopService()
		startService()
	case "reload":
		reloadService()
	case "--install":
		installService(binaryName)
	case "--uninstall":
		uninstallService()
	case "--disable":
		disableService()
	default:
		fmt.Fprintf(os.Stderr, "Unknown service command: %s\n", args[0])
		printServiceHelp(binaryName)
		os.Exit(1)
	}
}

func printServiceHelp(binaryName string) {
	fmt.Printf("Usage: %s --service <command>\n\n", binaryName)
	fmt.Println("Commands:")
	fmt.Println("  start         Start the service")
	fmt.Println("  stop          Stop the service")
	fmt.Println("  restart       Restart the service")
	fmt.Println("  reload        Reload configuration")
	fmt.Println("  --install     Install as system service")
	fmt.Println("  --uninstall   Remove system service")
	fmt.Println("  --disable     Disable service autostart")
}

func startService() {
	switch runtime.GOOS {
	case "linux":
		runCmd("systemctl", "start", projectName)
	case "darwin":
		runCmd("launchctl", "load", launchdPlistPath())
	case "freebsd", "openbsd", "netbsd":
		runCmd("service", projectName, "start")
	case "windows":
		runCmd("net", "start", projectName)
	default:
		fmt.Fprintf(os.Stderr, "Service management not supported on %s\n", runtime.GOOS)
		os.Exit(1)
	}
}

func stopService() {
	switch runtime.GOOS {
	case "linux":
		runCmd("systemctl", "stop", projectName)
	case "darwin":
		runCmd("launchctl", "unload", launchdPlistPath())
	case "freebsd", "openbsd", "netbsd":
		runCmd("service", projectName, "stop")
	case "windows":
		runCmd("net", "stop", projectName)
	default:
		fmt.Fprintf(os.Stderr, "Service management not supported on %s\n", runtime.GOOS)
		os.Exit(1)
	}
}

func reloadService() {
	switch runtime.GOOS {
	case "linux":
		runCmd("systemctl", "reload", projectName)
	default:
		// Fallback: restart
		stopService()
		startService()
	}
}

func installService(binaryName string) {
	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = filepath.Join("/usr/local/bin", binaryName)
	}

	switch runtime.GOOS {
	case "linux":
		installSystemdService(binaryPath)
	case "darwin":
		installLaunchdService(binaryPath)
	case "freebsd", "openbsd", "netbsd":
		installRCDService(binaryPath)
	default:
		fmt.Fprintf(os.Stderr, "Service installation not supported on %s\n", runtime.GOOS)
		os.Exit(1)
	}
}

func uninstallService() {
	stopService()

	switch runtime.GOOS {
	case "linux":
		os.Remove(fmt.Sprintf("/etc/systemd/system/%s.service", projectName))
		runCmd("systemctl", "daemon-reload")
	case "darwin":
		os.Remove(launchdPlistPath())
	case "freebsd", "openbsd", "netbsd":
		os.Remove(fmt.Sprintf("/usr/local/etc/rc.d/%s", projectName))
	}

	fmt.Println("Service uninstalled")
}

func disableService() {
	switch runtime.GOOS {
	case "linux":
		runCmd("systemctl", "disable", projectName)
	case "darwin":
		runCmd("launchctl", "unload", "-w", launchdPlistPath())
	}
	fmt.Println("Service disabled")
}

func installSystemdService(binaryPath string) {
	// See AI.md PART 25 for systemd service file
	unit := fmt.Sprintf(`[Unit]
Description=ytdl - Self-hosted media downloader
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`, binaryPath)

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", projectName)
	if err := os.WriteFile(servicePath, []byte(unit), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing service file: %v\n", err)
		os.Exit(1)
	}

	runCmd("systemctl", "daemon-reload")
	runCmd("systemctl", "enable", projectName)
	fmt.Printf("Service installed: %s\n", servicePath)
	fmt.Printf("Start with: systemctl start %s\n", projectName)
}

func installLaunchdService(binaryPath string) {
	// See AI.md PART 25 for launchd plist
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.%s.%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
`, projectOrg, projectName, binaryPath)

	plistPath := launchdPlistPath()
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing plist: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Service installed: %s\n", plistPath)
	fmt.Printf("Start with: launchctl load %s\n", plistPath)
}

func installRCDService(binaryPath string) {
	script := fmt.Sprintf(`#!/bin/sh
#
# PROVIDE: %s
# REQUIRE: NETWORKING
# KEYWORD: shutdown

. /etc/rc.subr

name="%s"
rcvar="%s_enable"
command="%s"
command_args="&"

run_rc_command "$1"
`, projectName, projectName, projectName, binaryPath)

	rcPath := fmt.Sprintf("/usr/local/etc/rc.d/%s", projectName)
	if err := os.WriteFile(rcPath, []byte(script), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing rc.d script: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Service installed: %s\n", rcPath)
	fmt.Printf("Enable: sysrc %s_enable=YES\n", projectName)
}

func launchdPlistPath() string {
	if os.Getuid() == 0 {
		return fmt.Sprintf("/Library/LaunchDaemons/com.%s.%s.plist", projectOrg, projectName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", fmt.Sprintf("com.%s.%s.plist", projectOrg, projectName))
}

func runCmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Don't exit on non-critical failures
		if !strings.Contains(err.Error(), "exit status") {
			fmt.Fprintf(os.Stderr, "Warning: %s %v: %v\n", name, args, err)
		}
	}
}
