package browser

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func launchVivaldi(proxyAddress string, customCertPath string, profileDir string) (*exec.Cmd, error) {
	log.Println("[launchVivaldi] Starting Vivaldi launch process")

	// Use provided profile directory
	vivaldiDataDir := profileDir
	log.Printf("[launchVivaldi] Vivaldi data directory: %s", vivaldiDataDir)

	// Create profile directory if it doesn't exist
	if err := os.MkdirAll(vivaldiDataDir, 0755); err != nil {
		return nil, fmt.Errorf("[launchVivaldi] failed to create Vivaldi data directory: %v", err)
	}
	log.Printf("[launchVivaldi] Created Vivaldi data directory successfully")

	// Copy CA certificate
	certPath := filepath.Join(vivaldiDataDir, "ca.crt")
	log.Printf("[launchVivaldi] Copying certificate from %s to %s", customCertPath, certPath)
	if err := copyFile(customCertPath, certPath); err != nil {
		return nil, fmt.Errorf("[launchVivaldi] failed to copy certificate: %v", err)
	}
	log.Printf("[launchVivaldi] Certificate copied successfully")

	// Get certificate fingerprint
	var fingerprint string
	leafSpkiPath := filepath.Join(filepath.Dir(customCertPath), "leaf.spki")
	if data, err := os.ReadFile(leafSpkiPath); err == nil {
		fingerprint = string(data)
		log.Printf("[launchVivaldi] Using leaf SPKI from %s", leafSpkiPath)
	} else {
		log.Printf("[launchVivaldi] leaf SPKI not found (%v), calculating CA SPKI instead", err)
		fp, ferr := GetSPKIFingerprint(certPath)
		if ferr != nil {
			log.Printf("[launchVivaldi] Warning: couldn't calculate certificate fingerprint: %v", ferr)
		} else {
			fingerprint = fp
			log.Printf("[launchVivaldi] Certificate fingerprint calculated successfully")
		}
	}

	// Determine Vivaldi executable path
	var vivaldiPath string
	switch runtime.GOOS {
	case "darwin": // macOS
		vivaldiPath = "/Applications/Vivaldi.app/Contents/MacOS/Vivaldi"
		if _, err := os.Stat(vivaldiPath); err != nil {
			// Try alternative location
			vivaldiPath = "/Applications/Vivaldi.app/Contents/MacOS/Vivaldi"
		}
	case "linux":
		vivaldiPath = "vivaldi"
		// Try to find vivaldi in PATH
		if path, err := exec.LookPath("vivaldi"); err == nil {
			vivaldiPath = path
		} else if path, err := exec.LookPath("vivaldi-stable"); err == nil {
			vivaldiPath = path
		}
	case "windows":
		// Vivaldi is typically installed in LocalAppData on Windows
		homeDir, _ := os.UserHomeDir()
		vivaldiPath = homeDir + "\\AppData\\Local\\Vivaldi\\Application\\vivaldi.exe"
		if _, err := os.Stat(vivaldiPath); err != nil {
			log.Printf("[launchVivaldi] Vivaldi not found at primary path, trying alternative path")
			vivaldiPath = "C:\\Users\\" + os.Getenv("USERNAME") + "\\AppData\\Local\\Vivaldi\\Application\\vivaldi.exe"
		}
	default:
		return nil, fmt.Errorf("[launchVivaldi] unsupported operating system: %s", runtime.GOOS)
	}
	log.Printf("[launchVivaldi] Using Vivaldi path: %s", vivaldiPath)

	// Verify Vivaldi executable exists
	if _, err := os.Stat(vivaldiPath); err != nil {
		return nil, fmt.Errorf("[launchVivaldi] Vivaldi executable not found at %s: %v", vivaldiPath, err)
	}
	log.Printf("[launchVivaldi] Vivaldi executable found and verified")

	// Construct Vivaldi command line arguments (Chromium-based)
	args := []string{
		"--user-data-dir=" + vivaldiDataDir,
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

	log.Printf("[launchVivaldi] Vivaldi arguments: %v", args)

	// Launch Vivaldi
	log.Printf("[launchVivaldi] Attempting to launch Vivaldi with command: %s %v", vivaldiPath, args)
	cmd := exec.Command(vivaldiPath, args...)
	log.Println("[launchVivaldi] " + cmd.String())
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("[launchVivaldi] failed to launch Vivaldi: %v", err)
	}

	log.Printf("[launchVivaldi] Vivaldi process started successfully")
	log.Printf("[launchVivaldi] Vivaldi profile at: %s", vivaldiDataDir)
	return cmd, nil
}
