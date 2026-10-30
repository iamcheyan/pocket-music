# API

All routes live under `/api` and speak JSON unless stated otherwise. The service
listens on `MUSICD_ADDR` (default `:8765`).

Errors always use the same envelope and a stable code:

```json
{ "error": { "code": "track_not_found", "message": "track not found" } }
```

| Status | Meaning                                              |
| ------ | ---------------------------------------------------- |
| 400    | invalid JSON, invalid mode, invalid setting/action   |
| 404    | unknown entity or endpoint                           |
| 405    | wrong method for an existing route                   |
| 409    | a scan is already running                            |
| 500    | database or serialisation failure                    |

## Track shape

Most responses embed the same track projection:

```json
{
  "id": "mock-011",
  "title": "Opening Light 11",
  "artist": "Nova Lantern",
  "album": "Signal Garden II",
  "genre": "Ambient",
  "year": 2012,
  "duration": 265.0,
  "path": "mock:Nova Lantern/Signal Garden II/01 Opening Light 11.mp3",
  "added_at": "2026-10-08 09:23:18",
  "album_id": 5,
  "artist_id": 1,
  "favorite": false,
  "play_count": 3,
  "skip_count": 0,
  "track_number": 1
}
```

`path` is omitted from list responses when empty and is only meaningful to the
service; clients should use `album_id` for covers and `/api/media/{id}` for audio.

## Health and home

| Method | Path          | Notes                                                        |
| ------ | ------------- | ------------------------------------------------------------ |
| GET    | `/api/health` | provider, mock flag, root, library counts, audio engine/output |
| GET    | `/healthz`    | liveness `{ok, mock}`                                         |
| GET    | `/api/home`   | `recently_added`, `recently_played`, `most_played`, `favorite_tracks`, `favorite_albums` |

## Library

| Method | Path                    | Query                          | Response |
| ------ | ----------------------- | ------------------------------ | -------- |
| GET    | `/api/library/songs`    | `page` (≥1), `limit` (1–500), `q`, `folder` | `{items, total, page, limit}` |
| GET    | `/api/library/albums`   | `q`                            | `{items:[{id,title,artist,year,track_count,favorite,cover}], total}` |
| GET    | `/api/library/artists`  | `q`                            | `{items:[{id,name,track_count,favorite}], total}` |
| GET    | `/api/library/genres`   | —                              | `{items:[{name,track_count}], total}` |
| GET    | `/api/library/folders`  | —                              | `{items:[{name,track_count}], total}` |
| GET    | `/api/albums/{id}`      | —                              | `{id,title,artist,artist_id,year,cover,favorite,duration,tracks[]}` |
| GET    | `/api/artists/{id}`     | —                              | `{id,name,favorite,tracks[],albums[]}` |
| GET    | `/api/tracks/{id}`      | —                              | track shape plus `has_lyrics` |

`q` on songs uses the FTS index; `folder` filters by path segment (accepts both
absolute paths and the demo `mock:` prefix) and can be combined with `q`.
Folder names are artist/album relative paths, for example
`Nova Lantern/Night Cartography I`.

## Search

`GET /api/search?q=…&type=all|song|album|artist|playlist`

```json
{
  "query": "Vesper",
  "songs": [ … ],
  "albums": [{ "id": 9, "title": "Slow Machines II", "year": 2016, "cover": "/api/cover/9" }],
  "artists": [{ "id": 6, "name": "Vesper Lane" }],
  "playlists": [],
  "total": 27
}
```

The song list is a bounded ranked preview (first 400 FTS candidates, 50 results).
An empty `q` returns empty groups rather than the whole library.

## Playback

| Method | Path                  | Body                                   |
| ------ | --------------------- | -------------------------------------- |
| GET    | `/api/player/state`   | —                                      |
| POST   | `/api/play`           | `{track_id, queue?: [id], mode?}`       |
| POST   | `/api/player/pause`   | —                                      |
| POST   | `/api/player/resume`  | —                                      |
| POST   | `/api/player/next`    | —                                      |
| POST   | `/api/player/prev`    | —                                      |
| POST   | `/api/player/seek`    | `{position}` (clamped to `[0, duration]`) |
| POST   | `/api/player/volume`  | `{volume}` (clamped to `[0, 100]`)      |
| PUT    | `/api/player/mode`    | `{mode}` — `normal`, `repeat_all`, `repeat_one`, `shuffle` |
| POST   | `/api/queue`          | see below                              |

Player state:

```json
{
  "track": { … },
  "playing": true,
  "position": 44.2,
  "duration": 265.0,
  "volume": 80,
  "mode": "repeat_all",
  "queue": [ { … } ],
  "source": "local"
}
```

Queue actions (`POST /api/queue`, response `{queue: [...]}`):

| `action`    | Payload                    | Effect                                    |
| ----------- | -------------------------- | ----------------------------------------- |
| `add`       | `track_id` or `track_ids`  | append one or many tracks                 |
| `play_next` | `track_id`                 | insert at the head of the queue           |
| `remove`    | `index`                    | remove one position                       |
| `reorder`   | `index`, `to`              | move an entry, preserving length          |
| `clear`     | —                          | empty the queue                           |

`POST /api/play` replaces the queue when `queue` is supplied; it is how album,
artist and playlist playback work in one round trip.

## Favourites

| Method | Path              | Body                          |
| ------ | ----------------- | ----------------------------- |
| GET    | `/api/favorites`  | —                             |
| POST   | `/api/favorites`  | `{kind, id}` (`track`/`album`/`artist`) |
| PUT    | `/api/favorites`  | same as POST                  |
| DELETE | `/api/favorites`  | `{kind, id}` or `?kind=&id=`  |

`GET` returns the three entity lists plus raw ids for quick membership checks:

```json
{
  "tracks": [ … ],
  "albums": [ … ],
  "artists": [ … ],
  "ids": { "track": ["mock-007"], "album": ["2"], "artist": ["1"] }
}
```

Adding an unknown entity returns `404`; an unknown `kind` returns `400`.

## Playlists

| Method | Path                            | Body / Query                |
| ------ | ------------------------------- | --------------------------- |
| GET    | `/api/playlists`                | —                           |
| POST   | `/api/playlists`                | `{name, track_ids?}`         |
| GET    | `/api/playlists/{id}`           | —                           |
| PUT    | `/api/playlists/{id}`           | `{name}`                    |
| DELETE | `/api/playlists/{id}`           | — (`204`)                   |
| POST   | `/api/playlists/{id}/items`     | `{action: add|remove|reorder|clear, …}` |
| GET    | `/api/playlists/{id}/export`    | `?format=m3u|m3u8`           |
| POST   | `/api/playlists/import`         | multipart `file` (+ `name`)  |

Item actions: `add` takes `track_id` or `track_ids`, `remove` takes `index`,
`reorder` takes `index` and `to`; every mutation responds with the updated
playlist (`{id, name, tracks[], track_count, duration}`).

Export returns `audio/x-mpegurl` with `#EXTM3U` and one `#EXTINF:<seconds>,<artist> - <title>`
line per track; `m3u8` adds a UTF-8 BOM. Import accepts `#EXTM3U`/`#EXTINF`,
relative paths, percent-escaped paths and `file://` URIs, matching entries by
absolute path first and by path tail second (uploads do not carry their source
directory). Response: `{id, name, count, missing}`.

## Media

| Method | Path                  | Notes                                             |
| ------ | --------------------- | ------------------------------------------------- |
| GET    | `/api/media/{trackID}`| audio bytes, Range supported                      |
| GET    | `/api/cover/{albumID}`| stored art, folder.jpg/cover.jpg, embedded art, generated SVG |
| GET    | `/api/lyrics/{trackID}` | `{track_id, lines:[{time,text}], synced, count}` |

`/api/media` returns `404 media_missing` for demo rows (no file on disk).
Covers return `image/jpeg`/`image/png` for real art and `image/svg+xml` for the
generated fallback, with a one-day cache header.

## Scanning

| Method | Path                | Notes                                                      |
| ------ | ------------------- | ---------------------------------------------------------- |
| POST   | `/api/scan`         | start a scan; `409 scan_running` if one is active           |
| GET    | `/api/scan`         | current progress                                            |
| GET    | `/api/scan/status`  | current progress                                            |
| POST   | `/api/scan/cancel`  | cancel the running scan                                     |

```json
{ "scanning": true, "found": 812, "total": 2400, "skipped": 0, "current": "/music/a.flac", "done": false }
```

## Settings

`GET /api/settings` → `{settings: {…}, root}`. `PUT /api/settings` accepts
`{settings: {key: value}}` for the keys documented in
[DATABASE.md](DATABASE.md#settings-keys); unknown keys and invalid values are
rejected with `400` before anything is written. Changing `audio_output` applies
immediately; `auto_scan=true` starts a scan in the background.

## Statistics

`GET /api/stats`

```json
{
  "library": { "tracks": 240, "albums": 24, "artists": 8, "playlists": 3 },
  "play_time": 84120.5,
  "play_count": 385,
  "skip_count": 66,
  "completion": 0.73,
  "most_played": [ … ],
  "most_skipped": [ … ],
  "top_artists": [{ "id": 6, "name": "Vesper Lane", "play_count": 57 }],
  "top_albums": [{ "id": 9, "title": "Slow Machines II", "artist": "Vesper Lane", "play_count": 21 }]
}
```

`play_time` is `Σ(duration × play_count)` for tracks with history; `completion`
is the mean `completion_rate`.

## Static UI

When `MUSICD_WEB_DIR` is set, every non-`/api` route serves the compiled
frontend and falls back to `index.html` so client-side URLs survive a reload.
Without it, only the API and `/healthz` are served.
