package browser

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// launchExistingTerminal attempts to inject proxy settings into an already-running terminal
// This uses platform-specific methods:
// - macOS: AppleScript to send commands to frontmost Terminal/iTerm window
// - Linux: Attempts to find and signal running terminal processes
// - Windows: Limited support - requires PowerShell remoting
func launchExistingTerminal(proxyAddress string, customCertPath string) (*exec.Cmd, error) {
	log.Println("[launchExistingTerminal] Starting existing terminal injection process")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("[launchExistingTerminal] failed to get home directory: %v", err)
	}

	// Create shell commands to set up proxy environment
	proxyCommands := fmt.Sprintf(`
# Grroxy Proxy Configuration
export HTTP_PROXY='%s'
export HTTPS_PROXY='%s'
export http_proxy='%s'
export https_proxy='%s'
export SSL_CERT_FILE='%s'
echo "[grroxy] Proxy configured: %s"
echo "[grroxy] Certificate: %s"
cd '%s'
`, proxyAddress, proxyAddress, proxyAddress, proxyAddress, customCertPath, proxyAddress, customCertPath, homeDir)

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin": // macOS
		// Try to inject into frontmost Terminal.app or iTerm2 window
		appleScript := fmt.Sprintf(`
tell application "System Events"
    set frontApp to name of first application process whose frontmost is true
    
    if frontApp is "Terminal" then
        tell application "Terminal"
            do script "%s" in front window
        end tell
        return "Injected into Terminal.app"
    else if frontApp is "iTerm2" or frontApp is "iTerm" then
        tell application "iTerm"
            tell current session of current window
                write text "%s"
            end tell
        end tell
        return "Injected into iTerm2"
    else
        -- Try to activate Terminal and inject there
        tell application "Terminal"
            activate
            delay 0.5
            do script "%s" in front window
        end tell
        return "Opened Terminal.app and injected"
    end if
end tell
`, escapeAppleScript(proxyCommands), escapeAppleScript(proxyCommands), escapeAppleScript(proxyCommands))

		cmd = exec.Command("osascript", "-e", appleScript)
		log.Printf("[launchExistingTerminal] Attempting to inject into frontmost terminal on macOS")

	case "linux":
		// On Linux, try to find a running terminal and inject via /proc or signal
		// This is more complex and terminal-specific
		// Common terminals: gnome-terminal, konsole, xfce4-terminal, xterm

		// First, try to check if any known terminal is running
		terminalCheck := exec.Command("pgrep", "-l", "-f", "(gnome-terminal|konsole|xfce4-terminal|xterm|alacritty|kitty|wezterm)")
		output, _ := terminalCheck.Output()

		if len(output) > 0 {
			log.Printf("[launchExistingTerminal] Found running terminals: %s", string(output))
			// Create a notification or use xclip to put commands in clipboard
			// since direct injection is difficult on Linux

			notificationCmd := fmt.Sprintf(`
export HTTP_PROXY='%s'
export HTTPS_PROXY='%s'
export http_proxy='%s'
export https_proxy='%s'
export SSL_CERT_FILE='%s'
echo "Proxy environment variables have been configured."`, proxyAddress, proxyAddress, proxyAddress, proxyAddress, customCertPath)

			// Try to use notify-send if available
			notifyCmd := exec.Command("notify-send", "Grroxy Proxy", "Proxy environment variables configured. Check your terminal.")
			notifyCmd.Run() // Ignore errors - notification is optional

			// Put the export commands in clipboard for user to paste
			if xclipPath, err := exec.LookPath("xclip"); err == nil {
				echoCmd := exec.Command("echo", "-n", notificationCmd)
				clipCmd := exec.Command(xclipPath, "-selection", "clipboard")
				pipe, _ := echoCmd.StdoutPipe()
				clipCmd.Stdin = pipe
				echoCmd.Start()
				clipCmd.Run()
				log.Printf("[launchExistingTerminal] Proxy commands copied to clipboard")
			}

			cmd = exec.Command("echo", "Proxy settings prepared. Commands copied to clipboard - paste into your terminal.")
		} else {
			// No terminal found running, open a new one
			log.Printf("[launchExistingTerminal] No running terminal found, falling back to fresh terminal")
			return launchTerminal(proxyAddress, customCertPath)
		}

	case "windows":
		// On Windows, try to use PowerShell to inject into existing console
		// This requires PowerShell remoting which may not always work
		psScript := fmt.Sprintf(`
$proxyAddr = '%s'
$certPath = '%s'

# Try to find existing PowerShell or CMD windows
$windows = Get-Process | Where-Object { $_.ProcessName -match "powershell|cmd" }

if ($windows) {
    Write-Host "[grroxy] Found running terminal windows. Please manually set the following environment variables:"
    Write-Host "  $env:HTTP_PROXY = '$proxyAddr'"
    Write-Host "  $env:HTTPS_PROXY = '$proxyAddr'"
    Write-Host "  $env:SSL_CERT_FILE = '$certPath'"
} else {
    Write-Host "[grroxy] No existing terminal found. Opening new terminal..."
}
`, proxyAddress, customCertPath)

		cmd = exec.Command("powershell.exe", "-Command", psScript)

		// After showing instructions, also try to launch a new terminal
		go func() {
			time.Sleep(2 * time.Second)
			launchTerminal(proxyAddress, customCertPath)
		}()

	default:
		return nil, fmt.Errorf("[launchExistingTerminal] unsupported operating system: %s", runtime.GOOS)
	}

	log.Printf("[launchExistingTerminal] Command: %s", cmd.String())

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("[launchExistingTerminal] failed to inject into existing terminal: %v", err)
	}

	log.Printf("[launchExistingTerminal] Terminal injection process started")
	return cmd, nil
}

// escapeAppleScript escapes special characters for AppleScript string literals
func escapeAppleScript(s string) string {
	// Escape double quotes and backslashes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
