package browser

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
)

func launchTerminal(proxyAddress string, customCertPath string) (*exec.Cmd, error) {
	log.Println("[launchTerminal] Starting terminal launch process")

	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("[launchTerminal] failed to get home directory: %v", err)
	}
	log.Printf("[launchTerminal] Home directory: %s", homeDir)

	var cmd *exec.Cmd

	// Determine terminal executable and arguments based on OS
	switch runtime.GOOS {
	case "darwin": // macOS
		// Use Terminal.app on macOS
		terminalScript := fmt.Sprintf(`
tell application "Terminal"
    do script "echo 'Terminal configured with proxy: %s' && export HTTP_PROXY='%s' && export HTTPS_PROXY='%s' && export http_proxy='%s' && export https_proxy='%s' && export SSL_CERT_FILE='%s' && cd '%s' && exec $SHELL"
    activate
end tell
`, proxyAddress, proxyAddress, proxyAddress, proxyAddress, proxyAddress, customCertPath, homeDir)

		cmd = exec.Command("osascript", "-e", terminalScript)
		log.Printf("[launchTerminal] Launching macOS Terminal with proxy configuration")

	case "linux":
		// Try various Linux terminal emulators in order of preference
		terminals := []string{"gnome-terminal", "konsole", "xfce4-terminal", "xterm"}
		var terminalPath string

		for _, term := range terminals {
			if path, err := exec.LookPath(term); err == nil {
				terminalPath = path
				log.Printf("[launchTerminal] Found terminal: %s", term)
				break
			}
		}

		if terminalPath == "" {
			return nil, fmt.Errorf("[launchTerminal] no suitable terminal emulator found (tried: %v)", terminals)
		}

		// Create a shell script to set up proxy environment
		shellScript := fmt.Sprintf(`#!/bin/bash
echo "Terminal configured with proxy: %s"
export HTTP_PROXY='%s'
export HTTPS_PROXY='%s'
export http_proxy='%s'
export https_proxy='%s'
export SSL_CERT_FILE='%s'
cd '%s'
exec $SHELL
`, proxyAddress, proxyAddress, proxyAddress, proxyAddress, proxyAddress, customCertPath, homeDir)

		// Launch terminal based on which one was found
		switch terminalPath {
		case "/usr/bin/gnome-terminal", "/bin/gnome-terminal":
			cmd = exec.Command(terminalPath, "--", "bash", "-c", shellScript)
		case "/usr/bin/konsole", "/bin/konsole":
			cmd = exec.Command(terminalPath, "-e", "bash", "-c", shellScript)
		case "/usr/bin/xfce4-terminal", "/bin/xfce4-terminal":
			cmd = exec.Command(terminalPath, "-e", "bash", "-c", shellScript)
		default: // xterm or other
			cmd = exec.Command(terminalPath, "-e", "bash", "-c", shellScript)
		}

	case "windows":
		// Use PowerShell on Windows
		psScript := fmt.Sprintf(`$env:HTTP_PROXY='%s'; $env:HTTPS_PROXY='%s'; $env:SSL_CERT_FILE='%s'; Write-Host 'Terminal configured with proxy: %s'; Set-Location '%s'`,
			proxyAddress, proxyAddress, customCertPath, proxyAddress, homeDir)

		cmd = exec.Command("powershell.exe", "-NoExit", "-Command", psScript)
		log.Printf("[launchTerminal] Launching Windows PowerShell with proxy configuration")

	default:
		return nil, fmt.Errorf("[launchTerminal] unsupported operating system: %s", runtime.GOOS)
	}

	log.Printf("[launchTerminal] Terminal command: %s", cmd.String())

	// Set environment variables for the child process
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HTTP_PROXY=%s", proxyAddress),
		fmt.Sprintf("HTTPS_PROXY=%s", proxyAddress),
		fmt.Sprintf("http_proxy=%s", proxyAddress),
		fmt.Sprintf("https_proxy=%s", proxyAddress),
		fmt.Sprintf("SSL_CERT_FILE=%s", customCertPath),
	)

	// Launch terminal
	log.Printf("[launchTerminal] Attempting to launch terminal")
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("[launchTerminal] failed to launch terminal: %v", err)
	}

	log.Printf("[launchTerminal] Terminal process started successfully with PID: %d", cmd.Process.Pid)
	log.Printf("[launchTerminal] Proxy configured: %s", proxyAddress)
	log.Printf("[launchTerminal] Certificate path: %s", customCertPath)

	return cmd, nil
}
