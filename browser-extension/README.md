# Mairu Browser Context Extension

A Chrome extension that gives Mairu real-time browser awareness. It captures page content, visual bounds, storage state, and errors, syncing them to the local Mairu context server.

## Features

- **Live Page Capture:** Automatically syncs the DOM text, selection, and active element of the current tab.
- **Visual Context:** Captures bounding boxes of major structural elements to understand spatial layout.
- **State & Errors:** Captures local/session storage keys and records console/network errors.
- **Agent Commands:** Exposes execution hooks (click, fill, highlight, scroll, navigate) allowing an agent to manipulate the active page via the Mairu Native Messaging host.
- **Context Menu:** Select text or images, right-click, and "Send to Mairu Agent".

## Setup

### 1. Install Dependencies
You need `rust`, `cargo`, and `wasm-pack` installed.
```sh
cargo install wasm-pack
```

### 2. Build and Install Native Host
From the root of the Mairu monorepo, run:
```sh
make install-browser-extension
```
This will:
1. Build the Rust native host binary (`browser-extension-host`).
2. Build the WASM module for the service worker.
3. Register the native host with Chrome.

### 3. Load the Extension
1. Open Chrome and navigate to `chrome://extensions/`.
2. Enable **Developer mode** in the top right corner.
3. Click **Load unpacked** and select the `mairu/browser-extension/extension` directory.

### 4. Verify Connection
1. Click the Mairu extension icon in your browser toolbar to open the popup.
2. The popup should show a green status dot indicating it's active.
3. The "Native Host" status should say "Connected" (this means the extension is successfully talking to the local native host).
