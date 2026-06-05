import React from 'react';
import { Music, Library } from 'lucide-react';
import type { TrackInfo } from '../types';
import styles from './NowPlayingBar.module.css';

interface Props {
  track: TrackInfo | null;
  status: string;
  view: 'now-playing' | 'saved-songs';
  onViewChange: (view: 'now-playing' | 'saved-songs') => void;
}

export const NowPlayingBar: React.FC<Props> = ({ track, status, view, onViewChange }) => {
  const tabs = (
    <div className={styles.viewTabs}>
      <button
        className={`${styles.viewTab} ${view === 'now-playing' ? styles.viewTabActive : ''}`}
        onClick={() => onViewChange('now-playing')}
        aria-label="Now Playing view"
      >
        <Music size={15} />
        Now Playing
      </button>
      <button
        className={`${styles.viewTab} ${view === 'saved-songs' ? styles.viewTabActive : ''}`}
        onClick={() => onViewChange('saved-songs')}
        aria-label="Saved Songs view"
      >
        <Library size={15} />
        Saved Songs
      </button>
    </div>
  );

  if (!track) {
    if (status !== 'no_player') {
      return (
        <div className={styles.bar}>
          <div className={styles.skeletonBar} />
          {tabs}
        </div>
      );
    }
    return (
      <div className={styles.bar}>
        <span className={styles.noTrack}>No track playing</span>
        {tabs}
        <span className={styles.status}>{status}</span>
      </div>
    );
  }

  return (
    <div className={styles.bar}>
      {track.cover_art_url && (
        <img
          className={styles.cover}
          src={track.cover_art_url}
          alt={`${track.artist} - ${track.title} cover`}
          onError={e => { (e.target as HTMLImageElement).style.display = 'none'; }}
        />
      )}
      <div className={styles.trackInfo}>
        <span className={styles.title} title={track.title}>{track.title}</span>
        <span className={styles.artist} title={track.artist}>{track.artist}</span>
        {track.album && <span className={styles.album}>{track.album}</span>}
      </div>
      {tabs}
      <span className={styles.status}>{status}</span>
    </div>
  );
};
