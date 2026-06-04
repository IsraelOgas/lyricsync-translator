import React, { useState } from 'react';
import { Settings } from 'lucide-react';
import { usePlayerState } from './hooks/usePlayerState';
import { useSettings } from './hooks/useSettings';
import { useCoverColor } from './hooks/useCoverColor';
import { NowPlayingBar } from './components/NowPlayingBar';
import { LyricsViewer } from './components/LyricsViewer';
import { SettingsPanel } from './components/SettingsPanel';
import { PlayerBar } from './components/PlayerBar';
import ErrorBoundary from './components/ErrorBoundary';
import styles from './App.module.css';

const App: React.FC = () => {
  const { track, status, positionMs, lines, notFound, fetchingLyrics, translating, paused, lyricsError, offsetMs, handleRetryLyrics, handleUpdateOffset } = usePlayerState();
  const { settings, updateSetting } = useSettings();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const coverColor = useCoverColor(track?.cover_art_url);

  return (
    <ErrorBoundary>
      <div
        className={styles.app}
        style={settings.cinemaMode && coverColor ? { backgroundColor: coverColor } : undefined}
      >
        <SettingsPanel
          isOpen={settingsOpen}
          settings={settings}
          onUpdateSetting={updateSetting}
          onClose={() => setSettingsOpen(false)}
          offsetMs={offsetMs}
          onUpdateOffset={handleUpdateOffset}
        />
        <NowPlayingBar track={track} status={status} />
        <LyricsViewer
          lines={lines}
          positionMs={positionMs}
          offsetMs={offsetMs}
          paused={paused}
          notFound={notFound}
          fetchingLyrics={fetchingLyrics}
          translating={translating}
          lyricsError={lyricsError}
          onRetry={handleRetryLyrics}
          showRomanization={settings.showRomanization}
        />
        <PlayerBar track={track} status={status} positionMs={positionMs} />
        <button
          className={styles.settingsFab}
          onClick={() => setSettingsOpen(true)}
          aria-label="Open settings"
          title="Settings"
        >
          <Settings size={20} />
        </button>
      </div>
    </ErrorBoundary>
  );
};

export default App;
