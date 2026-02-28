// This file is the entry point for the Electron application.

const { app, BrowserWindow, ipcMain, nativeImage } = require('electron')
const { spawn } = require('child_process')
const path = require('path')

let mainWindow = null
let grroxyProcess = null

function startGrroxy() {
    grroxyProcess = spawn('grroxy', ['start'], {
        stdio: 'pipe',
    })

    grroxyProcess.stdout.on('data', (data) => {
        console.log(`[grroxy] ${data.toString().trimEnd()}`)
    })

    grroxyProcess.stderr.on('data', (data) => {
        console.error(`[grroxy] ${data.toString().trimEnd()}`)
    })

    grroxyProcess.on('error', (err) => {
        console.error(`[grroxy] Failed to start: ${err.message}`)
        grroxyProcess = null
    })

    grroxyProcess.on('close', (code) => {
        console.log(`[grroxy] Process exited with code ${code}`)
        grroxyProcess = null
    })
}

function stopGrroxy() {
    if (grroxyProcess) {
        grroxyProcess.kill()
        grroxyProcess = null
    }
}

function createWindow() {
    const iconPath = path.resolve(__dirname, "icons", "grroxy.png")

    // Windows-specific: use frameless window for custom titlebar
    const isWindows = process.platform === 'win32';

    mainWindow = new BrowserWindow({
        width: 1080,
        height: 720,
        // fullscreen: false,
        frame: !isWindows,                    // frameless on Windows for custom titlebar
        autoHideMenuBar: true,                // hide the menu bar

        icon: iconPath,

        /* ------------- title-bar flags ------------- */
        titleBarStyle: isWindows ? undefined : 'hiddenInset',  // macOS only
        // transparent: true,
        title: 'Grroxy',

        /* ------------- transparent overlay -------- */
        titleBarOverlay: isWindows ? undefined : {  // macOS only
            color: '#00000000',                // fully transparent (ARGB = 0×00)
            symbolColor: '#FFFFFF',            // traffic-light glyph colour
        },

        vibrancy: isWindows ? undefined : 'under-window',  // macOS only

        webPreferences: {
            preload: path.join(__dirname, 'preload.js'),
            contextIsolation: true,
            nodeIntegration: false,
        }
    })

    // win.setWindowButtonVisibility(false)

    if (process.env.NODE_ENV !== 'development') {
        // Load production build
        mainWindow.loadFile(`${__dirname}/frontend/dist/index.html`)
    } else {
        // Load vite dev server page 
        console.log('Development mode')
        mainWindow.loadURL('http://localhost:5173')
        // mainWindow.loadFile(`${__dirname}/frontend/dist/index.html`)
    }

    // Maximize the window on startup
    // mainWindow.maximize()

    // setTimeout(() => {
    //     mainWindow.webContents.openDevTools()
    // }, 5000)


    // Send fullscreen change to renderer
    // const sendFullscreenState = () => {
    //     mainWindow.webContents.send('fullscreen-changed', mainWindow.isFullScreen());
    // };

    // mainWindow.on('enter-full-screen', sendFullscreenState);
    // mainWindow.on('leave-full-screen', sendFullscreenState);
    mainWindow.on('enter-full-screen', () => {
        console.log('[main] Entered fullscreen');
        mainWindow.webContents.send('fullscreen-changed', true);
    });

    mainWindow.on('leave-full-screen', () => {
        console.log('[main] Left fullscreen');
        mainWindow.webContents.send('fullscreen-changed', false);
    });

    // Send window state changes to renderer (for Windows custom titlebar)
    if (isWindows) {
        mainWindow.on('maximize', () => {
            mainWindow.webContents.send('window-maximized', true);
        });

        mainWindow.on('unmaximize', () => {
            mainWindow.webContents.send('window-maximized', false);
        });
    }

    // macOS dock icon
    if (process.platform === 'darwin') {
        app.dock.setIcon(nativeImage.createFromPath(iconPath))
    }


}

app.whenReady()
    .then(() => {
        startGrroxy()

        // Register IPC handlers once when app is ready
        ipcMain.handle('check-fullscreen', (event) => {
            if (mainWindow) {
                const isFs = mainWindow.isFullScreen();
                console.log('[main] check-fullscreen →', isFs);
                return isFs;
            }
            return false;
        });

        // Window control handlers for custom titlebar (Windows)
        ipcMain.handle('window-minimize', (event) => {
            if (mainWindow) {
                mainWindow.minimize();
            }
        });

        ipcMain.handle('window-maximize', (event) => {
            if (mainWindow) {
                if (mainWindow.isMaximized()) {
                    mainWindow.unmaximize();
                } else {
                    mainWindow.maximize();
                }
            }
        });

        ipcMain.handle('window-close', (event) => {
            if (mainWindow) {
                mainWindow.close();
            }
        });

        ipcMain.handle('window-is-maximized', (event) => {
            if (mainWindow) {
                return mainWindow.isMaximized();
            }
            return false;
        });

        createWindow()

        app.on('activate', function () {
            if (BrowserWindow.getAllWindows().length === 0) createWindow()
        })




    })

app.on('before-quit', () => {
    stopGrroxy()
})

app.on('window-all-closed', function () {
    if (process.platform !== 'darwin') app.quit()
})
