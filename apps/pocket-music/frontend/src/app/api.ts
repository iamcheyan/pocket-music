import type { AlbumDetail, ArtistDetail, Favorites, Health, HomeData, ItemPage, Lyrics, Player, Playlist, ScanProgress, SearchResults, SettingsMap, SongPage, Stats, Track, AlbumSummary, ArtistSummary, NamedCount, PlaybackMode } from './types';

const BASE = '/api';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${BASE}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
  });
  if (!response.ok) {
    let detail = `${response.status} ${response.statusText}`;
    try {
      const body = await response.json();
      if (body?.error?.message) {
        detail = `${body.error.code}: ${body.error.message}`;
      }
    } catch {
      /* non-JSON error body */
    }
    throw new Error(detail);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

const post = <T>(path: string, body?: unknown) =>
  request<T>(path, { method: 'POST', body: body === undefined ? undefined : JSON.stringify(body) });
const put = <T>(path: string, body?: unknown) =>
  request<T>(path, { method: 'PUT', body: body === undefined ? undefined : JSON.stringify(body) });

export const api = {
  health: () => request<Health>('/health'),
  home: () => request<HomeData>('/home'),

  songs: (params: { page?: number; limit?: number; q?: string } = {}) => {
    const search = new URLSearchParams();
    if (params.page) search.set('page', String(params.page));
    if (params.limit) search.set('limit', String(params.limit));
    if (params.q) search.set('q', params.q);
    return request<SongPage>(`/library/songs?${search.toString()}`);
  },
  albums: (q = '') => request<ItemPage<AlbumSummary>>(`/library/albums?q=${encodeURIComponent(q)}`),
  artists: (q = '') => request<ItemPage<ArtistSummary>>(`/library/artists?q=${encodeURIComponent(q)}`),
  genres: () => request<ItemPage<NamedCount>>('/library/genres'),
  folders: () => request<ItemPage<NamedCount>>('/library/folders'),
  album: (id: number | string) => request<AlbumDetail & { favorite: boolean }>(`/albums/${id}`),
  artist: (id: number | string) => request<ArtistDetail>(`/artists/${id}`),
  track: (id: string) => request<Track>(`/tracks/${encodeURIComponent(id)}`),
  lyrics: (id: string) => request<Lyrics>(`/lyrics/${encodeURIComponent(id)}`),
  search: (q: string, type = 'all') => request<SearchResults>(`/search?q=${encodeURIComponent(q)}&type=${type}`),

  player: () => request<Player>('/player/state'),
  play: (trackId: string, queue: string[] = [], mode?: PlaybackMode) =>
    post<Player>('/play', { track_id: trackId, queue, mode }),
  pause: () => post<Player>('/player/pause'),
  resume: () => post<Player>('/player/resume'),
  next: () => post<Player>('/player/next'),
  prev: () => post<Player>('/player/prev'),
  seek: (position: number) => post<Player>('/player/seek', { position }),
  volume: (volume: number) => post<Player>('/player/volume', { volume }),
  mode: (mode: PlaybackMode) => put<Player>('/player/mode', { mode }),

  queue: (action: 'add' | 'play_next' | 'remove' | 'reorder' | 'clear', payload: Record<string, unknown> = {}) =>
    post<{ queue: Track[] }>('/queue', { action, ...payload }),

  settings: () => request<{ settings: SettingsMap; root: string }>('/settings'),
  saveSettings: (settings: Partial<SettingsMap>) => put<{ settings: SettingsMap; root: string }>('/settings', { settings }),

  scan: () => post<{ started: boolean }>('/scan'),
  scanStatus: () => request<ScanProgress>('/scan/status'),
  scanCancel: () => post<{ cancelled: boolean }>('/scan/cancel'),

  playlists: () => request<ItemPage<Playlist>>('/playlists'),
  playlist: (id: number | string) => request<Playlist>(`/playlists/${id}`),
  createPlaylist: (name: string, trackIds: string[] = []) => post<Playlist>('/playlists', { name, track_ids: trackIds }),
  renamePlaylist: (id: number | string, name: string) => put<Playlist>(`/playlists/${id}`, { name }),
  deletePlaylist: (id: number | string) => request<void>(`/playlists/${id}`, { method: 'DELETE' }),
  playlistItems: (
    id: number | string,
    action: 'add' | 'remove' | 'reorder' | 'clear',
    payload: Record<string, unknown> = {},
  ) => post<Playlist>(`/playlists/${id}/items`, { action, ...payload }),
  importPlaylist: async (file: File, name: string) => {
    const form = new FormData();
    form.append('file', file, file.name);
    form.append('name', name);
    const response = await fetch(`${BASE}/playlists/import`, { method: 'POST', body: form });
    if (!response.ok) {
      throw new Error(`import failed: ${response.status}`);
    }
    return (await response.json()) as { id: number; name: string; count: number; missing: number };
  },

  favorites: () => request<Favorites>('/favorites'),
  addFavorite: (kind: 'track' | 'album' | 'artist', id: string | number) =>
    post<{ favorite: boolean }>('/favorites', { kind, id: String(id) }),
  removeFavorite: (kind: 'track' | 'album' | 'artist', id: string | number) =>
    request<{ favorite: boolean }>('/favorites', { method: 'DELETE', body: JSON.stringify({ kind, id: String(id) }) }),

  stats: () => request<Stats>('/stats'),
  coverUrl: (albumId?: number | null) => (albumId ? `${BASE}/cover/${albumId}` : ''),
};
