# Browser Interception Implementation Plan

## Current Architecture Analysis

### Existing Implementations
1. **Chrome** (`grx/browser/chrome.go`)
   - Uses `--proxy-server=` flag for proxy configuration
   - Uses `--ignore-certificate-errors-spki-list=` for certificate trust
   - Creates isolated profile directory with `--user-data-dir=`
   - Cross-platform: macOS, Linux, Windows

2. **Firefox** (`grx/browser/firefox.go`)
   - Creates custom profile directory
   - Uses prefs.js/user.js for proxy configuration
   - Sets `network.proxy.type=1` with HTTP/HTTPS proxy settings
   - Cross-platform with platform-specific executable paths

3. **Safari** (`grx/browser/safari.go`)
   - macOS only (Safari is macOS-only browser)
   - Opens Safari.app but requires manual system proxy configuration
   - Copies certificate for manual installation

4. **Terminal** (`grx/browser/terminal.go`)
   - Sets HTTP_PROXY, HTTPS_PROXY, SSL_CERT_FILE environment variables
   - Uses AppleScript on macOS (Terminal.app/iTerm2)
   - Uses various terminal emulators on Linux (gnome-terminal, konsole, etc.)
   - Uses PowerShell on Windows

### How Adding a New Browser Works
1. Add case in `grx/browser/browser.go` `LaunchBrowser()` switch statement
2. Create `launch<BrowserName>()` function in new file or existing file
3. Function signature: `(proxyAddress, customCertPath, profileDir string) (*exec.Cmd, error)`
4. API automatically accepts new browser type via `/api/proxy/start` `browser` field

---

## Implementation Groups

### Group 1: EASIEST - Chromium-Based Browsers (Week 1)
**Difficulty: Easy** - Reuse Chrome implementation pattern
**Browsers:** Brave, Edge, Vivaldi, Opera

All Chromium-based browsers use the same command-line flags as Chrome:
- `--proxy-server=` for proxy
- `--ignore-certificate-errors-spki-list=` for certificate trust
- `--user-data-dir=` for isolated profile

**Platform Paths:**

**Brave:**
- macOS: `/Applications/Brave Browser.app/Contents/MacOS/Brave Browser`
- Linux: `brave` (in PATH)
- Windows: `%ProgramFiles%\BraveSoftware\Brave-Browser\Application\brave.exe`

**Edge:**
- macOS: `/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge`
- Linux: `microsoft-edge` (in PATH)
- Windows: `%ProgramFiles(x86)%\Microsoft\Edge\Application\msedge.exe`

**Vivaldi:**
- macOS: `/Applications/Vivaldi.app/Contents/MacOS/Vivaldi`
- Linux: `vivaldi` (in PATH)
- Windows: `%LocalAppData%\Vivaldi\Application\vivaldi.exe`

**Opera:**
- macOS: `/Applications/Opera.app/Contents/MacOS/Opera`
- Linux: `opera` (in PATH)
- Windows: `%UserProfile%\AppData\Local\Programs\Opera\opera.exe`

**Branch:** `feature/chromium-browsers`

---

### Group 2: MEDIUM - Terminal Variants (Week 2)
**Difficulty: Medium** - Extend Terminal implementation

**Fresh Terminal** - Current implementation
- Opens new terminal window with proxy env vars

**Existing Terminal** - New implementation needed
- Inject proxy settings into already-running terminal
- Platform-specific: AppleScript for macOS, shell integration for Linux/Windows

**Branch:** `feature/terminal-variants`

---

### Group 3: HARD - Development Environments (Week 3-4)
**Difficulty: Hard** - Requires additional tooling

**Docker Container:**
- Requires Docker to be installed
- Create container with proxy environment variables
- Mount certificate into container
- Support for different container runtimes (docker, podman)

**JVM (Java/Kotlin/Clojure):**
- Set JVM system properties: `-Dhttp.proxyHost`, `-Dhttp.proxyPort`
- Set truststore with certificate: `-Djavax.net.ssl.trustStore`
- Launch with `java` command with proxy arguments

**Node.js:**
- Set `NODE_EXTRA_CA_CERTS` environment variable
- Use `global-agent` or similar for proxy

**Electron Applications:**
- Similar to Chrome but for packaged Electron apps
- Requires finding Electron app executable

**Branch:** `feature/dev-environments`

---

### Group 4: VERY HARD - System-Level Integration (Week 5+)
**Difficulty: Very Hard** - Requires system configuration changes

**Global Chrome:**
- Intercept main Chrome profile (not isolated)
- Requires modifying existing Chrome installation
- Risk: affects user's normal browsing

**Network-wide interception:**
- Set system-wide proxy settings
- macOS: `networksetup` command
- Linux: gsettings/dconf or environment variables
- Windows: registry modifications

**VirtualBox/VMware VMs:**
- Configure VM network to use host proxy
- Requires VM guest additions or manual configuration

**Mobile (Android/iOS):**
- Android: ADB integration, certificate injection
- iOS: Usbmuxd, certificate trust settings
- Requires physical device or emulator setup

**Branch:** `feature/system-integration`

---

## Cross-Platform Testing Matrix

| Browser | macOS | Linux | Windows | Notes |
|---------|-------|-------|---------|-------|
| Chrome | ✓ | ✓ | ✓ | Current |
| Firefox | ✓ | ✓ | ✓ | Current |
| Safari | ✓ | N/A | N/A | macOS only |
| Terminal | ✓ | ✓ | ✓ | Current |
| Brave | ✓ | ✓ | ✓ | Group 1 |
| Edge | ✓ | ✓ | ✓ | Group 1 |
| Vivaldi | ✓ | ✓ | ✓ | Group 1 |
| Opera | ✓ | ✓ | ✓ | Group 1 |

---

## Implementation Checklist

### Group 1: Chromium Browsers
- [ ] Create branch `feature/chromium-browsers`
- [ ] Implement `launchBrave()` in `grx/browser/brave.go`
- [ ] Implement `launchEdge()` in `grx/browser/edge.go`
- [ ] Implement `launchVivaldi()` in `grx/browser/vivaldi.go`
- [ ] Implement `launchOpera()` in `grx/browser/opera.go`
- [ ] Add cases to `browser.go` LaunchBrowser()
- [ ] Test on macOS
- [ ] Test on Linux
- [ ] Test on Windows
- [ ] Update API documentation
- [ ] Merge to main

### Group 2: Terminal Variants
- [ ] Create branch `feature/terminal-variants`
- [ ] Implement `launchExistingTerminal()`
- [ ] Test on macOS (Terminal.app and iTerm2)
- [ ] Test on Linux (GNOME Terminal, Konsole)
- [ ] Test on Windows (PowerShell, CMD)
- [ ] Update API documentation
- [ ] Merge to main

---

## Risk Assessment

**Low Risk (Group 1):**
- Chromium browsers use exact same logic as Chrome
- Simple executable path detection
- Well-tested Chrome code can be reused

**Medium Risk (Group 2):**
- Terminal injection may not work on all terminal types
- Existing terminal detection is platform-specific

**High Risk (Group 3+):**
- Docker requires external dependency
- JVM requires Java installation detection
- System-level changes can affect user's machine

---

## Recommended Order
1. **Start with Group 1** (Chromium browsers) - immediate value, low risk
2. **Group 2** (Terminal variants) - medium complexity
3. **Group 3** (Dev environments) - requires more testing
4. **Group 4** (System integration) - requires careful implementation
