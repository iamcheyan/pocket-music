import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api } from './api';
import type {
  AlbumDetail, AlbumSummary, ArtistDetail, ArtistSummary, Favorites, Health, HomeData,
  Lyrics, NamedCount, PlaybackMode, Player, Playlist, ScanProgress, SettingsMap,
  SongPage, Stats, Track,
} from './types';
import { locale, type Lang } from '../locale';
import { nextMode } from '../lib/mode';

export type Page = 'home' | 'library' | 'album' | 'artist' | 'now' | 'queue' | 'playlists' | 'playlist' | 'search' | 'settings' | 'stats';
export type Layout = 'music' | 'library' | 'minimal';
export type Theme = 'system' | 'light' | 'dark';
export type LibraryTab = 'songs' | 'albums' | 'artists' | 'genres' | 'folders' | 'favorites';

export type Nav = { page: Page; tab: LibraryTab; id: string; q: string };

const PAGE_SIZE = 100;

const emptyPlayer: Player = { track: null, playing: false, position: 0, duration: 0, volume: 70, mode: 'normal', queue: [], source: 'local' };

function readString(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function writeString(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {
    /* storage may be unavailable in a sandboxed shell */
  }
}

export function usePocket() {
  const params = useMemo(() => new URLSearchParams(window.location.search), []);

  const [lang, setLangState] = useState<Lang>(() => (params.get('lang') as Lang) || (readString('pm-language') as Lang) || 'en');
  const [theme, setThemeState] = useState<Theme>(() => (params.get('theme') as Theme) || (readString('pm-theme') as Theme) || 'system');
  const [layout, setLayoutState] = useState<Layout>(() => (params.get('layout') as Layout) || (readString('pm-layout') as Layout) || 'music');

  const [nav, setNav] = useState<Nav>(() => ({
    page: (params.get('view') as Page) || 'home',
    tab: (params.get('tab') as LibraryTab) || 'songs',
    id: params.get('id') || '',
    q: params.get('q') || '',
  }));
  const historyRef = useRef<Nav[]>([]);

  const [home, setHome] = useState<HomeData>({ recently_added: [], recently_played: [], most_played: [], favorite_tracks: [], favorite_albums: [] });
  const [songs, setSongs] = useState<Track[]>([]);
  const [songTotal, setSongTotal] = useState(0);
  const [albums, setAlbums] = useState<AlbumSummary[]>([]);
  const [artists, setArtists] = useState<ArtistSummary[]>([]);
  const [genres, setGenres] = useState<NamedCount[]>([]);
  const [folders, setFolders] = useState<NamedCount[]>([]);
  const [playlists, setPlaylists] = useState<Playlist[]>([]);
  const [favorites, setFavorites] = useState<Favorites>({ tracks: [], albums: [], artists: [], ids: { track: [], album: [], artist: [] } });
  const [settings, setSettings] = useState<SettingsMap>({ language: 'en', theme: 'system', music_folder: '', cache_size: '512', auto_scan: 'false', audio_output: 'null', proxy: '' });
  const [health, setHealth] = useState<Health | null>(null);
  const [player, setPlayer] = useState<Player>(emptyPlayer);
  const [scan, setScan] = useState<ScanProgress>({ scanning: false, found: 0, total: 0, skipped: 0, done: false });

  const [album, setAlbum] = useState<AlbumDetail | null>(null);
  const [artist, setArtist] = useState<ArtistDetail | null>(null);
  const [playlist, setPlaylist] = useState<Playlist | null>(null);
  const [lyrics, setLyrics] = useState<Lyrics | null>(null);
  const [searchResults, setSearchResults] = useState<{ songs: Track[]; albums: AlbumSummary[]; artists: ArtistSummary[]; playlists: Playlist[]; total: number }>({ songs: [], albums: [], artists: [], playlists: [], total: 0 });
  const [stats, setStats] = useState<Stats | null>(null);

  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState('');
  const [drawer, setDrawer] = useState(false);
  const [playingId, setPlayingId] = useState<string | null>(null);

  const tr = useCallback((key: string) => locale[lang][key] ?? locale.en[key] ?? key, [lang]);

  const fail = useCallback((reason: unknown) => {
    setError(reason instanceof Error ? reason.message : String(reason));
  }, []);

  const announce = useCallback((message: string) => {
    setToast(message);
    window.setTimeout(() => setToast((current) => (current === message ? '' : current)), 2600);
  }, []);

  const loadLibrary = useCallback(async (tab: LibraryTab = 'songs', q = '') => {
    const [songPage, albumPage, artistPage, genrePage, folderPage, playlistPage] = await Promise.allSettled([
      api.songs({ page: 1, limit: PAGE_SIZE, q: tab === 'songs' ? q : '' }),
      api.albums(tab === 'albums' ? q : ''),
      api.artists(tab === 'artists' ? q : ''),
      api.genres(),
      api.folders(),
      api.playlists(),
    ]);
    if (songPage.status === 'fulfilled') {
      setSongs(songPage.value.items);
      setSongTotal(songPage.value.total);
    }
    if (albumPage.status === 'fulfilled') setAlbums(albumPage.value.items);
    if (artistPage.status === 'fulfilled') setArtists(artistPage.value.items);
    if (genrePage.status === 'fulfilled') setGenres(genrePage.value.items);
    if (folderPage.status === 'fulfilled') setFolders(folderPage.value.items);
    if (playlistPage.status === 'fulfilled') setPlaylists(playlistPage.value.items);
  }, []);

  const loadHome = useCallback(async () => {
    try {
      setHome(await api.home());
    } catch (reason) {
      fail(reason);
    }
  }, [fail]);

  const loadFavorites = useCallback(async () => {
    try {
      setFavorites(await api.favorites());
    } catch {
      /* favourites are additive; a failure must not blank the library */
    }
  }, []);

  const refresh = useCallback(async () => {
    setBusy(true);
    const [homeResult, healthResult, settingsResult, favoritesResult, playerResult, playlistResult] = await Promise.allSettled([
      api.home(), api.health(), api.settings(), api.favorites(), api.player(), api.playlists(),
    ]);
    if (homeResult.status === 'fulfilled') setHome(homeResult.value);
    if (healthResult.status === 'fulfilled') setHealth(healthResult.value);
    if (settingsResult.status === 'fulfilled') setSettings(settingsResult.value.settings);
    if (favoritesResult.status === 'fulfilled') setFavorites(favoritesResult.value);
    if (playerResult.status === 'fulfilled') setPlayer(playerResult.value);
    if (playlistResult.status === 'fulfilled') setPlaylists(playlistResult.value.items);
    if ([homeResult, healthResult, settingsResult].every((result) => result.status === 'rejected')) {
      setError(tr('errorService'));
    }
    setBusy(false);
  }, [tr]);

  const loadSongs = useCallback(async (page: number, q = '', append = false) => {
    try {
      const result: SongPage = await api.songs({ page, limit: PAGE_SIZE, q });
      setSongTotal(result.total);
      setSongs((current) => (append ? [...current, ...result.items] : result.items));
    } catch (reason) {
      fail(reason);
    }
  }, [fail]);

  const loadMoreSongs = useCallback(() => {
    const next = Math.floor(songs.length / PAGE_SIZE) + 1;
    return loadSongs(next, nav.tab === 'songs' ? nav.q : '', true);
  }, [loadSongs, nav.q, nav.tab, songs.length]);

  const navigate = useCallback((page: Page, options: Partial<Omit<Nav, 'page'>> = {}) => {
    setNav((current) => {
      historyRef.current = [...historyRef.current, current];
      return { page, tab: options.tab ?? current.tab, id: options.id ?? '', q: options.q ?? (page === 'search' ? current.q : '') };
    });
    setDrawer(false);
    setError('');
  }, []);

  const goBack = useCallback(() => {
    const previous = historyRef.current.pop();
    setNav(previous ?? { page: 'home', tab: 'songs', id: '', q: '' });
    setDrawer(false);
  }, []);

  // URL state keeps deep links and screenshots deterministic.
  useEffect(() => {
    const search = new URLSearchParams();
    search.set('view', nav.page);
    if (nav.page === 'library') search.set('tab', nav.tab);
    if (nav.id) search.set('id', nav.id);
    if (nav.q) search.set('q', nav.q);
    search.set('theme', theme);
    search.set('lang', lang);
    search.set('layout', layout);
    window.history.replaceState(null, '', `${window.location.pathname}?${search.toString()}`);
  }, [nav, theme, lang, layout]);

  useEffect(() => {
    const root = document.documentElement;
    root.dataset.theme = theme;
    root.dataset.layout = layout;
    root.lang = lang === 'zh' ? 'zh-CN' : lang;
    writeString('pm-theme', theme);
    writeString('pm-layout', layout);
    writeString('pm-language', lang);
  }, [theme, lang, layout]);

  useEffect(() => {
    void refresh();
    void loadLibrary('songs', '');
  }, [refresh, loadLibrary]);

  // Poll playback state; the service owns the clock (mpv or the mock engine).
  useEffect(() => {
    let cancelled = false;
    const tick = async () => {
      try {
        const state = await api.player();
        if (!cancelled) {
          setPlayer(state);
          setPlayingId(state.track?.id ?? null);
        }
      } catch {
        /* transient service hiccup while polling */
      }
    };
    void tick();
    const timer = window.setInterval(tick, 1200);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, []);

  // Load the view-specific resource when navigation changes.
  useEffect(() => {
    setError('');
    if (nav.page === 'album' && nav.id) {
      void api.album(nav.id).then((value) => setAlbum({ ...value, favorite: (value as { favorite?: boolean }).favorite })).catch(fail);
    }
    if (nav.page === 'artist' && nav.id) {
      void api.artist(nav.id).then(setArtist).catch(fail);
    }
    if (nav.page === 'playlist' && nav.id) {
      void api.playlist(nav.id).then(setPlaylist).catch(fail);
    }
    if (nav.page === 'stats') {
      void api.stats().then(setStats).catch(fail);
    }
    if (nav.page === 'library' || nav.page === 'home') {
      void loadLibrary(nav.tab, nav.q);
    }
  }, [nav.page, nav.id, nav.tab, nav.q, fail, loadLibrary]);

  // Instant search with a short debounce so typing stays responsive.
  useEffect(() => {
    if (nav.page !== 'search') {
      return;
    }
    const query = nav.q.trim();
    if (!query) {
      setSearchResults({ songs: [], albums: [], artists: [], playlists: [], total: 0 });
      return;
    }
    const timer = window.setTimeout(() => {
      void api.search(query, 'all').then(setSearchResults).catch(fail);
    }, 160);
    return () => window.clearTimeout(timer);
  }, [nav.page, nav.q, fail]);

  // Lyrics follow the playing track.
  useEffect(() => {
    const track = player.track;
    if (!track) {
      setLyrics(null);
      return;
    }
    void api.lyrics(track.id).then(setLyrics).catch(() => setLyrics(null));
  }, [player.track?.id]);

  // Refresh "recently played" when the track changes.
  useEffect(() => {
    if (playingId) {
      void loadHome();
    }
  }, [playingId, loadHome]);

  useEffect(() => {
    if (!scan.scanning) {
      return;
    }
    const timer = window.setInterval(() => {
      void api.scanStatus().then(setScan).catch(() => {});
    }, 700);
    return () => window.clearInterval(timer);
  }, [scan.scanning]);

  const withBusy = useCallback(async (action: () => Promise<void>) => {
    setBusy(true);
    try {
      await action();
    } catch (reason) {
      fail(reason);
    } finally {
      setBusy(false);
    }
  }, [fail]);

  const playTrack = useCallback((track: Track, context: Track[] = []) => {
    void withBusy(async () => {
      const index = context.findIndex((item) => item.id === track.id);
      const queue = index >= 0 ? context.slice(index + 1).map((item) => item.id) : [];
      setPlayer(await api.play(track.id, queue, player.mode));
      setPlayingId(track.id);
      setNav((current) => ({ ...current, page: 'now' }));
    });
    void api.favorites().then(setFavorites).catch(() => {});
  }, [player.mode, withBusy]);

  const playAll = useCallback((tracks: Track[]) => {
    const [first, ...rest] = tracks;
    if (!first) {
      return;
    }
    void withBusy(async () => {
      setPlayer(await api.play(first.id, rest.map((item) => item.id), player.mode));
      setPlayingId(first.id);
      setNav((current) => ({ ...current, page: 'now' }));
    });
  }, [player.mode, withBusy]);

  const toggle = useCallback(() => {
    void withBusy(async () => {
      setPlayer(player.playing ? await api.pause() : await api.resume());
    });
  }, [player.playing, withBusy]);

  const next = useCallback(() => {
    void withBusy(async () => {
      setPlayer(await api.next());
    });
    void loadHome();
  }, [withBusy, loadHome]);

  const prev = useCallback(() => {
    void withBusy(async () => {
      setPlayer(await api.prev());
    });
  }, [withBusy]);

  const seek = useCallback((position: number) => {
    setPlayer((current) => ({ ...current, position }));
    void api.seek(position).catch(fail);
  }, [fail]);

  const setVolume = useCallback((volume: number) => {
    setPlayer((current) => ({ ...current, volume }));
    void api.volume(volume).catch(fail);
  }, [fail]);

  const cycleMode = useCallback(() => {
    const mode: PlaybackMode = nextMode(player.mode);
    setPlayer((current) => ({ ...current, mode }));
    void api.mode(mode).then(setPlayer).catch(fail);
  }, [player.mode, fail]);

  const setMode = useCallback((mode: PlaybackMode) => {
    setPlayer((current) => ({ ...current, mode }));
    void api.mode(mode).then(setPlayer).catch(fail);
  }, [fail]);

  const toggleFavorite = useCallback(async (kind: 'track' | 'album' | 'artist', id: string | number) => {
    const key = kind === 'track' ? 'track' : kind;
    const active = favorites.ids[key].includes(String(id));
    try {
      if (active) {
        await api.removeFavorite(kind, id);
      } else {
        await api.addFavorite(kind, id);
      }
      await loadFavorites();
      await loadHome();
    } catch (reason) {
      fail(reason);
    }
  }, [favorites.ids, loadFavorites, loadHome, fail]);

  const queueAdd = useCallback((track: Track) => {
    void withBusy(async () => {
      const result = await api.queue('add', { track_id: track.id });
      setPlayer((current) => ({ ...current, queue: result.queue }));
      announce(`${tr('addQueue')}: ${track.title}`);
    });
  }, [withBusy, tr, announce]);

  const queuePlayNext = useCallback((track: Track) => {
    void withBusy(async () => {
      const result = await api.queue('play_next', { track_id: track.id });
      setPlayer((current) => ({ ...current, queue: result.queue }));
      announce(`${tr('playNext')}: ${track.title}`);
    });
  }, [withBusy, tr, announce]);

  const queueAddMany = useCallback((tracks: Track[]) => {
    void withBusy(async () => {
      const result = await api.queue('add', { track_ids: tracks.map((item) => item.id) });
      setPlayer((current) => ({ ...current, queue: result.queue }));
      announce(`${tracks.length} ${tr('trackCount')}`);
    });
  }, [withBusy, tr, announce]);

  const queueRemove = useCallback((index: number) => {
    void withBusy(async () => {
      const result = await api.queue('remove', { index });
      setPlayer((current) => ({ ...current, queue: result.queue }));
    });
  }, [withBusy]);

  const queueReorder = useCallback((from: number, to: number) => {
    void withBusy(async () => {
      const result = await api.queue('reorder', { index: from, to });
      setPlayer((current) => ({ ...current, queue: result.queue }));
    });
  }, [withBusy]);

  const queueClear = useCallback(() => {
    void withBusy(async () => {
      const result = await api.queue('clear');
      setPlayer((current) => ({ ...current, queue: result.queue }));
    });
  }, [withBusy]);

  const startScan = useCallback(() => {
    void withBusy(async () => {
      await api.scan();
      setScan(await api.scanStatus());
      announce(tr('scanNow'));
    });
  }, [withBusy, tr, announce]);

  const cancelScan = useCallback(() => {
    void api.scanCancel().then(() => api.scanStatus()).then(setScan).catch(fail);
  }, [fail]);

  const saveSettings = useCallback((patch: Partial<SettingsMap>) => {
    void withBusy(async () => {
      const result = await api.saveSettings(patch);
      setSettings(result.settings);
      if (result.settings.language) setLangState(result.settings.language as Lang);
      announce(tr('save'));
      await refresh();
    });
  }, [withBusy, tr, announce, refresh]);

  const createPlaylist = useCallback((name: string, trackIds: string[] = []) => {
    void withBusy(async () => {
      const created = await api.createPlaylist(name, trackIds);
      setPlaylists((await api.playlists()).items);
      announce(`${created.name}`);
    });
  }, [withBusy, announce]);

  const renamePlaylist = useCallback((id: number | string, name: string) => {
    void withBusy(async () => {
      setPlaylist(await api.renamePlaylist(id, name));
      setPlaylists((await api.playlists()).items);
    });
  }, [withBusy]);

  const deletePlaylist = useCallback((id: number | string) => {
    void withBusy(async () => {
      await api.deletePlaylist(id);
      setPlaylist(null);
      setPlaylists((await api.playlists()).items);
      setNav({ page: 'playlists', tab: 'songs', id: '', q: '' });
    });
  }, [withBusy]);

  const playlistMutate = useCallback((id: number | string, action: 'add' | 'remove' | 'reorder' | 'clear', payload: Record<string, unknown> = {}) => {
    void withBusy(async () => {
      setPlaylist(await api.playlistItems(id, action, payload));
      setPlaylists((await api.playlists()).items);
    });
  }, [withBusy]);

  const addToPlaylist = useCallback((id: number | string, trackIds: string[]) => {
    void withBusy(async () => {
      await api.playlistItems(id, 'add', { track_ids: trackIds });
      setPlaylist((current) => current);
      setPlaylists((await api.playlists()).items);
      announce(`${trackIds.length} ${tr('trackCount')}`);
    });
  }, [withBusy, tr, announce]);

  const importPlaylist = useCallback((file: File, name: string) => {
    void withBusy(async () => {
      const result = await api.importPlaylist(file, name);
      setPlaylists((await api.playlists()).items);
      announce(`${result.count} ${tr('importedCount')}`);
    });
  }, [withBusy, announce, tr]);

  const setLang = useCallback((value: Lang) => {
    setLangState(value);
    void api.saveSettings({ language: value }).catch(() => {});
  }, []);

  const setTheme = useCallback((value: Theme) => {
    setThemeState(value);
    void api.saveSettings({ theme: value }).catch(() => {});
  }, []);

  const setLayout = useCallback((value: Layout) => {
    setLayoutState(value);
  }, []);

  const setQuery = useCallback((value: string) => {
    setNav((current) => ({ ...current, q: value }));
  }, []);

  const favoriteIds = useMemo(() => ({
    track: new Set(favorites.ids.track),
    album: new Set(favorites.ids.album),
    artist: new Set(favorites.ids.artist),
  }), [favorites.ids]);

  const isFavorite = useCallback((kind: 'track' | 'album' | 'artist', id: string | number) => favoriteIds[kind].has(String(id)), [favoriteIds]);

  return {
    lang, theme, layout, nav, tr,
    data: { home, songs, songTotal, albums, artists, genres, folders, playlists, favorites, settings, health, player, album, artist, playlist, lyrics, stats, searchResults, scan },
    ui: { error, setError, busy, toast, drawer, setDrawer },
    actions: {
      setLang, setTheme, setLayout, setQuery, navigate, goBack,
      refresh, loadLibrary, loadSongs, loadMoreSongs,
      playTrack, playAll, toggle, next, prev, seek, setVolume, cycleMode, setMode,
      toggleFavorite, isFavorite,
      queueAdd, queuePlayNext, queueAddMany, queueRemove, queueReorder, queueClear,
      startScan, cancelScan, saveSettings,
      createPlaylist, renamePlaylist, deletePlaylist, playlistMutate, addToPlaylist, importPlaylist,
    },
  };
}

export type Pocket = ReturnType<typeof usePocket>;
