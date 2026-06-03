import React, { useState } from 'react';
import { usePlayerState } from './hooks/usePlayerState';
import { useSettings } from './hooks/useSettings';
import { NowPlayingBar } from './components/NowPlayingBar';
import { LyricsViewer } from './components/LyricsViewer';
import { SettingsPanel } from './components/SettingsPanel';
import ErrorBoundary from './components/ErrorBoundary';
import styles from './App.module.css';

const App: React.FC = () => {
  const { track, status, positionMs, lines, notFound, fetchingLyrics, translating, paused, lyricsError, handleTogglePlayPause, handleRetryLyrics } = usePlayerState();
  const { settings, updateSetting } = useSettings();
  const [settingsOpen, setSettingsOpen] = useState(false);

  return (
    <ErrorBoundary>
      <div className={styles.app}>
        <SettingsPanel
          isOpen={settingsOpen}
          settings={settings}
          onUpdateSetting={updateSetting}
          onClose={() => setSettingsOpen(false)}
        />
        <NowPlayingBar track={track} status={status} />
        <LyricsViewer
          lines={lines}
          positionMs={positionMs}
          paused={paused}
          notFound={notFound}
          fetchingLyrics={fetchingLyrics}
          translating={translating}
          lyricsError={lyricsError}
          onRetry={handleRetryLyrics}
          showRomanization={settings.showRomanization}
        />
        <footer className={styles.footer}>
          <button
            className={styles.btn}
            onClick={handleTogglePlayPause}
            aria-label={status === 'playing' ? 'Pause' : 'Play'}
            title={status === 'playing' ? 'Pause' : 'Play'}
          >
            {status === 'playing' ? '⏸' : '▶'}
          </button>
          <button
            className={styles.btn}
            onClick={() => setSettingsOpen(true)}
            aria-label="Open settings"
            title="Settings"
          >
            ⚙
          </button>
        </footer>
      </div>
    </ErrorBoundary>
  );
};

export default App;
