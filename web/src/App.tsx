import React, { useCallback, useEffect, useRef, useState } from 'react';
import { usePlayerState } from './hooks/usePlayerState';
import { useSettings } from './hooks/useSettings';
import { useCoverColor } from './hooks/useCoverColor';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { NowPlayingBar } from './components/NowPlayingBar';
import { LyricsViewer } from './components/LyricsViewer';
import { SettingsPanel } from './components/SettingsPanel';
import { PlayerBar } from './components/PlayerBar';
import { HelpDialog } from './components/HelpDialog';
import { SavedSongsView } from './components/SavedSongsView';
import ErrorBoundary from './components/ErrorBoundary';
import { apiUrl } from './api';
import styles from './App.module.css';

const App: React.FC = () => {
  const { track, status, positionMs, lines, notFound, fetchingLyrics, translating, paused, lyricsError, offsetMs, handleTogglePlayPause, handleRetryLyrics, handleUpdateOffset } = usePlayerState();
  const { settings, updateSetting } = useSettings();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [view, setView] = useState<'now-playing' | 'saved-songs'>('now-playing');
  const [deepseekApiKey, setDeepseekApiKey] = useState('');
  const coverColor = useCoverColor(track?.cover_art_url);

  const hasFetchedRef = useRef(false);

  // Fetch config to read current API key. Runs on mount and retries when
  // settings opens (backend might not be ready yet on mount in Wails dev).
  useEffect(() => {
    if (!settingsOpen && hasFetchedRef.current) return;
    const controller = new AbortController();
    fetch(apiUrl('/api/config'), { signal: controller.signal })
      .then(res => res.json())
      .then(data => {
        hasFetchedRef.current = true;
        const key = data?.translation?.deepseek?.api_key || '';
        if (key) {
          setDeepseekApiKey(prev => prev || key);
        }
      })
      .catch(err => {
        if (err.name !== 'AbortError') {
          hasFetchedRef.current = false; // allow retry on next open
        }
      });
    return () => controller.abort();
  }, [settingsOpen]);

  const handleOpenHelp = useCallback(() => setHelpOpen(true), []);

  const handleUpdateDeepseekKey = useCallback((apiKey: string) => {
    fetch(apiUrl('/api/config/provider'), {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'deepseek', api_key: apiKey }),
    }).catch(() => {});

    // Update local state so the UI shows the masked key immediately.
    setDeepseekApiKey(apiKey ? '••••••••' : '');
  }, []);

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
          deepseekApiKey={deepseekApiKey}
          onUpdateDeepseekKey={handleUpdateDeepseekKey}
        />
        <HelpDialog open={helpOpen} onClose={() => setHelpOpen(false)} />
        <NowPlayingBar track={track} status={status} view={view} onViewChange={setView} />
        {view === 'now-playing' ? (
          <>
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
            <PlayerBar
              track={track}
              status={status}
              positionMs={positionMs}
              onOpenSettings={() => setSettingsOpen(true)}
              onOpenHelp={() => setHelpOpen(true)}
            />
          </>
        ) : (
          <SavedSongsView showRomanization={settings.showRomanization} />
        )}

      </div>
    </ErrorBoundary>
  );
};

export default App;
