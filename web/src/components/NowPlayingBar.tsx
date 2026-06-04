import React from 'react';
import type { TrackInfo } from '../types';
import styles from './NowPlayingBar.module.css';

interface Props {
  track: TrackInfo | null;
  status: string;
}

export const NowPlayingBar: React.FC<Props> = ({ track, status }) => {
  if (!track) {
    if (status !== 'no_player') {
      return (
        <div className={styles.bar}>
          <div className={styles.skeletonBar} />
        </div>
      );
    }
    return (
      <div className={styles.bar}>
        <span className={styles.noTrack}>No track playing</span>
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
      <span className={styles.status}>{status}</span>
    </div>
  );
};
