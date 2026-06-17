import React, { useCallback, useEffect, useRef, useState } from 'react';
import { flushSync } from 'react-dom';
import { usePlayerState } from './hooks/usePlayerState';
import { useSettings } from './hooks/useSettings';
import { useCoverColor } from './hooks/useCoverColor';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { NowPlayingBar } from './components/NowPlayingBar';
import { LyricsViewer } from './components/LyricsViewer';
import { SettingsPanel } from './components/SettingsPanel';
import { PlayerBar } from './components/PlayerBar';
import { CinemaProgressBar } from './components/CinemaProgressBar';
import { CinemaParticles } from './components/CinemaParticles';
import { HelpDialog } from './components/HelpDialog';
import { SavedSongsView } from './components/SavedSongsView';
import ErrorBoundary from './components/ErrorBoundary';
import { apiUrl } from './api';
import type { Settings } from './types';
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

  // Wrap updateSetting to use View Transitions API when toggling cinema mode.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleUpdateSetting = useCallback((key: keyof Settings, value: any) => {
    if (key === 'cinemaMode' && 'startViewTransition' in document) {
      document.startViewTransition(() => {
        flushSync(() => {
          updateSetting(key, value);
        });
      });
    } else {
      updateSetting(key, value);
    }
  }, [updateSetting]);

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

  // Calculate song progress for hue shift (0..1)
  const songProgress = track?.duration_ms ? Math.min(1, Math.max(0, positionMs / track.duration_ms)) : 0;
  // Detect near end for fade-to-black (last 3 seconds)
  const nearEnd = track?.duration_ms ? positionMs > track.duration_ms - 3000 && positionMs < track.duration_ms : false;
  // Hue shift: 0deg at start → 30deg at end (subtle warm shift)
  const hueShift = songProgress * 30;

  return (
    <ErrorBoundary>
      <div
        className={`${styles.app} ${nearEnd ? styles.nearEnd : ''}`}
        style={settings.cinemaMode && coverColor ? {
          backgroundColor: coverColor,
          filter: `hue-rotate(${hueShift}deg)`,
        } : undefined}
      >
        {/* Animated karaoke background — only in cinema mode */}
        {settings.cinemaMode && (
          <div className={styles.cinemaBg}>
            {/* Blurred cover art as background layer */}
            {track?.cover_art_url && (
              <img
                className={styles.cinemaCoverBg}
                src={track.cover_art_url}
                alt=""
                onError={e => { (e.target as HTMLImageElement).style.display = 'none'; }}
              />
            )}
            {/* Floating orbs with CSS-only pulse — togglable for performance */}
            {settings.cinemaOrbs && coverColor && (
              <>
                <div
                  className={`${styles.cinemaOrb} ${styles.cinemaOrb1}`}
                  style={{ backgroundColor: coverColor }}
                />
                <div
                  className={`${styles.cinemaOrb} ${styles.cinemaOrb2}`}
                  style={{ backgroundColor: coverColor }}
                />
                <div
                  className={`${styles.cinemaOrb} ${styles.cinemaOrb3}`}
                  style={{ backgroundColor: coverColor }}
                />
              </>
            )}
            {/* Floating particles — togglable for performance */}
            {settings.cinemaOrbs && (
              <CinemaParticles color={coverColor || 'rgba(255,255,255,0.3)'} />
            )}
            <div className={styles.cinemaVignette} />
          </div>
        )}

        <SettingsPanel
          isOpen={settingsOpen}
          settings={settings}
          onUpdateSetting={handleUpdateSetting}
          onClose={() => setSettingsOpen(false)}
          offsetMs={offsetMs}
          onUpdateOffset={handleUpdateOffset}
          deepseekApiKey={deepseekApiKey}
          onUpdateDeepseekKey={handleUpdateDeepseekKey}
        />
        <HelpDialog open={helpOpen} onClose={() => setHelpOpen(false)} />
        {!settings.cinemaMode && <NowPlayingBar track={track} status={status} view={view} onViewChange={setView} />}

        {/* Floating track info — only in cinema mode */}
        {settings.cinemaMode && track && (
          <div className={styles.cinemaTrackInfo}>
            {track.cover_art_url && (
              <img
                className={styles.cinemaCover}
                src={track.cover_art_url}
                alt=""
                onError={e => { (e.target as HTMLImageElement).style.display = 'none'; }}
              />
            )}
            <div className={styles.cinemaMeta}>
              <span className={styles.cinemaTitle}>{track.title}</span>
              <span className={styles.cinemaArtist}>{track.artist}</span>
              {track.album && <span className={styles.cinemaAlbum}>{track.album}</span>}
            </div>
          </div>
        )}
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
            {!settings.cinemaMode && <PlayerBar
              track={track}
              status={status}
              positionMs={positionMs}
              onOpenSettings={() => setSettingsOpen(true)}
              onOpenHelp={() => setHelpOpen(true)}
            />}
          </>
        ) : (
          !settings.cinemaMode && <SavedSongsView showRomanization={settings.showRomanization} />
        )}

        {/* Cinema mode: full-width progress bar at bottom */}
        {settings.cinemaMode && (
          <CinemaProgressBar track={track} positionMs={positionMs} />
        )}
      </div>
    </ErrorBoundary>
  );
};

export default App;
