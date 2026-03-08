package browser

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func launchEdge(proxyAddress string, customCertPath string, profileDir string) (*exec.Cmd, error) {
	log.Println("[launchEdge] Starting Edge launch process")

	// Use provided profile directory
	edgeDataDir := profileDir
	log.Printf("[launchEdge] Edge data directory: %s", edgeDataDir)

	// Create profile directory if it doesn't exist
	if err := os.MkdirAll(edgeDataDir, 0755); err != nil {
		return nil, fmt.Errorf("[launchEdge] failed to create Edge data directory: %v", err)
	}
	log.Printf("[launchEdge] Created Edge data directory successfully")

	// Copy CA certificate to Edge's certificate store directory
	certPath := filepath.Join(edgeDataDir, "ca.crt")
	log.Printf("[launchEdge] Copying certificate from %s to %s", customCertPath, certPath)
	if err := copyFile(customCertPath, certPath); err != nil {
		return nil, fmt.Errorf("[launchEdge] failed to copy certificate: %v", err)
	}
	log.Printf("[launchEdge] Certificate copied successfully")

	// Get certificate fingerprint
	var fingerprint string
	leafSpkiPath := filepath.Join(filepath.Dir(customCertPath), "leaf.spki")
	if data, err := os.ReadFile(leafSpkiPath); err == nil {
		fingerprint = string(data)
		log.Printf("[launchEdge] Using leaf SPKI from %s", leafSpkiPath)
	} else {
		log.Printf("[launchEdge] leaf SPKI not found (%v), calculating CA SPKI instead", err)
		fp, ferr := GetSPKIFingerprint(certPath)
		if ferr != nil {
			log.Printf("[launchEdge] Warning: couldn't calculate certificate fingerprint: %v", ferr)
		} else {
			fingerprint = fp
			log.Printf("[launchEdge] Certificate fingerprint calculated successfully")
		}
	}

	// Determine Edge executable path
	var edgePath string
	switch runtime.GOOS {
	case "darwin": // macOS
		edgePath = "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"
		if _, err := os.Stat(edgePath); err != nil {
			log.Printf("[launchEdge] Edge not found at primary path, trying alternate location")
			edgePath = "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"
		}
	case "linux":
		// Try various Edge executable names on Linux
		possiblePaths := []string{"microsoft-edge", "microsoft-edge-stable", "msedge"}
		edgePath = "microsoft-edge" // default
		for _, path := range possiblePaths {
			if foundPath, err := exec.LookPath(path); err == nil {
				edgePath = foundPath
				log.Printf("[launchEdge] Found Edge at: %s", foundPath)
				break
			}
		}
	case "windows":
		edgePath = "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe"
		if _, err := os.Stat(edgePath); err != nil {
			log.Printf("[launchEdge] Edge not found at primary path, trying alternative path")
			edgePath = "C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe"
		}
	default:
		return nil, fmt.Errorf("[launchEdge] unsupported operating system: %s", runtime.GOOS)
	}
	log.Printf("[launchEdge] Using Edge path: %s", edgePath)

	// Verify Edge executable exists
	if _, err := os.Stat(edgePath); err != nil {
		return nil, fmt.Errorf("[launchEdge] Edge executable not found at %s: %v", edgePath, err)
	}
	log.Printf("[launchEdge] Edge executable found and verified")

	// Construct Edge command line arguments (Chromium-based)
	args := []string{
		"--user-data-dir=" + edgeDataDir,
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

	log.Printf("[launchEdge] Edge arguments: %v", args)

	// Launch Edge
	log.Printf("[launchEdge] Attempting to launch Edge with command: %s %v", edgePath, args)
	cmd := exec.Command(edgePath, args...)
	log.Println("[launchEdge] " + cmd.String())
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("[launchEdge] failed to launch Edge: %v", err)
	}

	log.Printf("[launchEdge] Edge process started successfully")
	log.Printf("[launchEdge] Edge profile at: %s", edgeDataDir)
	return cmd, nil
}
