import { useState, useEffect, useCallback } from 'react';
import { Settings, DEFAULT_SETTINGS } from '../types';

function loadFromStorage(): Settings {
  try {
    const raw = localStorage.getItem('lyricsync-settings');
    if (!raw) return DEFAULT_SETTINGS;
    const parsed = JSON.parse(raw);
    return {
      fontSize: typeof parsed.fontSize === 'number' ? parsed.fontSize : DEFAULT_SETTINGS.fontSize,
      showRomanization: typeof parsed.showRomanization === 'boolean' ? parsed.showRomanization : DEFAULT_SETTINGS.showRomanization,
    };
  } catch {
    return DEFAULT_SETTINGS;
  }
}

function saveToStorage(settings: Settings): void {
  try {
    localStorage.setItem('lyricsync-settings', JSON.stringify(settings));
  } catch (err) {
    // QuotaExceededError or unavailable — silently skip persistence.
    // Settings live in-memory for the rest of the session.
    if (err instanceof DOMException && (err.name === 'QuotaExceededError' || err.code === 22)) {
      return;
    }
    // Other storage errors (e.g., private browsing restrictions) — also skip.
  }
}

export interface UseSettingsReturn {
  settings: Settings;
  updateSetting: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
}

export function useSettings(): UseSettingsReturn {
  const [settings, setSettings] = useState<Settings>(loadFromStorage);

  // Apply fontSize to CSS custom property on mount (restore persisted value)
  useEffect(() => {
    document.documentElement.style.setProperty('--font-size-lyrics', settings.fontSize + 'px');
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Keep CSS custom property in sync with fontSize
  useEffect(() => {
    document.documentElement.style.setProperty('--font-size-lyrics', settings.fontSize + 'px');
  }, [settings.fontSize]);

  const updateSetting = useCallback(<K extends keyof Settings>(key: K, value: Settings[K]) => {
    setSettings(prev => {
      const next = { ...prev, [key]: value };
      saveToStorage(next);
      return next;
    });
  }, []);

  return { settings, updateSetting };
}
