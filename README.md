# grroxy

A cyber security toolkit blending manual testing with AI Agents.

[![Website](https://img.shields.io/badge/Website-grroxy.com-blue)](https://grroxy.com) [![Discord](https://img.shields.io/badge/Discord-Join-5865F2?logo=discord&logoColor=white)](https://discord.gg/K8pGK6XatC)

<img width="1200" height="747" alt="image" src="https://github.com/user-attachments/assets/cf1d8388-f41e-47b1-bade-2206a1f561f8" />

### Why?

The idea is to have a toolkit that prioritises how hackers work.

## Installation

### Desktop App

Download the latest release for your platform from [Releases](https://github.com/glitchedgitz/grroxy/releases):

```bash
# Note: On macOS, the app may show a prompt saying it's not signed. I've applied for an Apple Developer ID — it will take some time.

# current workaround: run the command and restart the app

xattr -cr /Applications/Grroxy.app
```

### Terminal

If you prefer using grroxy without the desktop app (it's no fun)

```bash
go install github.com/glitchedgitz/grroxy/cmd/grroxy@latest
go install github.com/glitchedgitz/grroxy/cmd/grroxy-app@latest
go install github.com/glitchedgitz/grroxy/cmd/grroxy-tool@latest
go install -v github.com/glitchedgitz/cook/v2/cmd/cook@latest
```
```bash
grroxy start
# http://127.0.0.1:8090
```

---


# Contributing

You can help by suggesting new features or joining active discussions on [discord](https://discord.gg/J4VPhZqnUu). 

Or by contirbuting to the code. A separate developer interface is available at `grx/dev/` for contributors to test and build backend features ([#36](https://github.com/glitchedgitz/grroxy/issues/36)).  

Use this interface to build and test backend features — the UI for accepted contributions will be added to the main frontend in subsequent releases.

```bash
cd grx/dev
npm install
npm run dev

# use sudo if you have too
```

But please refrain from creating PRs for new features without first discussing the implementation details.

### AI Contributions
Use of AI is recommended.

### Frontend directory is private ([#36](https://github.com/glitchedgitz/grroxy/issues/36))
I have kept the `frontend` directory private for now. Contributing directly to the frontend wouldn't be ideal, as AI lacks good design sense and we want to maintain product quality, experience, and the direction we're moving forward in — plus the frontend is messy.


