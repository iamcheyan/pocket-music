# Roadmap

This document is an integration checklist. Items are marked verified only after implementation and evidence have been reviewed. At documentation authoring time, repository implementation and runtime checks had not been inspected, so status remains unverified.

## Required implementation

- [ ] Start `musicd` and the frontend together with one root command.
- [ ] Persist library, playlist, favorites, history, and settings data in SQLite with versioned migrations.
- [ ] Scan supported local audio formats incrementally, extract metadata and covers, report progress, and support cancellation.
- [ ] Provide indexed and paginated library queries and measure search/list behavior with 10,000+ tracks.
- [ ] Drive mpv through JSON IPC; provide useful playback state and queue, repeat, and shuffle behavior.
- [ ] Support mpv null audio output (`--ao=null`) on headless machines so playback state, progress, and completion continue without physical audio output. Allow output selection through settings.
- [ ] Complete the documented HTTP API and frontend workflows, layouts, responsive sizes, keyboard/gamepad navigation, localization, and themes.
- [ ] Add integration evidence, real application screenshots, and test results.

## Known limitations and evidence still needed

No implementation status, measured performance figures, endpoint test results, or real screenshots are asserted here. Fill in verified status and concrete limitations during integration. Remote Bilibili, Navidrome, and Jellyfin providers are extension points only; their implementation and synchronization are not required for the initial local-library application.

## Future PJOS migration

1. Keep user workflows and business state behind the `musicd` API.
2. Replace Vite development hosting with the target PJOS runtime integration while preserving API behavior.
3. Validate the API transport, input/focus handling, storage configuration, and media rendering on each target device.
4. Adapt provider implementations for device storage and any future remote services without moving service-only responsibilities into UI code.
