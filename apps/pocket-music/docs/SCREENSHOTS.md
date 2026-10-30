# Screenshot index

This index is the capture checklist; no screenshot files are present yet. Capture every image from the running application with Chromium headless mode. Do not create synthetic or manually authored screenshots. Replace the pending rows with captured filenames and notes, and verify each PNG's encoded dimensions match its filename.

## Required views and matrices

Required views: Home, Library, Albums, Artists, Now Playing, Queue, Playlist, Search, Settings, Light theme, and Dark theme.

Required viewport sizes: 320×240, 480×320, 720×720, and 1280×720. Ensure the small-screen capture demonstrates usable navigation and content. Capture theme examples in both Light and Dark modes. Add rows for each actual capture; include all required views and resolutions, adding captures as necessary to cover both themes.

| Filename | Resolution | Theme | View / proof | Capture command |
| --- | --- | --- | --- | --- |
| Pending real application capture | Pending | Pending | Home, Library, Albums, Artists, Now Playing, Queue, Playlist, Search, Settings; Light and Dark; all required sizes | See command below; set route/state and viewport for each image. |

## Capture workflow

1. Start the backend and frontend using `make dev` and wait until both are ready.
2. Open the intended application route in the running frontend and establish the view/theme state to capture. Use actual application navigation; do not fabricate UI state with a standalone HTML page.
3. Run Chromium with the selected viewport, for example:

```sh
chromium --headless=new --disable-gpu --no-sandbox --hide-scrollbars \
  --window-size=320,240 \
  --screenshot=apps/pocket-music/docs/screenshots/home-320x240-light.png \
  http://127.0.0.1:5173/
```

4. Repeat for each required view, resolution, and theme. Use the actual route or a documented interaction sequence where the UI is state-driven.
5. Check the PNG IHDR width and height against the filename, then record the exact command and any state-setting steps in the table. Store evidence under `apps/pocket-music/docs/screenshots/`.

The example command is a workflow template, not evidence that the application currently has a Home route or that a screenshot was captured.
