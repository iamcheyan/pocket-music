import { Library, Play, Sparkles } from 'lucide-react';
import type { Pocket } from '../app/usePocket';
import { AlbumCard, TrackCard } from '../components/cards';
import { Cover, EmptyState, Section } from '../components/ui';

export function HomePage({ pocket }: { pocket: Pocket }) {
  const { tr, data, actions } = pocket;
  const heroTrack = data.home.recently_added[0] ?? data.songs[0];
  const hero = heroTrack ?? null;

  return (
    <>
      <div className="hero">
        <div className="hero-copy">
          <span className="eyebrow">{tr('madeFor')}</span>
          <h1>{tr('browse')}</h1>
          <p>{tr('layoutNote')}</p>
          <div className="hero-actions">
            <button
              type="button"
              className="primary-button"
              disabled={!hero}
              onClick={() => (hero ? actions.playTrack(hero, data.home.recently_added) : undefined)}
            >
              <Play size={16} fill="currentColor" /> {tr('play')}
            </button>
            <button type="button" className="secondary-button" onClick={() => actions.navigate('library')}>
              <Library size={16} /> {tr('all')}
            </button>
          </div>
        </div>
        <div className="hero-art" aria-hidden={!hero}>
          <div className="hero-disc-wrap">
            {hero ? (
              <Cover albumId={hero.album_id} title={hero.album || hero.title} artist={hero.artist} className="hero-cover" />
            ) : (
              <div className="hero-disc">
                <span />
              </div>
            )}
          </div>
          <div className="hero-caption">
            <span>{hero ? String(hero.year || '—') : 'POCKET'}</span>
            <b>
              {hero ? hero.title : 'Pocket'}
              <br />
              {hero ? hero.artist : 'Music'}
            </b>
          </div>
        </div>
      </div>

      <Section
        title={tr('recent')}
        action={
          <button type="button" className="text-button" onClick={() => actions.navigate('library', { tab: 'songs' })}>
            {tr('seeAll')} →
          </button>
        }
      >
        {data.home.recently_added.length === 0 ? (
          <EmptyState icon={<Sparkles />} title={tr('empty')} hint={tr('emptyHint')} />
        ) : (
          <div className="card-row">
            {data.home.recently_added.slice(0, 8).map((track) => (
              <TrackCard key={track.id} track={track} pocket={pocket} context={data.home.recently_added} />
            ))}
          </div>
        )}
      </Section>

      {data.home.recently_played.length > 0 && (
        <Section title={tr('played')}>
          <div className="card-row">
            {data.home.recently_played.slice(0, 8).map((track) => (
              <TrackCard key={`played-${track.id}`} track={track} pocket={pocket} context={data.home.recently_played} />
            ))}
          </div>
        </Section>
      )}

      {data.home.most_played.length > 0 && (
        <Section title={tr('popular')}>
          <div className="card-row">
            {data.home.most_played.slice(0, 8).map((track) => (
              <TrackCard key={`popular-${track.id}`} track={track} pocket={pocket} context={data.home.most_played} />
            ))}
          </div>
        </Section>
      )}

      <Section
        title={tr('favoriteTracks')}
        action={
          <button type="button" className="text-button" onClick={() => actions.navigate('library', { tab: 'favorites' })}>
            {tr('seeAll')} →
          </button>
        }
      >
        {data.home.favorite_tracks.length === 0 ? (
          <p className="muted-note">{tr('emptyHint')}</p>
        ) : (
          <div className="card-row">
            {data.home.favorite_tracks.slice(0, 8).map((track) => (
              <TrackCard key={`fav-${track.id}`} track={track} pocket={pocket} context={data.home.favorite_tracks} />
            ))}
          </div>
        )}
      </Section>

      <Section
        title={tr('favoriteAlbums')}
        action={
          <button type="button" className="text-button" onClick={() => actions.navigate('library', { tab: 'favorites' })}>
            {tr('seeAll')} →
          </button>
        }
      >
        {data.home.favorite_albums.length === 0 ? (
          <p className="muted-note">{tr('emptyHint')}</p>
        ) : (
          <div className="card-row">
            {data.home.favorite_albums.slice(0, 8).map((album) => (
              <AlbumCard key={`fav-album-${album.id}`} album={album} pocket={pocket} onOpen={(value) => actions.navigate('album', { id: String(value.id) })} />
            ))}
          </div>
        )}
      </Section>
    </>
  );
}
