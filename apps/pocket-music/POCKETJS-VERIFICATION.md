# POCKETJS VERIFICATION — pocket-music

Round: PocketJS refactor. Device acceptance is a later pass.

## Shape

- Pocket app: `apps/pocket-music/pocket/`
- Frozen React reference: `apps/pocket-music/frontend/` (design reference; not extended).
- TSX files in the port: **6** (`main.tsx`, `app.tsx`, `ui.tsx`, three screens).
- Native tree only: `View`/`Text` from `@pocketjs/framework/solid/components`; no DOM, iframe, texture screenshot, or `dangerouslySetInnerHTML`.

## Manifest

- id: `dev.pocket-music.pocket`
- entry: `src/main.tsx`
- output: `pocket-music`
- viewport: `360×240 @2x` → physical `720×480`
- capabilities: `text.glyphs.baked`, `input.buttons`
- target: `rocknix-h700-kms`

## Admission / compile

```sh
cd apps/pocket-music/pocket
./setup-sdk.sh
SDK=${POCKETJS_SDK:-$HOME/development/.pocketjs-star-drift-sdk}
bun "$SDK/tools/pocket.ts" check \
  --runtime-config "$POCKET_VERIFY/pjs/profiles/pocket.runtime-config.ts" \
  --target rocknix-h700-kms --manifest pocket.json
bun "$SDK/tools/pocket.ts" compile \
  --runtime-config "$POCKET_VERIFY/pjs/profiles/pocket.runtime-config.ts" \
  --target rocknix-h700-kms --manifest pocket.json --output out.pocket
```

Admission: `sha256:ec5ffc3400a3e7e8edaa4931a940ffe075ebee63be83f6abda4e4e9827e50d18`; TypeScript passes for the app module graph. Package: `out.pocket`, sha256 `e46a89113d7606d43547be0befc867a00cc977b85692b245e85a44ccf88f9185`.

## Frames

Rendered with the SDK wasm core at `renderScaled(2)` (720×480), not a browser:

```sh
cd "$POCKET_VERIFY"
bun tools/render-pocket.ts \
  --dir repos/pocket-music/apps/pocket-music/pocket \
  --script repos/pocket-music/apps/pocket-music/pocket/frames.json \
  --out repos/pocket-music/docs/pocket
```

| file | state | sha256 (first 16) |
|---|---|---|
| `docs/pocket/screen-home.png` | home | `9906498c1d9d0f30` |
| `docs/pocket/screen-home-focus.png` | home-focus | `ecd2f3bf21448994` |
| `docs/pocket/screen-detail.png` | detail | `aac738c0f05a9d8c` |
| `docs/pocket/screen-detail-action.png` | detail-action | `6ca7302d095eb5bc` |
| `docs/pocket/screen-settings.png` | settings | `939427e8a2476b5e` |
| `docs/pocket/screen-settings-toggled.png` | settings-toggled | `25e03099751c030f` |

All six files have distinct bytes; the renderer rejects duplicate frames. The
tape is: move focus, open the item, run the first detail action, switch to
settings, move a row, toggle it.

Screens: Library/playlists, now-playing, playback settings. Interactions: PREV/PLAY/NEXT/FAV; progress and playing state change, no audio output.

## Adapter

- **fixture** — this port runs from JSON baked into the package and labels the
  header `fixture`. `client.ts` also contains the `http` and `bridge` adapters
  behind the same `list()/get()/action()` surface.
- `http` uses `@pocketjs/framework/net-api`; valid only in the host dev profile.
- `bridge` uses `host.__pvc(JSON.stringify(request))`; the runtime forwards to
  the Go daemon on `127.0.0.1:<port>` with a hard timeout.
- Device gap: musicd bridge plus a native audio sink. mpv/ALSA is not reachable from the device profile.
- Daemon contract: /api/library/*, /api/playlists, /api/playback; audio is mock.

## Not done / open questions

- Device run, saves/annotations and the bridge itself are out of this round.
- Real media/audio, OAuth, emulator, engine and system writes stay unimplemented
  where the original product already marked them mock or unavailable.
- Long lists use the six-item fixture surface; paging/virtual lists are a
  follow-up once the bridge provides paginated data.
