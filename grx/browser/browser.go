package browser

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func LaunchBrowser(browserType string, proxyAddress string, customCertPath string, profileDir string) (*exec.Cmd, error) {
	if browserType == "" {
		browserType = "chrome" // Default to Chrome
	}

	browserType = strings.ToLower(browserType)

	switch browserType {
	case "chrome":
		return launchChrome(proxyAddress, customCertPath, profileDir)
	case "firefox":
		return launchFirefox(proxyAddress, customCertPath, profileDir)
	case "safari":
		return launchSafari(proxyAddress, customCertPath, profileDir)
	case "terminal":
		return launchTerminal(proxyAddress, customCertPath)
	case "brave":
		return launchBrave(proxyAddress, customCertPath, profileDir)
	case "edge":
		return launchEdge(proxyAddress, customCertPath, profileDir)
	case "vivaldi":
		return launchVivaldi(proxyAddress, customCertPath, profileDir)
	case "opera":
		return launchOpera(proxyAddress, customCertPath, profileDir)
	default:
		return nil, fmt.Errorf("unsupported browser: %s", browserType)
	}
}

// copyFile is kept in this file as it's used by all browser implementations
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0644)
}
