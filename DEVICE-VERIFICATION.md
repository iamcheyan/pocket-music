# DEVICE VERIFICATION — pocket-music

Status: **DEVICE-PASS** for the captured states. Local host verification is in
`apps/pocket-music/POCKETJS-VERIFICATION.md`.

## Device

- Anbernic RG34XX, Base OS 1.5.2 (h700), kernel 4.9.170 aarch64.
- Access: Debian host -> HX90 -> USB ADB.
- Display: `/dev/fb0`, 720x480 visible page, 32 bpp.
- Runtime: `/mnt/sdcard/System/pocketjs/bin/pocket-linux-kms` (existing boot runtime, not rebuilt or replaced).
- Package: `pocket-music` sha256 `b4c7c4faa3329905e95f72654ceb7982e91ee8c80b6114b859814ca1e06219a7`.
- Deployed to `/storage/pjs/pocket-music/app.pocket` only.

## Procedure

1. Recorded boot launcher pid `15339` and `boot.pocket`
   sha256 `6ed41612bce1723d1eda64b5a0ee9333a105ed498b7085c03267539cf6c1b32d`.
2. Paused the launcher supervisor, stopped the launcher, and started the
   package with the same runtime and `/dev/fb0`.
3. Created a virtual keyboard with `/dev/uinput` and drove the native view
   tree through a FIFO (`uinputd-c`); no host browser and no HTML.
4. Captured `/dev/fb0` with `dd` -> base64 -> PIL.
5. Killed the package, restored the launcher, and read the pid
   `15746` and boot hash back.

Result: launcher pid changed, boot package sha256 stayed
`6ed41612bce1723d1eda64b5a0ee9333a105ed498b7085c03267539cf6c1b32d`. No write to `/storage/System`, `/storage/pocketjs`,
`/storage/baseos.conf` or `/storage/content`.

## Frames

The images are the device framebuffer. The UI was rasterized by the PocketJS
native renderer on the RG34XX, not by host Chrome.

| file | sha256 (first 16) |
|---|---|
| `docs/device/device-music-detail-action.png` | `4a8266729b6bbc6f` |
| `docs/device/device-music-detail.png` | `eff548351553b556` |
| `docs/device/device-music-fb-after.png` | `d59130e3329b8496` |
| `docs/device/device-music-fb-before.png` | `d59130e3329b8496` |
| `docs/device/device-music-home-focus.png` | `f03001ecb2a6d022` |
| `docs/device/device-music-home.png` | `bfa1d30c1daa16be` |
| `docs/device/device-music-settings-toggled.png` | `e41c99a183d440fc` |
| `docs/device/device-music-settings.png` | `bdb4ccd4d033e065` |

`device-music-fb-before.png` and `device-music-fb-after.png` are the boot
launcher before and after; they are byte-identical here, which is the intended
restore evidence.

Captured states: 6 states: home, focus, detail, detail action, settings, settings toggle.
