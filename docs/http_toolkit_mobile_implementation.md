# HTTP Toolkit-Style Mobile Support Implementation

## HTTP Toolkit Approach Analysis

HTTP Toolkit's mobile support works by:
1. **Automatic device discovery** via ADB (Android) and iOS tools
2. **One-click proxy configuration** using system APIs
3. **Automatic certificate installation** with user prompts
4. **Seamless traffic interception** through the desktop proxy
5. **Visual device management** in the web UI

## Minimal Viable Implementation

### Step 1: Core Mobile Detection (Week 1)

```go
// grx/mobile/detect.go
package mobile

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
    "sync"
    "time"
)

type Device struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    Type         string    `json:"type"` // "android" or "ios"
    OSVersion    string    `json:"osVersion"`
    Connected    bool      `json:"connected"`
    ProxyEnabled bool      `json:"proxyEnabled"`
    CertInstalled bool     `json:"certInstalled"`
    LastSeen     time.Time `json:"lastSeen"`
}

type Detector struct {
    devices map[string]*Device
    mu      sync.RWMutex
    ctx     context.Context
    cancel  context.CancelFunc
}

func NewDetector() *Detector {
    ctx, cancel := context.WithCancel(context.Background())
    return &Detector{
        devices: make(map[string]*Device),
        ctx:     ctx,
        cancel:  cancel,
    }
}

func (d *Detector) Start() {
    go d.discoveryLoop()
}

func (d *Detector) Stop() {
    d.cancel()
}

func (d *Detector) discoveryLoop() {
    ticker := time.NewTicker(3 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-d.ctx.Done():
            return
        case <-ticker.C:
            d.scanDevices()
        }
    }
}

func (d *Detector) scanDevices() {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    // Mark all as disconnected initially
    for _, device := range d.devices {
        device.Connected = false
    }
    
    // Scan Android devices
    androidDevices := d.scanAndroidDevices()
    for _, device := range androidDevices {
        existing, exists := d.devices[device.ID]
        if exists {
            existing.Connected = true
            existing.LastSeen = time.Now()
        } else {
            d.devices[device.ID] = device
        }
    }
    
    // Scan iOS devices
    iosDevices := d.scanIOSDevices()
    for _, device := range iosDevices {
        existing, exists := d.devices[device.ID]
        if exists {
            existing.Connected = true
            existing.LastSeen = time.Now()
        } else {
            d.devices[device.ID] = device
        }
    }
}

func (d *Detector) scanAndroidDevices() []*Device {
    cmd := exec.Command("adb", "devices", "-l")
    output, err := cmd.Output()
    if err != nil {
        return nil
    }
    
    var devices []*Device
    lines := strings.Split(string(output), "\n")
    
    for _, line := range lines[1:] { // Skip header
        if strings.TrimSpace(line) == "" {
            continue
        }
        
        parts := strings.Fields(line)
        if len(parts) < 2 || parts[1] != "device" {
            continue
        }
        
        deviceID := parts[0]
        
        // Get device model
        modelCmd := exec.Command("adb", "-s", deviceID, "shell", "getprop", "ro.product.model")
        model, _ := modelCmd.Output()
        modelName := strings.TrimSpace(string(model))
        
        // Get OS version
        versionCmd := exec.Command("adb", "-s", deviceID, "shell", "getprop", "ro.build.version.release")
        version, _ := versionCmd.Output()
        osVersion := strings.TrimSpace(string(version))
        
        device := &Device{
            ID:        deviceID,
            Name:      modelName,
            Type:      "android",
            OSVersion: osVersion,
            Connected: true,
            LastSeen:  time.Now(),
        }
        
        devices = append(devices, device)
    }
    
    return devices
}

func (d *Detector) scanIOSDevices() []*Device {
    cmd := exec.Command("idevice_id", "-l")
    output, err := cmd.Output()
    if err != nil {
        return nil
    }
    
    var devices []*Device
    lines := strings.Split(strings.TrimSpace(string(output)), "\n")
    
    for _, deviceID := range lines {
        if deviceID == "" {
            continue
        }
        
        // Get device name
        nameCmd := exec.Command("ideviceinfo", "-u", deviceID, "-k", "DeviceName")
        name, _ := nameCmd.Output()
        deviceName := strings.TrimSpace(string(name))
        
        // Get iOS version
        versionCmd := exec.Command("ideviceinfo", "-u", deviceID, "-k", "ProductVersion")
        version, _ := versionCmd.Output()
        osVersion := strings.TrimSpace(string(version))
        
        device := &Device{
            ID:        deviceID,
            Name:      deviceName,
            Type:      "ios",
            OSVersion: osVersion,
            Connected: true,
            LastSeen:  time.Now(),
        }
        
        devices = append(devices, device)
    }
    
    return devices
}

func (d *Detector) GetDevices() []*Device {
    d.mu.RLock()
    defer d.mu.RUnlock()
    
    var devices []*Device
    for _, device := range d.devices {
        devices = append(devices, device)
    }
    
    return devices
}

func (d *Detector) GetDevice(id string) *Device {
    d.mu.RLock()
    defer d.mu.RUnlock()
    
    return d.devices[id]
}
```

### Step 2: One-Click Android Proxy Setup (Week 1)

```go
// grx/mobile/android.go
package mobile

import (
    "fmt"
    "net"
    "os/exec"
)

type AndroidDevice struct {
    *Device
}

func (d *AndroidDevice) ConfigureProxy(proxyAddr string) error {
    host, port, err := net.SplitHostPort(proxyAddr)
    if err != nil {
        return fmt.Errorf("invalid proxy address: %v", err)
    }
    
    // Method 1: Try global settings (Android 4.4+)
    err = d.setGlobalProxy(host, port)
    if err == nil {
        return nil
    }
    
    // Method 2: Try Wi-Fi proxy configuration
    return d.setWiFiProxy(host, port)
}

func (d *AndroidDevice) setGlobalProxy(host, port string) error {
    proxySetting := fmt.Sprintf("%s:%s", host, port)
    
    cmd := exec.Command("adb", "-s", d.ID, "shell", "settings", "put", "global", "http_proxy", proxySetting)
    err := cmd.Run()
    if err != nil {
        return fmt.Errorf("failed to set global proxy: %v", err)
    }
    
    // Also set HTTPS proxy
    cmd = exec.Command("adb", "-s", d.ID, "shell", "settings", "put", "global", "https_proxy", proxySetting)
    err = cmd.Run()
    if err != nil {
        return fmt.Errorf("failed to set HTTPS proxy: %v", err)
    }
    
    return nil
}

func (d *AndroidDevice) setWiFiProxy(host, port string) error {
    // Get current Wi-Fi network
    cmd := exec.Command("adb", "-s", d.ID, "shell", "dumpsys", "connectivity")
    output, err := cmd.Output()
    if err != nil {
        return fmt.Errorf("failed to get Wi-Fi info: %v", err)
    }
    
    // This is simplified - in reality you'd parse the output to find the active network
    // For now, we'll try the most common network interface
    
    // Use svc command to set proxy (requires root on newer Android versions)
    proxyCmd := fmt.Sprintf("svc wifi setproxy %s %s", host, port)
    cmd = exec.Command("adb", "-s", d.ID, "shell", proxyCmd)
    err = cmd.Run()
    if err != nil {
        return fmt.Errorf("failed to set Wi-Fi proxy: %v", err)
    }
    
    return nil
}

func (d *AndroidDevice) RemoveProxy() error {
    // Clear global proxy
    cmd := exec.Command("adb", "-s", d.ID, "shell", "settings", "delete", "global", "http_proxy")
    cmd.Run()
    
    cmd = exec.Command("adb", "-s", d.ID, "shell", "settings", "delete", "global", "https_proxy")
    cmd.Run()
    
    // Clear Wi-Fi proxy
    cmd = exec.Command("adb", "-s", d.ID, "shell", "svc", "wifi", "setproxy", "", "")
    cmd.Run()
    
    return nil
}
```

### Step 3: One-Click Certificate Installation (Week 2)

```go
// grx/mobile/cert.go
package mobile

import (
    "fmt"
    "os/exec"
    "path/filepath"
)

func (d *AndroidDevice) InstallCertificate(certPath string) error {
    // Push certificate to device
    remotePath := "/sdcard/Download/grroxy-ca.crt"
    cmd := exec.Command("adb", "-s", d.ID, "push", certPath, remotePath)
    err := cmd.Run()
    if err != nil {
        return fmt.Errorf("failed to push certificate: %v", err)
    }
    
    // Try multiple installation methods
    
    // Method 1: Android 7.0+ - use pm install-ca-certificate
    err = d.installCertificateModern(remotePath)
    if err == nil {
        return d.verifyCertificateInstalled()
    }
    
    // Method 2: Manual installation prompt
    return d.installCertificateManual(remotePath)
}

func (d *AndroidDevice) installCertificateModern(remotePath string) error {
    cmd := exec.Command("adb", "-s", d.ID, "shell", "pm", "install-ca-certificate", remotePath)
    err := cmd.Run()
    if err != nil {
        return fmt.Errorf("modern certificate installation failed: %v", err)
    }
    return nil
}

func (d *AndroidDevice) installCertificateManual(remotePath string) error {
    // Copy to user certificate store and open settings
    userCertPath := "/data/misc/user/0/cacerts-added/grroxy-ca.crt"
    cmd := exec.Command("adb", "-s", d.ID, "shell", "cp", remotePath, userCertPath)
    cmd.Run()
    
    // Open certificate installation screen
    cmd = exec.Command("adb", "-s", d.ID, "shell", "am", "start", "-a", "android.intent.action.VIEW", 
        "-d", "file:///sdcard/Download/grroxy-ca.crt", 
        "-t", "application/x-x509-ca-cert")
    err := cmd.Run()
    if err != nil {
        return fmt.Errorf("failed to open certificate installation: %v", err)
    }
    
    return fmt.Errorf("manual installation required - please complete on device")
}

func (d *AndroidDevice) verifyCertificateInstalled() error {
    cmd := exec.Command("adb", "-s", d.ID, "shell", "ls", "/data/misc/user/0/cacerts-added/")
    output, err := cmd.Output()
    if err != nil {
        return fmt.Errorf("cannot verify certificate: %v", err)
    }
    
    if !contains(string(output), "grroxy-ca") {
        return fmt.Errorf("certificate not found in trust store")
    }
    
    return nil
}

func (d *AndroidDevice) RemoveCertificate() error {
    // Remove from user certificate store
    cmd := exec.Command("adb", "-s", d.ID, "shell", "rm", "-f", "/data/misc/user/0/cacerts-added/grroxy-ca.crt")
    cmd.Run()
    
    // Remove downloaded file
    cmd = exec.Command("adb", "-s", d.ID, "shell", "rm", "-f", "/sdcard/Download/grroxy-ca.crt")
    cmd.Run()
    
    return nil
}
```

### Step 4: iOS Support (Week 2-3)

```go
// grx/mobile/ios.go
package mobile

import (
    "fmt"
    "net"
    "os/exec"
    "path/filepath"
    "text/template"
)

type IOSDevice struct {
    *Device
    configDir string
}

func (d *IOSDevice) ConfigureProxy(proxyAddr string) error {
    host, port, err := net.SplitHostPort(proxyAddr)
    if err != nil {
        return fmt.Errorf("invalid proxy address: %v", err)
    }
    
    // Create mobile configuration profile
    profile, err := d.createProxyProfile(host, port)
    if err != nil {
        return err
    }
    
    // Install profile
    return d.installProfile(profile, "proxy.mobileconfig")
}

func (d *IOSDevice) createProxyProfile(host, port string) ([]byte, error) {
    profileTemplate := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <dict>
            <key>PayloadDescription</key>
            <string>Grroxy Proxy Configuration</string>
            <key>PayloadDisplayName</key>
            <string>Grroxy Proxy</string>
            <key>PayloadIdentifier</key>
            <string>com.grroxy.proxy.{{.DeviceID}}</string>
            <key>PayloadOrganization</key>
            <string>Grroxy</string>
            <key>PayloadType</key>
            <string>com.apple.proxy.managed</string>
            <key>PayloadUUID</key>
            <string>{{.UUID}}</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
            <key>ProxyServer</key>
            <dict>
                <key>ProxyServerPort</key>
                <integer>{{.Port}}</integer>
                <key>ProxyServerURL</key>
                <string>{{.Host}}</string>
                <key>ProxyType</key>
                <string>HTTP</string>
            </dict>
        </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>Grroxy Configuration</string>
    <key>PayloadIdentifier</key>
    <string>com.grroxy.{{.DeviceID}}</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>{{.UUID}}</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>`
    
    tmpl := template.Must(template.New("profile").Parse(profileTemplate))
    
    data := struct {
        DeviceID string
        Host     string
        Port     string
        UUID     string
    }{
        DeviceID: d.ID,
        Host:     host,
        Port:     port,
        UUID:     generateUUID(),
    }
    
    var result strings.Builder
    err := tmpl.Execute(&result, data)
    if err != nil {
        return nil, fmt.Errorf("failed to generate profile: %v", err)
    }
    
    return []byte(result.String()), nil
}

func (d *IOSDevice) installProfile(profile []byte, filename string) error {
    profilePath := filepath.Join(d.configDir, filename)
    err := os.WriteFile(profilePath, profile, 0644)
    if err != nil {
        return fmt.Errorf("failed to save profile: %v", err)
    }
    
    cmd := exec.Command("ideviceinstaller", "-i", profilePath, "-u", d.ID)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to install profile: %v, output: %s", err, string(output))
    }
    
    return nil
}

func (d *IOSDevice) InstallCertificate(certPath string) error {
    // Create certificate profile
    profile, err := d.createCertificateProfile(certPath)
    if err != nil {
        return err
    }
    
    return d.installProfile(profile, "cert.mobileconfig")
}

func (d *IOSDevice) createCertificateProfile(certPath string) ([]byte, error) {
    certData, err := os.ReadFile(certPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read certificate: %v", err)
    }
    
    profileTemplate := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <dict>
            <key>PayloadCertificateFileName</key>
            <string>grroxy-ca.crt</string>
            <key>PayloadContent</key>
            <data>{{.CertData}}</data>
            <key>PayloadDescription</key>
            <string>Grroxy Root Certificate</string>
            <key>PayloadDisplayName</key>
            <string>Grroxy CA</string>
            <key>PayloadIdentifier</key>
            <string>com.grroxy.cert.{{.DeviceID}}</string>
            <key>PayloadType</key>
            <string>com.apple.security.root</string>
            <key>PayloadUUID</key>
            <string>{{.UUID}}</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>Grroxy Certificate</string>
    <key>PayloadIdentifier</key>
    <string>com.grroxy.bundle.{{.DeviceID}}</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>{{.UUID}}</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>`
    
    tmpl := template.Must(template.New("certprofile").Parse(profileTemplate))
    
    data := struct {
        DeviceID string
        CertData string
        UUID     string
    }{
        DeviceID: d.ID,
        CertData: base64.StdEncoding.EncodeToString(certData),
        UUID:     generateUUID(),
    }
    
    var result strings.Builder
    err = tmpl.Execute(&result, data)
    if err != nil {
        return nil, fmt.Errorf("failed to generate certificate profile: %v", err)
    }
    
    return []byte(result.String()), nil
}

func (d *IOSDevice) RemoveProxy() error {
    cmd := exec.Command("ideviceinstaller", "-U", "com.grroxy.proxy."+d.ID, "-u", d.ID)
    cmd.Run()
    return nil
}

func (d *IOSDevice) RemoveCertificate() error {
    cmd := exec.Command("ideviceinstaller", "-U", "com.grroxy.cert."+d.ID, "-u", d.ID)
    cmd.Run()
    return nil
}
```

### Step 5: API Integration (Week 3)

```go
// apps/app/mobile.go
package app

import (
    "encoding/json"
    "net/http"
    
    "github.com/glitchedgitz/grroxy/grx/mobile"
    "github.com/labstack/echo/v5"
)

var mobileDetector *mobile.Detector

func (backend *Backend) InitializeMobileSupport() error {
    mobileDetector = mobile.NewDetector()
    mobileDetector.Start()
    return nil
}

func (backend *Backend) RegisterMobileAPI(e *core.ServeEvent) error {
    // List devices
    e.Router.AddRoute(echo.Route{
        Method: http.MethodGet,
        Path:   "/api/mobile/devices",
        Handler: func(c echo.Context) error {
            devices := mobileDetector.GetDevices()
            return c.JSON(http.StatusOK, map[string]interface{}{
                "devices": devices,
            })
        },
    })
    
    // Configure device proxy
    e.Router.AddRoute(echo.Route{
        Method: http.MethodPost,
        Path:   "/api/mobile/devices/{id}/proxy",
        Handler: func(c echo.Context) error {
            deviceID := c.Param("id")
            
            var req struct {
                ProxyAddr string `json:"proxyAddr"`
            }
            if err := c.Bind(&req); err != nil {
                return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
            }
            
            device := mobileDetector.GetDevice(deviceID)
            if device == nil {
                return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Device not found"})
            }
            
            var err error
            switch device.Type {
            case "android":
                androidDevice := &mobile.AndroidDevice{Device: device}
                err = androidDevice.ConfigureProxy(req.ProxyAddr)
            case "ios":
                iosDevice := &mobile.IOSDevice{Device: device, configDir: backend.Config.ConfigDirectory}
                err = iosDevice.ConfigureProxy(req.ProxyAddr)
            default:
                return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Unsupported device type"})
            }
            
            if err != nil {
                return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
            }
            
            device.ProxyEnabled = true
            return c.JSON(http.StatusOK, map[string]interface{}{"message": "Proxy configured successfully"})
        },
    })
    
    // Install certificate
    e.Router.AddRoute(echo.Route{
        Method: http.MethodPost,
        Path:   "/api/mobile/devices/{id}/certificate",
        Handler: func(c echo.Context) error {
            deviceID := c.Param("id")
            
            device := mobileDetector.GetDevice(deviceID)
            if device == nil {
                return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Device not found"})
            }
            
            certPath := backend.GetCertPath()
            
            var err error
            switch device.Type {
            case "android":
                androidDevice := &mobile.AndroidDevice{Device: device}
                err = androidDevice.InstallCertificate(certPath)
            case "ios":
                iosDevice := &mobile.IOSDevice{Device: device, configDir: backend.Config.ConfigDirectory}
                err = iosDevice.InstallCertificate(certPath)
            default:
                return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Unsupported device type"})
            }
            
            if err != nil {
                return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
            }
            
            device.CertInstalled = true
            return c.JSON(http.StatusOK, map[string]interface{}{"message": "Certificate installed successfully"})
        },
    })
    
    // Reset device
    e.Router.AddRoute(echo.Route{
        Method: http.MethodDelete,
        Path:   "/api/mobile/devices/{id}/reset",
        Handler: func(c echo.Context) error {
            deviceID := c.Param("id")
            
            device := mobileDetector.GetDevice(deviceID)
            if device == nil {
                return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Device not found"})
            }
            
            // Remove proxy and certificate
            switch device.Type {
            case "android":
                androidDevice := &mobile.AndroidDevice{Device: device}
                androidDevice.RemoveProxy()
                androidDevice.RemoveCertificate()
            case "ios":
                iosDevice := &mobile.IOSDevice{Device: device, configDir: backend.Config.ConfigDirectory}
                iosDevice.RemoveProxy()
                iosDevice.RemoveCertificate()
            }
            
            device.ProxyEnabled = false
            device.CertInstalled = false
            return c.JSON(http.StatusOK, map[string]interface{}{"message": "Device reset successfully"})
        },
    })
    
    return nil
}
```

### Step 6: Frontend Integration (Week 3-4)

```javascript
// grx/frontend/src/components/MobileDevices.js
class MobileDevices {
    constructor() {
        this.devices = new Map();
        this.pollInterval = null;
        this.init();
    }
    
    async init() {
        await this.loadDevices();
        this.startPolling();
        this.render();
    }
    
    async loadDevices() {
        try {
            const response = await fetch('/api/mobile/devices');
            const data = await response.json();
            
            this.devices.clear();
            data.devices.forEach(device => {
                this.devices.set(device.id, device);
            });
        } catch (error) {
            console.error('Failed to load mobile devices:', error);
        }
    }
    
    startPolling() {
        this.pollInterval = setInterval(() => {
            this.loadDevices().then(() => this.render());
        }, 3000);
    }
    
    render() {
        const container = document.getElementById('mobile-devices');
        if (!container) return;
        
        container.innerHTML = `
            <div class="mobile-section">
                <h3>Mobile Devices</h3>
                <div class="device-list">
                    ${this.renderDeviceList()}
                </div>
            </div>
        `;
    }
    
    renderDeviceList() {
        if (this.devices.size === 0) {
            return `
                <div class="no-devices">
                    <p>No mobile devices detected</p>
                    <small>Connect a device via USB and enable USB debugging (Android) or trust this computer (iOS)</small>
                </div>
            `;
        }
        
        let html = '';
        this.devices.forEach((device, id) => {
            html += this.renderDevice(device, id);
        });
        
        return html;
    }
    
    renderDevice(device, id) {
        const statusClass = device.connected ? 'connected' : 'disconnected';
        const proxyStatus = device.proxyEnabled ? 'enabled' : 'disabled';
        const certStatus = device.certInstalled ? 'installed' : 'not-installed';
        
        return `
            <div class="device-card ${statusClass}">
                <div class="device-header">
                    <div class="device-info">
                        <h4>${device.name}</h4>
                        <span class="device-type">${device.type.toUpperCase()} ${device.osVersion}</span>
                    </div>
                    <div class="device-status">
                        <span class="status-indicator ${statusClass}" title="Connected"></span>
                        <span class="status-indicator ${proxyStatus}" title="Proxy"></span>
                        <span class="status-indicator ${certStatus}" title="Certificate"></span>
                    </div>
                </div>
                
                <div class="device-actions">
                    ${this.renderDeviceActions(device)}
                </div>
            </div>
        `;
    }
    
    renderDeviceActions(device) {
        if (!device.connected) {
            return '<button disabled>Device Disconnected</button>';
        }
        
        return `
            <button onclick="mobileDevices.configureProxy('${device.id}')" 
                    ${device.proxyEnabled ? 'disabled' : ''}>
                ${device.proxyEnabled ? 'Proxy Configured ✓' : 'Configure Proxy'}
            </button>
            <button onclick="mobileDevices.installCert('${device.id}')"
                    ${device.certInstalled ? 'disabled' : ''}>
                ${device.certInstalled ? 'Certificate Installed ✓' : 'Install Certificate'}
            </button>
            <button onclick="mobileDevices.resetDevice('${device.id}')" 
                    class="danger">
                Reset
            </button>
        `;
    }
    
    async configureProxy(deviceId) {
        const device = this.devices.get(deviceId);
        if (!device) return;
        
        try {
            // Get current proxy address from the first running proxy
            const proxyAddr = await this.getCurrentProxyAddress();
            
            const response = await fetch(`/api/mobile/devices/${deviceId}/proxy`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ proxyAddr })
            });
            
            if (response.ok) {
                device.proxyEnabled = true;
                this.render();
            } else {
                const error = await response.json();
                alert('Failed to configure proxy: ' + error.error);
            }
        } catch (error) {
            alert('Failed to configure proxy: ' + error.message);
        }
    }
    
    async installCert(deviceId) {
        const device = this.devices.get(deviceId);
        if (!device) return;
        
        try {
            const response = await fetch(`/api/mobile/devices/${deviceId}/certificate`, {
                method: 'POST'
            });
            
            if (response.ok) {
                device.certInstalled = true;
                this.render();
            } else {
                const error = await response.json();
                alert('Failed to install certificate: ' + error.error);
            }
        } catch (error) {
            alert('Failed to install certificate: ' + error.message);
        }
    }
    
    async resetDevice(deviceId) {
        if (!confirm('Reset proxy and certificate settings for this device?')) return;
        
        try {
            const response = await fetch(`/api/mobile/devices/${deviceId}/reset`, {
                method: 'DELETE'
            });
            
            if (response.ok) {
                const device = this.devices.get(deviceId);
                if (device) {
                    device.proxyEnabled = false;
                    device.certInstalled = false;
                    this.render();
                }
            } else {
                const error = await response.json();
                alert('Failed to reset device: ' + error.error);
            }
        } catch (error) {
            alert('Failed to reset device: ' + error.message);
        }
    }
    
    async getCurrentProxyAddress() {
        // Get the first running proxy address
        const response = await fetch('/api/proxy/list');
        const data = await response.json();
        
        if (data.proxies && data.proxies.length > 0) {
            return data.proxies[0].listenAddr;
        }
        
        throw new Error('No running proxy found');
    }
}

// Initialize when DOM is ready
let mobileDevices;
document.addEventListener('DOMContentLoaded', () => {
    mobileDevices = new MobileDevices();
});
```

### Step 7: CSS Styling

```css
/* grx/frontend/src/styles/mobile.css */
.mobile-section {
    margin: 20px 0;
    padding: 20px;
    border: 1px solid #ddd;
    border-radius: 8px;
    background: #f9f9f9;
}

.device-card {
    margin: 10px 0;
    padding: 15px;
    border: 1px solid #ccc;
    border-radius: 6px;
    background: white;
    transition: all 0.3s ease;
}

.device-card.connected {
    border-color: #4CAF50;
    box-shadow: 0 2px 4px rgba(76, 175, 80, 0.2);
}

.device-card.disconnected {
    opacity: 0.6;
    border-color: #f44336;
}

.device-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
}

.device-info h4 {
    margin: 0;
    font-size: 16px;
}

.device-type {
    font-size: 12px;
    color: #666;
    background: #e0e0e0;
    padding: 2px 6px;
    border-radius: 3px;
}

.device-status {
    display: flex;
    gap: 8px;
}

.status-indicator {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    cursor: help;
}

.status-indicator.connected { background: #4CAF50; }
.status-indicator.disconnected { background: #f44336; }
.status-indicator.enabled { background: #2196F3; }
.status-indicator.disabled { background: #9E9E9E; }
.status-indicator.installed { background: #FF9800; }
.status-indicator.not-installed { background: #9E9E9E; }

.device-actions {
    display: flex;
    gap: 10px;
    margin-top: 10px;
}

.device-actions button {
    padding: 6px 12px;
    border: 1px solid #ddd;
    border-radius: 4px;
    background: white;
    cursor: pointer;
    font-size: 12px;
    transition: all 0.2s ease;
}

.device-actions button:hover:not(:disabled) {
    background: #f0f0f0;
    border-color: #2196F3;
}

.device-actions button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

.device-actions button.danger {
    color: #f44336;
    border-color: #f44336;
}

.device-actions button.danger:hover:not(:disabled) {
    background: #ffebee;
}

.no-devices {
    text-align: center;
    padding: 40px;
    color: #666;
}

.no-devices p {
    margin: 0 0 10px 0;
    font-size: 16px;
}

.no-devices small {
    font-size: 12px;
    color: #999;
}
```

## Implementation Timeline

**Week 1: Core Detection + Android Proxy**
- Device detection system
- Android proxy configuration
- Basic API endpoints

**Week 2: Certificate Installation**
- Android certificate installation
- iOS detection and proxy
- Certificate profile generation

**Week 3: API Integration**
- Complete API endpoints
- iOS certificate installation
- Error handling and validation

**Week 4: UI Integration**
- Frontend components
- Device management interface
- Real-time status updates

## Quick Start

1. **Install dependencies:**
   ```bash
   # Android
   brew install android-platform-tools  # macOS
   # Or download from Android SDK
   
   # iOS
   brew install libimobiledevice ios-deploy  # macOS
   ```

2. **Add mobile initialization:**
   ```go
   // In main.go
   backend.InitializeMobileSupport()
   backend.RegisterMobileAPI(e)
   ```

3. **Add mobile tab to frontend:**
   ```html
   <div id="mobile-devices"></div>
   <script src="/js/mobile.js"></script>
   ```

This implementation provides HTTP Toolkit-style mobile support with automatic device detection, one-click proxy configuration, and seamless certificate installation.
