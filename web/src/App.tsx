import React, { useCallback, useState } from 'react';
import { HelpCircle, Settings } from 'lucide-react';
import { usePlayerState } from './hooks/usePlayerState';
import { useSettings } from './hooks/useSettings';
import { useCoverColor } from './hooks/useCoverColor';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { NowPlayingBar } from './components/NowPlayingBar';
import { LyricsViewer } from './components/LyricsViewer';
import { SettingsPanel } from './components/SettingsPanel';
import { PlayerBar } from './components/PlayerBar';
import { HelpDialog } from './components/HelpDialog';
import ErrorBoundary from './components/ErrorBoundary';
import styles from './App.module.css';

const App: React.FC = () => {
  const { track, status, positionMs, lines, notFound, fetchingLyrics, translating, paused, lyricsError, offsetMs, handleTogglePlayPause, handleRetryLyrics, handleUpdateOffset } = usePlayerState();
  const { settings, updateSetting } = useSettings();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const coverColor = useCoverColor(track?.cover_art_url);

  const handleOpenHelp = useCallback(() => setHelpOpen(true), []);

  useKeyboardShortcuts({
    settingsOpen,
    setSettingsOpen,
    settings,
    updateSetting,
    positionMs,
    handleTogglePlayPause,
    onOpenHelp: handleOpenHelp,
  });

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
        <HelpDialog open={helpOpen} onClose={() => setHelpOpen(false)} />
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
          className={styles.helpFab}
          onClick={() => setHelpOpen(true)}
          aria-label="Keyboard shortcuts"
          title="Keyboard shortcuts (?)"
        >
          <HelpCircle size={20} />
        </button>
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
