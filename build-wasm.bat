@echo off
echo Building ATC Simulator for WebAssembly...

REM Set environment variables for WASM build
set GOOS=js
set GOARCH=wasm

REM Build the WASM binary
go build -o web\game.wasm

REM Copy the wasm_exec.js file from Go installation
copy "%GOROOT%\misc\wasm\wasm_exec.js" web\

echo.
echo Build complete!
echo.
echo To run the game:
echo 1. Navigate to the web directory: cd web
echo 2. Start a local server: python -m http.server 8080
echo    (or use any other web server)
echo 3. Open browser to: http://localhost:8080
echo.
