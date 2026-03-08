---
title: Feature Request: Support for additional applications and platforms for traffic interception
labels: ["enhancement", "feature-request"]
---

## Feature Request: Support for additional applications and platforms for traffic interception

### Current supported interception methods

Based on the current UI, the following interception methods are available:

**Core Methods (No Installation Required):**
- Google Chrome
- Terminal

**Manual Configuration Required:**
- Mozilla Firefox
- Manual (custom proxy configuration)

### Requested additional support

I'd like to request support for additional applications and platforms that are not currently covered. This would greatly enhance the utility of the tool for a wider range of development and testing scenarios.

**Suggested additions:**

**Enhanced Browser Support:**
- Global Chrome (main profile interception)
- Brave
- Edge
- Safari (macOS)
- Opera
- Arc Browser
- Vivaldi
- Other Chromium-based browsers

**Development Environments:**
- Docker Containers
- JVM processes (Java, Kotlin, Clojure)
- Fresh Terminal with enhanced features
- Existing Terminal interception
- Electron Applications
- Node.js processes
- Python applications
- .NET applications
- Go applications
- Ruby applications
- PHP applications

**Mobile Platforms:**
- Android Device via QR code
- Android App via Frida
- iOS via Manual Setup
- iOS App via Frida
- Automatic iOS Device Setup

**Virtualization & Network:**
- VirtualBox VMs
- VMware VMs
- Network-wide interception
- A Device on Your Network
- Everything (intercept all HTTP traffic on machine)

**Desktop Applications:**
- Native macOS applications
- Windows applications
- Linux applications
- Communication apps (Slack, Discord, Teams)
- Development tools (IDE integrations)

### Use cases

Adding support for these additional platforms would enable:
- Mobile app developers to test native app behavior more thoroughly
- Web developers to test across all major browsers
- Desktop application developers to debug network communications
- Network administrators to monitor traffic across entire networks
- Security researchers to analyze traffic from various sources

### Implementation considerations

Some of these may require:
- Platform-specific hooks and APIs
- Certificate installation procedures
- Permission handling
- Performance optimizations for high-traffic scenarios

Would it be possible to prioritize some of these based on community demand or technical feasibility? Any updates on the development roadmap for expanding platform support would be greatly appreciated.
