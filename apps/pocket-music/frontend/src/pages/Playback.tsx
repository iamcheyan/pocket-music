import { useMemo } from 'react';
import {
  Heart, ListMusic, Music2, Pause, Play, Repeat, Repeat1, Shuffle, SkipBack, SkipForward, Volume2,
} from 'lucide-react';
import type { Pocket } from '../app/usePocket';
import type { PlaybackMode } from '../app/types';
import { formatDuration } from '../lib/format';
import { modeLabelKey } from '../lib/mode';
import { Cover, EmptyState, Slider } from '../components/ui';
import { TrackList } from '../components/TrackList';
import { TrackRowCompact } from '../components/cards';

const MODE_ICON: Record<PlaybackMode, React.ReactNode> = {
  normal: <Repeat size={15} />,
  repeat_all: <Repeat size={15} />,
  repeat_one: <Repeat1 size={15} />,
  shuffle: <Shuffle size={15} />,
};

export function NowPlayingPage({ pocket }: { pocket: Pocket }) {
  const { tr, data, actions } = pocket;
  const { player, lyrics } = data;
  const track = player.track;

  const activeLine = useMemo(() => {
    if (!lyrics?.lines?.length) return -1;
    let index = -1;
    for (let i = 0; i < lyrics.lines.length; i += 1) {
      if (lyrics.lines[i].time <= player.position + 0.15) index = i;
    }
    return index;
  }, [lyrics, player.position]);

  const favourite = track ? actions.isFavorite('track', track.id) : false;

  if (!track) {
    return (
      <div className="now-empty">
        <EmptyState icon={<Music2 />} title={tr('noTrack')} hint={tr('emptyHint')} />
      </div>
    );
  }

  return (
    <div className="now-page">
      <div className="now-art">
        <Cover albumId={track.album_id} title={track.album || track.title} artist={track.artist} className="art-large" />
        <div className="now-meta-line">
          <span>{track.genre || tr('music')}</span>
          <span>{String(track.year || '—')}</span>
        </div>
      </div>

      <div className="now-detail">
        <span className="eyebrow">{tr('nowPlaying')}</span>
        <h1>{track.title}</h1>
        <button type="button" className="detail-artist" onClick={() => actions.navigate('artist', { id: String(track.artist_id) })}>
          {track.artist || tr('unknownArtist')}
        </button>
        <button type="button" className="detail-album" onClick={() => actions.navigate('album', { id: String(track.album_id) })}>
          {track.album || tr('unknownAlbum')}
        </button>

        <div className="seek">
          <span>{formatDuration(player.position)}</span>
          <Slider
            value={Math.min(player.position, player.duration || 0)}
            max={Math.max(player.duration, 1)}
            onChange={actions.seek}
            label={tr('seek')}
            step={5}
            valueText={`${formatDuration(player.position)} / ${formatDuration(player.duration)}`}
          />
          <span>{formatDuration(player.duration)}</span>
        </div>

        <div className="transport">
          <button type="button" className="icon-button" aria-label={tr('prev')} onClick={actions.prev}>
            <SkipBack size={18} />
          </button>
          <button type="button" className="play-button" aria-label={player.playing ? 'Pause' : tr('play')} onClick={actions.toggle}>
            {player.playing ? <Pause size={20} fill="currentColor" /> : <Play size={20} fill="currentColor" />}
          </button>
          <button type="button" className="icon-button" aria-label={tr('next')} onClick={actions.next}>
            <SkipForward size={18} />
          </button>
        </div>

        <div className="now-tools">
          <button
            type="button"
            className={`mode-button ${player.mode !== 'normal' ? 'active' : ''}`}
            aria-label={`${tr('mode')}: ${tr(modeLabelKey(player.mode))}`}
            onClick={actions.cycleMode}
          >
            {MODE_ICON[player.mode]}
            <span>{tr(modeLabelKey(player.mode))}</span>
          </button>
          <button
            type="button"
            className={`icon-button heart-large ${favourite ? 'is-favourite' : ''}`}
            aria-label={favourite ? tr('favoriteRemove') : tr('favoriteAdd')}
            aria-pressed={favourite}
            onClick={() => void actions.toggleFavorite('track', track.id)}
          >
            <Heart size={17} fill={favourite ? 'currentColor' : 'none'} />
          </button>
          <label className="volume-control">
            <Volume2 size={15} />
            <Slider value={player.volume} max={100} onChange={actions.setVolume} label={tr('volume')} step={5} valueText={`${Math.round(player.volume)}%`} />
          </label>
          <button type="button" className="icon-button" aria-label={tr('queue')} onClick={() => actions.navigate('queue')}>
            <ListMusic size={17} />
          </button>
        </div>

        <div className="lyrics-panel" aria-label={tr('lyrics')}>
          <div className="lyrics-head">
            <span className="eyebrow">{tr('lyrics')}</span>
            {lyrics?.synced && <small>{lyrics.lines.length}</small>}
          </div>
          {lyrics && lyrics.lines.length > 0 ? (
            <ol className="lyrics-lines">
              {lyrics.lines.map((line, index) => (
                <li key={`${line.time}-${index}`} className={index === activeLine ? 'active' : ''} aria-current={index === activeLine}>
                  {line.text}
                </li>
              ))}
            </ol>
          ) : (
            <p className="muted-note">{tr('noLyrics')}</p>
          )}
        </div>

        {player.queue.length > 0 && (
          <div className="upnext">
            <div className="lyrics-head">
              <span className="eyebrow">{tr('upNext')}</span>
              <button type="button" className="text-button" onClick={() => actions.navigate('queue')}>
                {tr('queue')} →
              </button>
            </div>
            {player.queue.slice(0, 4).map((item, index) => (
              <TrackRowCompact key={`${item.id}-${index}`} track={item} pocket={pocket} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export function QueuePage({ pocket }: { pocket: Pocket }) {
  const { tr, data, actions } = pocket;
  const { player } = data;
  const total = player.queue.reduce((sum, track) => sum + (track.duration || 0), 0);

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="eyebrow">{tr('upNext')}</span>
          <h1>{tr('queue')}</h1>
          <p className="muted-note">
            {player.queue.length} {tr('trackCount')} · {Math.round(total / 60)} {tr('minute')}
          </p>
        </div>
        <div className="page-actions">
          <button type="button" className="secondary-button" onClick={actions.queueClear} disabled={player.queue.length === 0}>
            {tr('clear')}
          </button>
        </div>
      </div>

      {player.track && (
        <div className="queue-current">
          <Cover albumId={player.track.album_id} title={player.track.album} className="queue-current-cover" />
          <div>
            <span className="eyebrow">{tr('nowPlaying')}</span>
            <b>{player.track.title}</b>
            <span>{player.track.artist}</span>
          </div>
          <span className="queue-current-time">{formatDuration(player.position)}</span>
        </div>
      )}

      <TrackList
        tracks={player.queue}
        pocket={pocket}
        variant="queue"
        onRemove={(index) => actions.queueRemove(index)}
        onReorder={(from, to) => actions.queueReorder(from, to)}
        emptyTitle={tr('queueEmpty')}
        emptyHint={tr('queueEmptyHint')}
      />
    </>
  );
}
