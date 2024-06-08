# CHANGELOGS

## Current Changes

#### ReadFile param changes from `from` to `location` changed to

```javascript
    //previously 
    const filedata = await readFile({
        fileName: filename,
        from: 'cwd'
    });

    //now 
    const filedata = await readFile({
        fileName: filename,
        folder: 'cwd'
    });
```
---------------
#### DESKTOP APP
  - Use `--app` to launch the desktop application
  - Installation `wails build -skipbindings -debug`
  - We are using one `/frontend` throughout the project, althought wails require to have a fodler name `/frontend` so we have `/cmd/grroxy/frontend`

#### Package name changed from `base` to `utils`