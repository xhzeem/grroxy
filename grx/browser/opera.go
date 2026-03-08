package browser

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func launchOpera(proxyAddress string, customCertPath string, profileDir string) (*exec.Cmd, error) {
	log.Println("[launchOpera] Starting Opera launch process")

	// Use provided profile directory
	operaDataDir := profileDir
	log.Printf("[launchOpera] Opera data directory: %s", operaDataDir)

	// Create profile directory if it doesn't exist
	if err := os.MkdirAll(operaDataDir, 0755); err != nil {
		return nil, fmt.Errorf("[launchOpera] failed to create Opera data directory: %v", err)
	}
	log.Printf("[launchOpera] Created Opera data directory successfully")

	// Copy CA certificate
	certPath := filepath.Join(operaDataDir, "ca.crt")
	log.Printf("[launchOpera] Copying certificate from %s to %s", customCertPath, certPath)
	if err := copyFile(customCertPath, certPath); err != nil {
		return nil, fmt.Errorf("[launchOpera] failed to copy certificate: %v", err)
	}
	log.Printf("[launchOpera] Certificate copied successfully")

	// Get certificate fingerprint
	var fingerprint string
	leafSpkiPath := filepath.Join(filepath.Dir(customCertPath), "leaf.spki")
	if data, err := os.ReadFile(leafSpkiPath); err == nil {
		fingerprint = string(data)
		log.Printf("[launchOpera] Using leaf SPKI from %s", leafSpkiPath)
	} else {
		log.Printf("[launchOpera] leaf SPKI not found (%v), calculating CA SPKI instead", err)
		fp, ferr := GetSPKIFingerprint(certPath)
		if ferr != nil {
			log.Printf("[launchOpera] Warning: couldn't calculate certificate fingerprint: %v", ferr)
		} else {
			fingerprint = fp
			log.Printf("[launchOpera] Certificate fingerprint calculated successfully")
		}
	}

	// Determine Opera executable path
	var operaPath string
	switch runtime.GOOS {
	case "darwin": // macOS
		operaPath = "/Applications/Opera.app/Contents/MacOS/Opera"
		if _, err := os.Stat(operaPath); err != nil {
			// Try alternative location
			operaPath = "/Applications/Opera.app/Contents/MacOS/Opera"
		}
	case "linux":
		// Try various Opera executable names on Linux
		possiblePaths := []string{"opera", "opera-stable"}
		operaPath = "opera" // default
		for _, path := range possiblePaths {
			if foundPath, err := exec.LookPath(path); err == nil {
				operaPath = foundPath
				log.Printf("[launchOpera] Found Opera at: %s", foundPath)
				break
			}
		}
	case "windows":
		// Opera is typically installed in AppData\Local on Windows
		homeDir, _ := os.UserHomeDir()
		operaPath = homeDir + "\\AppData\\Local\\Programs\\Opera\\opera.exe"
		if _, err := os.Stat(operaPath); err != nil {
			log.Printf("[launchOpera] Opera not found at primary path, trying alternative path")
			// Try alternative paths
			operaPath = "C:\\Users\\" + os.Getenv("USERNAME") + "\\AppData\\Local\\Programs\\Opera\\opera.exe"
			if _, err := os.Stat(operaPath); err != nil {
				operaPath = "C:\\Program Files\\Opera\\opera.exe"
			}
		}
	default:
		return nil, fmt.Errorf("[launchOpera] unsupported operating system: %s", runtime.GOOS)
	}
	log.Printf("[launchOpera] Using Opera path: %s", operaPath)

	// Verify Opera executable exists
	if _, err := os.Stat(operaPath); err != nil {
		return nil, fmt.Errorf("[launchOpera] Opera executable not found at %s: %v", operaPath, err)
	}
	log.Printf("[launchOpera] Opera executable found and verified")

	// Construct Opera command line arguments (Chromium-based)
	args := []string{
		"--user-data-dir=" + operaDataDir,
		"--proxy-server=" + proxyAddress,
	}

	// Add certificate fingerprint
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
		"grroxy.com",
	)

	log.Printf("[launchOpera] Opera arguments: %v", args)

	// Launch Opera
	log.Printf("[launchOpera] Attempting to launch Opera with command: %s %v", operaPath, args)
	cmd := exec.Command(operaPath, args...)
	log.Println("[launchOpera] " + cmd.String())
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("[launchOpera] failed to launch Opera: %v", err)
	}

	log.Printf("[launchOpera] Opera process started successfully")
	log.Printf("[launchOpera] Opera profile at: %s", operaDataDir)
	return cmd, nil
}
