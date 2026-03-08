# Implementing Android and iOS Support in Grroxy (Like HTTP Toolkit)

## Overview

This document provides a comprehensive guide for adding Android and iOS mobile device support to Grroxy, similar to how HTTP Toolkit handles mobile proxying. The implementation involves creating a mobile device management system that can configure proxies, install certificates, and monitor traffic from mobile devices.

## Current State Analysis

Grroxy currently supports:
- Chrome browser with DevTools integration
- Firefox browser
- Safari browser  
- Terminal/manual proxy configuration
- Certificate generation and management
- HTTP/HTTPS proxy interception

The existing proxy infrastructure is solid with:
- `ProxyManager` for managing multiple proxy instances
- `RawProxyWrapper` for HTTP/HTTPS interception
- Certificate management in `grx/browser/cert.go`
- Browser launching system in `grx/browser/`

## Architecture for Mobile Support

### 1. Mobile Device Manager

Create a new mobile device management system:

```
grx/mobile/
├── device.go          # Device abstraction and management
├── android/
│   ├── adb.go         # Android Debug Bridge integration
│   ├── proxy.go       # Android proxy configuration
│   └── cert.go        # Certificate installation
├── ios/
│   ├── ios.go         # iOS device management
│   ├── proxy.go       # iOS proxy configuration  
│   └── cert.go        # Certificate installation
└── simulator/
    ├── android_sim.go # Android emulator support
    └── ios_sim.go     # iOS simulator support
```

### 2. Device Types

Support multiple device categories:
- **Physical Android Devices** via ADB
- **Physical iOS Devices** via iOS configuration
- **Android Emulators** via Android Studio
- **iOS Simulators** via Xcode

## Implementation Plan

### Phase 1: Core Mobile Infrastructure

#### 1.1 Device Abstraction Layer

```go
// grx/mobile/device.go
type Device interface {
    ID() string
    Name() string
    Type() DeviceType
    OSVersion() string
    IsConnected() bool
    ConfigureProxy(proxyAddr string) error
    InstallCertificate(certPath string) error
    RemoveProxy() error
    RemoveCertificate() error
    GetStatus() DeviceStatus
}

type DeviceType string
const (
    DeviceTypeAndroidPhysical DeviceType = "android_physical"
    DeviceTypeAndroidEmulator DeviceType = "android_emulator"
    DeviceTypeIOSPhysical     DeviceType = "ios_physical"
    DeviceTypeIOSSimulator    DeviceType = "ios_simulator"
)

type DeviceStatus struct {
    Connected       bool   `json:"connected"`
    ProxyConfigured bool   `json:"proxyConfigured"`
    CertInstalled   bool   `json:"certInstalled"`
    LastSeen        time.Time `json:"lastSeen"`
    Error           string `json:"error,omitempty"`
}
```

#### 1.2 Mobile Device Manager

```go
// grx/mobile/manager.go
type MobileManager struct {
    devices    map[string]Device
    mu         sync.RWMutex
    proxyMgr   *ProxyManager
    certPath   string
    discovery  time.Ticker
}

func (mm *MobileManager) StartDiscovery()
func (mm *MobileManager) StopDiscovery()
func (mm *MobileManager) GetDevices() []Device
func (mm *MobileManager) GetDevice(id string) Device
func (mm *MobileManager) ConfigureDevice(deviceID, proxyID string) error
func (mm *MobileManager) ResetDevice(deviceID string) error
```

### Phase 2: Android Implementation

#### 2.1 ADB Integration

```go
// grx/mobile/android/adb.go
type ADBManager struct {
    adbPath string
    devices map[string]*AndroidDevice
}

func (am *ADBManager) DetectDevices() ([]*AndroidDevice, error)
func (am *ADBManager) ExecuteCommand(deviceID, cmd string) (string, error)
func (am *ADBManager) PushFile(deviceID, localPath, remotePath string) error
func (am *ADBManager) InstallAPK(deviceID, apkPath string) error
func (am *ADBManager) GetDeviceProperty(deviceID, prop string) (string, error)
```

#### 2.2 Android Proxy Configuration

```go
// grx/mobile/android/proxy.go
func (ad *AndroidDevice) ConfigureProxy(proxyAddr string) error {
    // Set global HTTP proxy
    cmd := fmt.Sprintf("settings put global http_proxy %s", proxyAddr)
    _, err := ad.adb.ExecuteCommand(ad.ID(), cmd)
    if err != nil {
        return err
    }
    
    // Configure Wi-Fi proxy for each network
    networks, err := ad.getWifiNetworks()
    if err != nil {
        return err
    }
    
    for _, network := range networks {
        err := ad.configureWifiProxy(network, proxyAddr)
        if err != nil {
            log.Printf("Failed to configure proxy for network %s: %v", network, err)
        }
    }
    
    return nil
}

func (ad *AndroidDevice) configureWifiProxy(networkID, proxyAddr string) error {
    host, port, err := net.SplitHostPort(proxyAddr)
    if err != nil {
        return err
    }
    
    cmd := fmt.Sprintf("svc wifi setproxy %s %s %s", networkID, host, port)
    _, err = ad.adb.ExecuteCommand(ad.ID(), cmd)
    return err
}
```

#### 2.3 Android Certificate Installation

```go
// grx/mobile/android/cert.go
func (ad *AndroidDevice) InstallCertificate(certPath string) error {
    // Push certificate to device
    remotePath := "/sdcard/Download/grroxy-ca.crt"
    err := ad.adb.PushFile(ad.ID(), certPath, remotePath)
    if err != nil {
        return err
    }
    
    // Install certificate using security commands
    cmd := fmt.Sprintf("pm install-ca-certificate %s", remotePath)
    _, err = ad.adb.ExecuteCommand(ad.ID(), cmd)
    if err != nil {
        // Fallback for older Android versions
        return ad.installCertificateManual(certPath)
    }
    
    return ad.verifyCertificateInstalled()
}

func (ad *AndroidDevice) installCertificateManual(certPath string) error {
    // For devices that don't support pm install-ca-certificate
    // Copy to user certificate store and trigger installation
    remotePath := "/data/local/tmp/grroxy-ca.crt"
    err := ad.adb.PushFile(ad.ID(), certPath, remotePath)
    if err != nil {
        return err
    }
    
    cmd := fmt.Sprintf("security import-certificate %s", remotePath)
    _, err = ad.adb.ExecuteCommand(ad.ID(), cmd)
    return err
}
```

### Phase 3: iOS Implementation

#### 3.1 iOS Device Detection

```go
// grx/mobile/ios/ios.go
type IOSManager struct {
    devices map[string]*IOSDevice
    cfgDir  string
}

func (im *IOSManager) DetectDevices() ([]*IOSDevice, error) {
    // Use ios-deploy or libimobiledevice
    cmd := exec.Command("idevice_id", "-l")
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("failed to detect iOS devices: %v", err)
    }
    
    var devices []*IOSDevice
    for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
        if line != "" {
            device := &IOSDevice{
                id:       line,
                manager:  im,
                isActive: true,
            }
            devices = append(devices, device)
        }
    }
    
    return devices, nil
}
```

#### 3.2 iOS Proxy Configuration

```go
// grx/mobile/ios/proxy.go
func (id *IOSDevice) ConfigureProxy(proxyAddr string) error {
    // Create mobile configuration profile
    profile := id.createProxyProfile(proxyAddr)
    
    // Install profile using ideviceinstaller
    profilePath := filepath.Join(id.manager.cfgDir, "proxy.mobileconfig")
    err := os.WriteFile(profilePath, profile, 0644)
    if err != nil {
        return err
    }
    
    cmd := exec.Command("ideviceinstaller", "-i", profilePath, "-u", id.id)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to install proxy profile: %v, output: %s", err, string(output))
    }
    
    return id.verifyProxyConfigured()
}

func (id *IOSDevice) createProxyProfile(proxyAddr string) []byte {
    host, port, _ := net.SplitHostPort(proxyAddr)
    
    profile := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
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
            <string>com.grroxy.proxy.%s</string>
            <key>PayloadOrganization</key>
            <string>Grroxy</string>
            <key>PayloadType</key>
            <string>com.apple.proxy.managed</string>
            <key>PayloadUUID</key>
            <string>%s</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
            <key>ProxyServer</key>
            <dict>
                <key>ProxyServerPort</key>
                <integer>%s</integer>
                <key>ProxyServerURL</key>
                <string>%s</string>
                <key>ProxyType</key>
                <string>HTTP</string>
            </dict>
        </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>Grroxy Configuration</string>
    <key>PayloadIdentifier</key>
    <string>com.grroxy.%s</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>%s</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>`, id.id, generateUUID(), port, host, id.id, generateUUID())
    
    return []byte(profile)
}
```

#### 3.3 iOS Certificate Installation

```go
// grx/mobile/ios/cert.go
func (id *IOSDevice) InstallCertificate(certPath string) error {
    // Create certificate profile
    profile := id.createCertificateProfile(certPath)
    
    // Install profile
    profilePath := filepath.Join(id.manager.cfgDir, "cert.mobileconfig")
    err := os.WriteFile(profilePath, profile, 0644)
    if err != nil {
        return err
    }
    
    cmd := exec.Command("ideviceinstaller", "-i", profilePath, "-u", id.id)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to install certificate profile: %v, output: %s", err, string(output))
    }
    
    return id.verifyCertificateInstalled()
}
```

### Phase 4: Simulator Support

#### 4.1 Android Emulator

```go
// grx/mobile/simulator/android_sim.go
type AndroidSimulator struct {
    name     string
    port     int
    adbPath  string
    running  bool
}

func (as *AndroidSimulator) ConfigureProxy(proxyAddr string) error {
    // Use emulator console commands
    conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", as.port))
    if err != nil {
        return err
    }
    defer conn.Close()
    
    // Set proxy via emulator console
    cmd := fmt.Sprintf("redir add tcp:%s tcp:%s\n", proxyAddr, proxyAddr)
    _, err = conn.Write([]byte(cmd))
    return err
}
```

#### 4.2 iOS Simulator

```go
// grx/mobile/simulator/ios_sim.go
type IOSSimulator struct {
    deviceID string
    name     string
    running  bool
}

func (is *IOSSimulator) ConfigureProxy(proxyAddr string) error {
    // Use simctl to configure proxy
    host, port, _ := net.SplitHostPort(proxyAddr)
    
    cmd := exec.Command("xcrun", "simctl", "spawn", is.deviceID, 
        "networksetup", "-setwebproxy", "Wi-Fi", host, port)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to set iOS simulator proxy: %v, output: %s", err, string(output))
    }
    
    return nil
}
```

### Phase 5: Integration with Existing Proxy System

#### 5.1 Extend ProxyManager

```go
// apps/app/proxy.go - Add to existing ProxyManager
type ProxyManager struct {
    // ... existing fields
    mobileMgr *mobile.MobileManager
}

func (pm *ProxyManager) InitializeMobileSupport(certPath string) error {
    pm.mobileMgr = mobile.NewMobileManager(pm, certPath)
    return pm.mobileMgr.StartDiscovery()
}

func (pm *ProxyManager) GetMobileDevices() []mobile.Device {
    if pm.mobileMgr == nil {
        return nil
    }
    return pm.mobileMgr.GetDevices()
}

func (pm *ProxyManager) ConfigureMobileDevice(deviceID, proxyID string) error {
    if pm.mobileMgr == nil {
        return fmt.Errorf("mobile support not initialized")
    }
    return pm.mobileMgr.ConfigureDevice(deviceID, proxyID)
}
```

#### 5.2 API Endpoints

```go
// apps/app/proxy.go - Add new API endpoints
func (backend *Backend) StartMobileSupport(e *core.ServeEvent) error {
    e.Router.AddRoute(echo.Route{
        Method: http.MethodGet,
        Path:   "/api/mobile/devices",
        Handler: func(c echo.Context) error {
            devices := ProxyMgr.GetMobileDevices()
            return c.JSON(http.StatusOK, map[string]interface{}{
                "devices": devices,
            })
        },
    })
    
    e.Router.AddRoute(echo.Route{
        Method: http.MethodPost,
        Path:   "/api/mobile/configure",
        Handler: func(c echo.Context) error {
            var req struct {
                DeviceID string `json:"deviceId"`
                ProxyID  string `json:"proxyId"`
            }
            if err := c.Bind(&req); err != nil {
                return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
            }
            
            err := ProxyMgr.ConfigureMobileDevice(req.DeviceID, req.ProxyID)
            if err != nil {
                return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
            }
            
            return c.JSON(http.StatusOK, map[string]interface{}{"message": "Device configured"})
        },
    })
    
    return nil
}
```

### Phase 6: UI Integration

#### 6.1 Frontend Components

```javascript
// grx/frontend/src/components/MobileDeviceManager.js
class MobileDeviceManager {
    constructor() {
        this.devices = new Map();
        this.pollInterval = null;
    }
    
    async startDiscovery() {
        const response = await fetch('/api/mobile/devices');
        const data = await response.json();
        this.updateDevices(data.devices);
        
        this.pollInterval = setInterval(() => {
            this.refreshDevices();
        }, 5000);
    }
    
    async configureDevice(deviceId, proxyId) {
        const response = await fetch('/api/mobile/configure', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ deviceId, proxyId })
        });
        
        if (!response.ok) {
            throw new Error('Failed to configure device');
        }
        
        await this.refreshDevices();
    }
    
    renderDeviceList() {
        const container = document.getElementById('mobile-devices');
        container.innerHTML = '';
        
        this.devices.forEach((device, id) => {
            const deviceEl = this.createDeviceElement(device, id);
            container.appendChild(deviceEl);
        });
    }
}
```

#### 6.2 Device Status Indicators

```javascript
// Device status component
function createDeviceStatus(device) {
    const status = document.createElement('div');
    status.className = 'device-status';
    
    const indicators = {
        connected: device.connected ? '🟢' : '🔴',
        proxy: device.proxyConfigured ? '🔧' : '⚠️',
        cert: device.certInstalled ? '🔒' : '🔓'
    };
    
    status.innerHTML = `
        <span class="indicator" title="Connected">${indicators.connected}</span>
        <span class="indicator" title="Proxy Configured">${indicators.proxy}</span>
        <span class="indicator" title="Certificate Installed">${indicators.cert}</span>
    `;
    
    return status;
}
```

## Dependencies and Requirements

### System Dependencies

**For Android Support:**
- Android SDK Platform Tools (ADB)
- Android Studio (for emulators)
- Java Runtime Environment

**For iOS Support:**
- Xcode (for simulators)
- libimobiledevice (for physical devices)
- ios-deploy (alternative to libimobiledevice)

### Go Dependencies

```go
// Add to go.mod
require (
    github.com/google/wire v0.5.0
    github.com/spf13/viper v1.16.0
    golang.org/x/mobile v0.0.0-20231102165836-aa491e761e89
)
```

### Frontend Dependencies

```json
// package.json additions
{
  "dependencies": {
    "react-device-detect": "^2.2.3",
    "react-dropzone": "^14.2.3"
  }
}
```

## Configuration

### Environment Variables

```bash
# Android
ANDROID_HOME=/path/to/android/sdk
ADB_PATH=/path/to/adb

# iOS  
XCODE_PATH=/Applications/Xcode.app
IDEVICE_PATH=/usr/local/bin

# Mobile
MOBILE_CERT_PATH=/path/to/mobile/certs
MOBILE_CONFIG_PATH=/path/to/mobile/configs
```

### Configuration File

```yaml
# config.yaml
mobile:
  enabled: true
  auto_discovery: true
  discovery_interval: 5s
  
  android:
    adb_path: "/usr/local/bin/adb"
    emulator_port_range: "5554-5585"
    
  ios:
    idevice_path: "/usr/local/bin/idevice_id"
    simulator_timeout: 30s
    
  certificates:
    auto_install: true
    trust_store: "system"
```

## Security Considerations

### Certificate Management
- Generate unique certificates per device
- Implement certificate rotation
- Secure certificate storage
- Proper certificate cleanup on device disconnect

### Network Security
- Validate proxy configurations
- Prevent proxy redirection attacks
- Secure ADB/iOS connections
- Network isolation for testing

### Privacy Protection
- Clear device data on disconnect
- Secure storage of device information
- User consent for device access
- Audit logging of all operations

## Testing Strategy

### Unit Tests
```go
// grx/mobile/manager_test.go
func TestMobileManager_DeviceDiscovery(t *testing.T) {
    manager := NewMobileManager(nil, "")
    
    devices := manager.GetDevices()
    assert.NotNil(t, devices)
}

func TestAndroidDevice_ConfigureProxy(t *testing.T) {
    device := &AndroidDevice{mock: true}
    err := device.ConfigureProxy("127.0.0.1:8080")
    assert.NoError(t, err)
}
```

### Integration Tests
```go
// grx/mobile/integration_test.go
func TestFullMobileWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Test device discovery
    // Test proxy configuration  
    // Test certificate installation
    // Test traffic interception
}
```

### End-to-End Tests
```javascript
// frontend/test/mobile.e2e.js
describe('Mobile Device Management', () => {
    it('should discover devices', async () => {
        await page.goto('/proxy');
        await page.click('[data-testid="mobile-tab"]');
        await expect(page.locator('[data-testid="device-list"]')).toBeVisible();
    });
    
    it('should configure device proxy', async () => {
        // Test device configuration flow
    });
});
```

## Deployment Considerations

### Binary Distribution
- Bundle required tools (ADB, idevice tools)
- Cross-platform compilation
- Dependency management
- Installation scripts

### Docker Support
```dockerfile
# Dockerfile.mobile
FROM golang:1.21-alpine AS builder

# Install mobile dependencies
RUN apk add --no-cache \
    android-tools \
    libimobiledevice \
    usbmuxd

# Copy and build
COPY . /app
WORKDIR /app
RUN go build -o grroxy-mobile ./cmd/grroxy

FROM alpine:latest
RUN apk add --no-cache android-tools libimobiledevice
COPY --from=builder /app/grroxy-mobile /usr/local/bin/
```

### Documentation
- User guide for mobile setup
- Troubleshooting guide
- Platform-specific instructions
- Video tutorials

## Migration Path

### Phase 1 (Weeks 1-2): Core Infrastructure
- Implement device abstraction
- Create mobile manager
- Add basic ADB support

### Phase 2 (Weeks 3-4): Android Support
- Complete ADB integration
- Implement proxy configuration
- Add certificate installation

### Phase 3 (Weeks 5-6): iOS Support
- Implement iOS device detection
- Add proxy configuration
- Certificate installation

### Phase 4 (Weeks 7-8): Simulators
- Android emulator support
- iOS simulator support
- Integration testing

### Phase 5 (Weeks 9-10): UI Integration
- Frontend components
- API endpoints
- User experience polish

### Phase 6 (Weeks 11-12): Testing & Documentation
- Comprehensive testing
- Documentation
- Performance optimization

## Success Metrics

### Functional Metrics
- Device discovery reliability > 95%
- Proxy configuration success rate > 90%
- Certificate installation success rate > 85%
- End-to-end traffic interception success rate > 80%

### Performance Metrics
- Device discovery time < 5 seconds
- Proxy configuration time < 10 seconds
- Certificate installation time < 15 seconds
- UI response time < 2 seconds

### User Experience Metrics
- Setup completion rate > 70%
- User satisfaction score > 4.0/5.0
- Support ticket reduction > 30%

## Conclusion

This implementation provides comprehensive Android and iOS support for Grroxy, matching HTTP Toolkit's capabilities while leveraging Grroxy's existing proxy infrastructure. The modular design allows for incremental development and testing, with clear separation of concerns between device management, proxy configuration, and certificate handling.

The solution addresses key challenges:
- **Cross-platform compatibility** through device abstraction
- **Certificate management** with automated installation
- **User experience** with intuitive UI components
- **Security** with proper validation and cleanup
- **Maintainability** through modular architecture

By following this implementation plan, Grroxy can become a comprehensive proxy solution supporting both desktop and mobile platforms, significantly expanding its utility for developers and security professionals.
