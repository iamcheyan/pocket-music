import { useEffect, useRef } from 'react';
import {
  AudioLines, BarChart3, Disc3, Heart, Home, Library, ListMusic, Menu, Pause, Play, Search,
  Settings as SettingsIcon, SkipBack, SkipForward, Volume2, X,
} from 'lucide-react';
import { usePocket, type Page } from './app/usePocket';
import { modeLabelKey } from './lib/mode';
import { formatDuration } from './lib/format';
import { clampVolume } from './lib/queue';
import { Cover, Slider } from './components/ui';
import { HomePage } from './pages/Home';
import { AlbumPage, ArtistPage, LibraryPage } from './pages/Library';
import { NowPlayingPage, QueuePage } from './pages/Playback';
import { PlaylistPage, PlaylistsPage } from './pages/Playlists';
import { SearchPage, StatsPage } from './pages/Search';
import { SettingsPage } from './pages/Settings';

const NAV: { page: Page; key: string; icon: React.ReactNode }[] = [
  { page: 'home', key: 'home', icon: <Home /> },
  { page: 'library', key: 'library', icon: <Library /> },
  { page: 'now', key: 'now', icon: <AudioLines /> },
  { page: 'queue', key: 'queue', icon: <ListMusic /> },
  { page: 'playlists', key: 'playlists', icon: <Disc3 /> },
  { page: 'search', key: 'search', icon: <Search /> },
  { page: 'stats', key: 'stats', icon: <BarChart3 /> },
  { page: 'settings', key: 'settings', icon: <SettingsIcon /> },
];

const MOBILE_NAV: Page[] = ['home', 'library', 'now', 'queue', 'settings'];

export default function App() {
  const pocket = usePocket();
  const { tr, nav, actions, data, ui } = pocket;
  const { player, health } = data;
  const gamepadHeld = useRef<Record<string, boolean>>({});
  const toggleRef = useRef(actions.toggle);
  toggleRef.current = actions.toggle;

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const editing = target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable);
      if (editing) {
        if (event.key === 'Escape') (target as HTMLInputElement).blur();
        return;
      }
      if (event.key === ' ') {
        event.preventDefault();
        actions.toggle();
      } else if (event.key === 'ArrowRight') {
        event.preventDefault();
        actions.next();
      } else if (event.key === 'ArrowLeft') {
        event.preventDefault();
        actions.prev();
      } else if (event.key === 'ArrowUp') {
        event.preventDefault();
        actions.setVolume(clampVolume(player.volume + 5));
      } else if (event.key === 'ArrowDown') {
        event.preventDefault();
        actions.setVolume(clampVolume(player.volume - 5));
      } else if (event.key === '/') {
        event.preventDefault();
        actions.navigate('search', { q: nav.q });
        window.setTimeout(() => document.getElementById('global-search')?.focus(), 30);
      } else if (event.key === 'Escape') {
        actions.navigate('home');
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [actions, player.volume, nav.q]);

  // Gamepad: edge-triggered so a held button fires once.
  useEffect(() => {
    let frame = 0;
    const held = gamepadHeld.current;
    const pressed = (name: string, down: boolean) => {
      if (!down) {
        held[name] = false;
        return false;
      }
      if (held[name]) return false;
      held[name] = true;
      return true;
    };
    const poll = () => {
      const pads = navigator.getGamepads?.() ?? [];
      for (const pad of pads) {
        if (!pad) continue;
        if (pressed('toggle', Boolean(pad.buttons[0]?.pressed))) toggleRef.current();
        if (pressed('prev', Boolean(pad.buttons[4]?.pressed))) actions.prev();
        if (pressed('next', Boolean(pad.buttons[5]?.pressed))) actions.next();
        if (pressed('up', Boolean(pad.buttons[12]?.pressed))) actions.setVolume(clampVolume(player.volume + 5));
        if (pressed('down', Boolean(pad.buttons[13]?.pressed))) actions.setVolume(clampVolume(player.volume - 5));
        if (pressed('hd', (pad.axes[0] ?? 0) < -0.7)) actions.navigate('home');
        if (pressed('hr', (pad.axes[0] ?? 0) > 0.7)) actions.navigate('now');
      }
      frame = requestAnimationFrame(poll);
    };
    frame = requestAnimationFrame(poll);
    return () => cancelAnimationFrame(frame);
  }, [actions, player.volume]);

  const content = () => {
    switch (nav.page) {
      case 'library':
        return <LibraryPage pocket={pocket} />;
      case 'album':
        return <AlbumPage pocket={pocket} />;
      case 'artist':
        return <ArtistPage pocket={pocket} />;
      case 'now':
        return <NowPlayingPage pocket={pocket} />;
      case 'queue':
        return <QueuePage pocket={pocket} />;
      case 'playlists':
        return <PlaylistsPage pocket={pocket} />;
      case 'playlist':
        return <PlaylistPage pocket={pocket} />;
      case 'search':
        return <SearchPage pocket={pocket} />;
      case 'stats':
        return <StatsPage pocket={pocket} />;
      case 'settings':
        return <SettingsPage pocket={pocket} />;
      default:
        return <HomePage pocket={pocket} />;
    }
  };

  const canGoBack = ['album', 'artist', 'playlist'].includes(nav.page);

  return (
    <div className={`app-shell layout-${pocket.layout}`}>
      <aside className={`sidebar ${ui.drawer ? 'drawer-open' : ''}`} aria-label={tr('menu')}>
        <button type="button" className="brand" onClick={() => actions.navigate('home')}>
          <span className="brand-mark">
            <AudioLines />
          </span>
          <span>
            pocket<span className="brand-light">music</span>
          </span>
        </button>
        <div className="nav-label">LISTEN</div>
        <nav>
          {NAV.map((item) => (
            <button
              key={item.page}
              type="button"
              onClick={() => actions.navigate(item.page)}
              className={`nav-item ${nav.page === item.page || (nav.page === 'playlist' && item.page === 'playlists') ? 'selected' : ''}`}
            >
              <span className="nav-icon">{item.icon}</span>
              <span>{tr(item.key)}</span>
              {item.page === 'queue' && player.queue.length > 0 && <small>{player.queue.length}</small>}
            </button>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <div className="sidebar-note">
            <span className={`pulse ${health ? '' : 'offline'}`} />
            <span>
              {health?.provider ?? 'musicd'} <i>{health ? tr('connected') : tr('offline')}</i>
            </span>
          </div>
          <button type="button" className="user-chip" onClick={() => actions.navigate('settings')}>
            <div className="avatar">PM</div>
            <span>
              <b>{health?.mock ? tr('mockLibrary') : tr('realLibrary')}</b>
              <small>{pocket.layout} · {formatDuration(player.duration)}</small>
            </span>
            <SettingsIcon size={15} />
          </button>
        </div>
      </aside>

      <main className="main-area">
        <header className="topbar">
          <div className="mobile-brand">
            <button type="button" className="icon-button" aria-label={tr('menu')} aria-expanded={ui.drawer} onClick={() => ui.setDrawer(!ui.drawer)}>
              {ui.drawer ? <X size={18} /> : <Menu size={18} />}
            </button>
            <button type="button" className="brand-mini" onClick={() => actions.navigate('home')}>
              <span className="brand-mark">
                <AudioLines />
              </span>
              <b>pocketmusic</b>
            </button>
          </div>
          <div className="breadcrumbs">
            <span>Pocket Music</span>
            <span>/</span>
            <b>{tr(nav.page)}</b>
            {nav.page === 'album' && data.album && <span>/ {data.album.title}</span>}
            {nav.page === 'artist' && data.artist && <span>/ {data.artist.name}</span>}
          </div>
          <div className="top-actions">
            {canGoBack && (
              <button type="button" className="secondary-button compact" onClick={actions.goBack}>
                ← {tr('back')}
              </button>
            )}
            <button type="button" className="quick-search" onClick={() => actions.navigate('search', { q: nav.q })}>
              <Search size={15} />
              <span>{tr('search')}</span>
              <kbd>/</kbd>
            </button>
            <button
              type="button"
              className="theme-toggle"
              aria-label={tr('toggleTheme')}
              onClick={() => actions.setTheme(pocket.theme === 'dark' ? 'light' : 'dark')}
            >
              {pocket.theme === 'dark' ? '☼' : '◐'}
            </button>
          </div>
        </header>

        <div className="view-content">
          {ui.error && (
            <div className="error-banner" role="alert">
              <span>{ui.error}</span>
              <button type="button" onClick={() => ui.setError('')} aria-label={tr('close')}>
                ×
              </button>
            </div>
          )}
          {content()}
        </div>
      </main>

      <footer className="player-bar">
        <button type="button" className="mini-art" onClick={() => actions.navigate('now')} aria-label={tr('nowPlaying')}>
          <Cover albumId={player.track?.album_id} title={player.track?.album || tr('noTrack')} className="cover-thumb" rounded />
        </button>
        <button type="button" className="mini-track" onClick={() => actions.navigate('now')}>
          <b>{player.track?.title || 'Pocket Music'}</b>
          <span>{player.track ? player.track.artist : tr('noTrack')}</span>
        </button>
        <div className="mini-controls">
          <button type="button" className="icon-button" aria-label={tr('prev')} onClick={actions.prev}>
            <SkipBack size={17} />
          </button>
          <button type="button" className="mini-play" aria-label={player.playing ? 'Pause' : tr('play')} onClick={actions.toggle}>
            {player.playing ? <Pause size={17} fill="currentColor" /> : <Play size={17} fill="currentColor" />}
          </button>
          <button type="button" className="icon-button" aria-label={tr('next')} onClick={actions.next}>
            <SkipForward size={17} />
          </button>
        </div>
        <div className="mini-progress">
          <div>
            <span>{formatDuration(player.position)}</span>
            <span>{formatDuration(player.duration)}</span>
          </div>
          <Slider
            value={Math.min(player.position, player.duration || 0)}
            max={Math.max(player.duration, 1)}
            onChange={actions.seek}
            label={tr('playbackProgress')}
            step={5}
          />
        </div>
        <button type="button" className={`icon-button mini-mode ${player.mode !== 'normal' ? 'active' : ''}`} aria-label={tr(modeLabelKey(player.mode))} onClick={actions.cycleMode}>
          {player.mode === 'shuffle' ? '⤨' : player.mode === 'repeat_one' ? '↻1' : '↻'}
        </button>
        <div className="mini-volume">
          <Volume2 size={14} />
          <Slider value={player.volume} max={100} onChange={actions.setVolume} label={tr('volume')} step={5} />
        </div>
        <button
          type="button"
          className={`icon-button mini-heart ${player.track && actions.isFavorite('track', player.track.id) ? 'is-favourite' : ''}`}
          aria-label={tr('favoriteAdd')}
          disabled={!player.track}
          onClick={() => (player.track ? void actions.toggleFavorite('track', player.track.id) : undefined)}
        >
          <Heart size={16} fill={player.track && actions.isFavorite('track', player.track.id) ? 'currentColor' : 'none'} />
        </button>
        <button type="button" className="icon-button mini-queue" aria-label={tr('queue')} onClick={() => actions.navigate('queue')}>
          <ListMusic size={19} />
        </button>
      </footer>

      <nav className="mobile-nav" aria-label={tr('menu')}>
        {NAV.filter((item) => MOBILE_NAV.includes(item.page)).map((item) => (
          <button key={item.page} type="button" className={nav.page === item.page ? 'selected' : ''} onClick={() => actions.navigate(item.page)}>
            {item.icon}
            <span>{tr(item.key)}</span>
          </button>
        ))}
      </nav>

      {ui.toast && (
        <div className="toast" role="status">
          {ui.toast}
        </div>
      )}
    </div>
  );
}
