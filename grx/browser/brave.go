package browser

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func launchBrave(proxyAddress string, customCertPath string, profileDir string) (*exec.Cmd, error) {
	log.Println("[launchBrave] Starting Brave launch process")

	// Use provided profile directory
	braveDataDir := profileDir
	log.Printf("[launchBrave] Brave data directory: %s", braveDataDir)

	// Create profile directory if it doesn't exist
	if err := os.MkdirAll(braveDataDir, 0755); err != nil {
		return nil, fmt.Errorf("[launchBrave] failed to create Brave data directory: %v", err)
	}
	log.Printf("[launchBrave] Created Brave data directory successfully")

	// Copy CA certificate to Brave's certificate store directory
	certPath := filepath.Join(braveDataDir, "ca.crt")
	log.Printf("[launchBrave] Copying certificate from %s to %s", customCertPath, certPath)
	if err := copyFile(customCertPath, certPath); err != nil {
		return nil, fmt.Errorf("[launchBrave] failed to copy certificate: %v", err)
	}
	log.Printf("[launchBrave] Certificate copied successfully")

	// Get certificate fingerprint for ignoring certificate errors
	var fingerprint string
	leafSpkiPath := filepath.Join(filepath.Dir(customCertPath), "leaf.spki")
	if data, err := os.ReadFile(leafSpkiPath); err == nil {
		fingerprint = string(data)
		log.Printf("[launchBrave] Using leaf SPKI from %s", leafSpkiPath)
	} else {
		log.Printf("[launchBrave] leaf SPKI not found (%v), calculating CA SPKI instead", err)
		fp, ferr := GetSPKIFingerprint(certPath)
		if ferr != nil {
			log.Printf("[launchBrave] Warning: couldn't calculate certificate fingerprint: %v", ferr)
		} else {
			fingerprint = fp
			log.Printf("[launchBrave] Certificate fingerprint calculated successfully")
		}
	}

	// Determine Brave executable path
	var bravePath string
	switch runtime.GOOS {
	case "darwin": // macOS
		bravePath = "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"
		if _, err := os.Stat(bravePath); err != nil {
			log.Printf("[launchBrave] Brave not found at primary path, trying alternate location")
			bravePath = "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"
		}
	case "linux":
		bravePath = "brave"
		// Try to find brave in PATH
		if path, err := exec.LookPath("brave"); err == nil {
			bravePath = path
		} else if path, err := exec.LookPath("brave-browser"); err == nil {
			bravePath = path
		}
	case "windows":
		bravePath = "C:\\Program Files\\BraveSoftware\\Brave-Browser\\Application\\brave.exe"
		if _, err := os.Stat(bravePath); err != nil {
			log.Printf("[launchBrave] Brave not found at primary path, trying alternative path")
			bravePath = "C:\\Program Files (x86)\\BraveSoftware\\Brave-Browser\\Application\\brave.exe"
		}
	default:
		return nil, fmt.Errorf("[launchBrave] unsupported operating system: %s", runtime.GOOS)
	}
	log.Printf("[launchBrave] Using Brave path: %s", bravePath)

	// Verify Brave executable exists
	if _, err := os.Stat(bravePath); err != nil {
		return nil, fmt.Errorf("[launchBrave] Brave executable not found at %s: %v", bravePath, err)
	}
	log.Printf("[launchBrave] Brave executable found and verified")

	// Construct Brave command line arguments (Chromium-based)
	args := []string{
		"--user-data-dir=" + braveDataDir,
		"--proxy-server=" + proxyAddress,
	}

	// Add certificate fingerprint to the ignore list
	if fingerprint != "" {
		args = append(args, "--ignore-certificate-errors-spki-list="+fingerprint)
	} else {
		args = append(args, "--ignore-certificate-errors")
	}

	// Add other standard Chromium arguments
	args = append(args,
		"--remote-debugging-port=0",
		"--allow-insecure-localhost",
		"--unsafely-treat-insecure-origin-as-secure",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-restore-session-state",
		"--disable-popup-blocking",
		"--disable-translate",
		"--disable-infobars",
		"--enable-features=SuppressDifferentOriginSubframeDialogs",
		"--disable-extensions-except=",
		"--disable-component-extensions-with-background-pages",
		"--start-maximized",
		"--disable-default-apps",
		"--disable-sync",
		"--enable-fixed-layout",
		"--noerrdialogs",
		"--test-type",
		"--disable-features=BraveRewards",
		"grroxy.com",
	)

	log.Printf("[launchBrave] Brave arguments: %v", args)

	// Launch Brave
	log.Printf("[launchBrave] Attempting to launch Brave with command: %s %v", bravePath, args)
	cmd := exec.Command(bravePath, args...)
	log.Println("[launchBrave] " + cmd.String())
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("[launchBrave] failed to launch Brave: %v", err)
	}

	log.Printf("[launchBrave] Brave process started successfully")
	log.Printf("[launchBrave] Brave profile at: %s", braveDataDir)
	return cmd, nil
}
