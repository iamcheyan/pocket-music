# Database

SQLite via `modernc.org/sqlite` (pure Go, no cgo). The schema is created and
migrated at startup in one transaction; `schema_version.version` is currently
**2**.

## Tables

```sql
schema_version(version INTEGER NOT NULL)

artists(id INTEGER PRIMARY KEY,
        name TEXT UNIQUE NOT NULL)

albums(id INTEGER PRIMARY KEY,
       title TEXT NOT NULL,
       artist_id INTEGER,
       year INTEGER,
       cover TEXT,
       UNIQUE(title, artist_id))

tracks(id TEXT PRIMARY KEY,           -- hex(path) for scanned files, mock-NNN for demo
       title TEXT,
       artist_id INTEGER,
       album_id INTEGER,
       genre TEXT,
       year INTEGER,
       duration REAL,                  -- seconds, from ffprobe
       path TEXT UNIQUE NOT NULL,
       mtime INTEGER,                  -- nanoseconds, incremental scan
       size INTEGER,                   -- bytes, incremental scan
       track_no INTEGER DEFAULT 0,
       added_at TEXT DEFAULT CURRENT_TIMESTAMP)

playlists(id INTEGER PRIMARY KEY,
          name TEXT NOT NULL,
          created_at TEXT DEFAULT CURRENT_TIMESTAMP)

playlist_items(playlist_id INTEGER,
               position INTEGER,
               track_id TEXT,
               PRIMARY KEY(playlist_id, position))

favorites(kind TEXT,                   -- 'track' | 'album' | 'artist'
          entity_id TEXT,              -- track id, or album/artist id as text
          created_at TEXT DEFAULT CURRENT_TIMESTAMP,
          PRIMARY KEY(kind, entity_id))

history(track_id TEXT PRIMARY KEY,
        last_played TEXT,
        play_count INTEGER DEFAULT 0,
        skip_count INTEGER DEFAULT 0,
        completion_rate REAL DEFAULT 0)

settings(key TEXT PRIMARY KEY,
         value TEXT)

-- external-content style full text index, maintained by the service
tracks_fts USING fts5(track_id UNINDEXED, title, artist, album, genre)
```

## Indexes

```sql
tracks_title_idx   ON tracks(title)
tracks_artist_idx  ON tracks(artist_id)
tracks_album_idx   ON tracks(album_id)
tracks_added_idx   ON tracks(added_at)
```

`tracks_title_idx` supports the `ORDER BY t.title LIMIT/OFFSET` of the paged
library, and the artist/album indexes back the per-album `count(*)` subqueries
that replaced a grouped join (roughly a 3× improvement on the 200-album,
10,000-track benchmark).

## Full-text search

`tracks_fts` holds a denormalised copy of `title`, `artist`, `album` and `genre`
for every track. It is written in the same transaction as the track row and
deleted when a row is pruned.

`ftsQuery()` turns free text into a safe MATCH expression: every word is quoted
and prefix-matched (`"perf"* "track"*`), so punctuation, quotes or operators in a
user query cannot produce a syntax error. `/api/search` bounds the candidate set
to 400 rows before joining back to `tracks`; the paginated
`/api/library/songs?q=` path deliberately keeps the full match set so totals and
paging stay exact.

## Write paths

- **Scan/import** — one transaction per file: upsert `artists`, `albums`, the
  `tracks` row (conflict on `path`), then refresh `tracks_fts`.
- **Prune** — after the walk, tracks whose file no longer exists are removed from
  `tracks_fts`, `playlist_items` and `tracks`.
- **Playback** — `history` is upserted once when a track starts
  (`last_played = CURRENT_TIMESTAMP`, `play_count += 1`) and once when it
  completes (`completion_rate = 1`) or is skipped
  (`skip_count += 1`, `completion_rate = position / duration`).
- **Playlists** — item ordering uses a transactional *position shift*: the moved
  row is deleted, following rows shift by 100000, the row is reinserted at the
  target position, then the temporaries shift back. This keeps
  `PRIMARY KEY(playlist_id, position)` valid throughout.
- **Favourites** — `INSERT OR IGNORE` / `DELETE`, no duplicates by construction.

## Settings keys

| Key            | Allowed values                              |
| -------------- | ------------------------------------------- |
| `language`     | `en`, `zh`, `ja`                            |
| `theme`        | `system`, `light`, `dark`                   |
| `music_folder` | any non-empty path                          |
| `cache_size`   | 16 – 32768 (MB)                             |
| `auto_scan`    | `true`, `false`                             |
| `audio_output` | `null`, `pulse`, `alsa`, `pipewire`, `jack` |
| `proxy`        | free text (reserved for remote providers)   |

Values are validated before writing; an unknown key or invalid value returns
`400` with a stable error code and nothing is written.

## Seed data (mock mode)

`MUSICD_MOCK=true` seeds only when `tracks` is empty, so restarts are idempotent.
The demo library is deterministic (`math/rand` seeded with a constant): 8 artists
× 3 albums × 10 tracks, eight genres, years 2011–2024, durations 118–308 s,
artist/album folder paths (`mock:Artist/Album/NN Title.mp3`), history for about a
quarter of the tracks, 15 favourites and 3 playlists.

## Migration notes

- `schema_version` starts at 1 and is bumped to 2 when the scanner bookkeeping
  columns (`mtime`, `size`) and `track_no` exist.
- Column additions are attempted with `ALTER TABLE … ADD COLUMN`; duplicate
  column errors on a fresh database are ignored on purpose.
- Startup also prunes orphaned FTS rows, so a database written by an older build
  converges to the current invariants without a destructive migration.
