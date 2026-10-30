export type Track = {
  id: string;
  title: string;
  artist: string;
  album: string;
  genre: string;
  year: number;
  duration: number;
  path?: string;
  added_at: string;
  album_id: number;
  artist_id: number;
  favorite: boolean;
  play_count: number;
  skip_count: number;
  track_number: number;
};

export type AlbumSummary = {
  id: number;
  title: string;
  artist: string;
  year: number;
  track_count: number;
  favorite?: boolean;
  cover?: string;
};

export type ArtistSummary = {
  id: number;
  name: string;
  track_count: number;
  favorite?: boolean;
};

export type NamedCount = { name: string; track_count: number };

export type AlbumDetail = AlbumSummary & { tracks: Track[]; duration: number; artist_id: number };
export type ArtistDetail = { id: number | string; name: string; favorite: boolean; tracks: Track[]; albums: AlbumSummary[] };

export type Player = {
  track: Track | null;
  playing: boolean;
  position: number;
  duration: number;
  volume: number;
  mode: PlaybackMode;
  queue: Track[];
  source: string;
};

export type PlaybackMode = 'normal' | 'repeat_all' | 'repeat_one' | 'shuffle';

export type HomeData = {
  recently_added: Track[];
  recently_played: Track[];
  most_played: Track[];
  favorite_tracks: Track[];
  favorite_albums: AlbumSummary[];
};

export type Playlist = {
  id: number | string;
  name: string;
  created_at?: string;
  track_count?: number;
  tracks?: Track[];
  duration?: number;
};

export type SearchResults = {
  query: string;
  songs: Track[];
  albums: AlbumSummary[];
  artists: ArtistSummary[];
  playlists: Playlist[];
  total: number;
};

export type Health = {
  provider: string;
  mock: boolean;
  root: string;
  library: { tracks: number; albums: number; artists: number };
  audio: { status: string; ao: string; engine: string };
};

export type ScanProgress = {
  scanning: boolean;
  found: number;
  total: number;
  skipped: number;
  current?: string;
  error?: string;
  done: boolean;
};

export type Stats = {
  library: { tracks: number; albums: number; artists: number; playlists: number };
  play_time: number;
  play_count: number;
  skip_count: number;
  completion: number;
  most_played: Track[];
  most_skipped: Track[];
  top_artists: { id: number; name: string; play_count: number }[];
  top_albums: (AlbumSummary & { play_count: number })[];
};

export type Lyrics = { track_id: string; lines: { time: number; text: string }[]; synced: boolean; count: number };

export type Favorites = {
  tracks: Track[];
  albums: AlbumSummary[];
  artists: ArtistSummary[];
  ids: { track: string[]; album: string[]; artist: string[] };
};

export type SettingsMap = {
  language: string;
  theme: string;
  music_folder: string;
  cache_size: string;
  auto_scan: string;
  audio_output: string;
  proxy: string;
};

export type SongPage = { items: Track[]; total: number; page: number; limit: number };

export type ItemPage<T> = { items: T[]; total: number };
