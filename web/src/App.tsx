import React from 'react';
import { usePlayerState } from './hooks/usePlayerState';
import { NowPlayingBar } from './components/NowPlayingBar';
import { LyricsViewer } from './components/LyricsViewer';
import ErrorBoundary from './components/ErrorBoundary';
import styles from './App.module.css';

const App: React.FC = () => {
  const { track, status, positionMs, lines, notFound, fetchingLyrics, translating, paused, handleTogglePlayPause } = usePlayerState();

  return (
    <ErrorBoundary>
      <div className={styles.app}>
        <NowPlayingBar track={track} status={status} />
        <LyricsViewer lines={lines} positionMs={positionMs} paused={paused} notFound={notFound} fetchingLyrics={fetchingLyrics} translating={translating} />
        <footer className={styles.footer}>
          <button
            className={styles.btn}
            onClick={handleTogglePlayPause}
            aria-label={status === 'playing' ? 'Pause' : 'Play'}
            title={status === 'playing' ? 'Pause' : 'Play'}
          >
            {status === 'playing' ? '⏸' : '▶'}
          </button>
        </footer>
      </div>
    </ErrorBoundary>
  );
};

export default App;
