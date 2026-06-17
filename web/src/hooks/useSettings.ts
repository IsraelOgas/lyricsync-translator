import { useState, useEffect, useCallback } from 'react';
import { Settings, DEFAULT_SETTINGS } from '../types';
import { apiUrl } from '../api';

const FONT_FAMILIES: Record<Settings['fontFamily'], string> = {
  sans: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
  serif: "'Lora', 'Georgia', 'Times New Roman', serif",
  mono: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
  rounded: "'Nunito', 'Quicksand', 'Segoe UI', sans-serif",
};

function loadFromStorage(): Settings {
  try {
    const raw = localStorage.getItem('lyricsync-settings');
    if (!raw) return DEFAULT_SETTINGS;
    const parsed = JSON.parse(raw);
    return {
      fontSize: typeof parsed.fontSize === 'number' ? parsed.fontSize : DEFAULT_SETTINGS.fontSize,
      showRomanization: typeof parsed.showRomanization === 'boolean' ? parsed.showRomanization : DEFAULT_SETTINGS.showRomanization,
      fontFamily: ['sans', 'serif', 'mono', 'rounded'].includes(parsed.fontFamily) ? parsed.fontFamily : DEFAULT_SETTINGS.fontFamily,
      lineSpacing: typeof parsed.lineSpacing === 'number' ? parsed.lineSpacing : DEFAULT_SETTINGS.lineSpacing,
      theme: ['dark-purple', 'dark-blue', 'warm-amber', 'minimal-mono'].includes(parsed.theme) ? parsed.theme : DEFAULT_SETTINGS.theme,
      translationColor: typeof parsed.translationColor === 'string' ? parsed.translationColor : DEFAULT_SETTINGS.translationColor,
      romanizationColor: typeof parsed.romanizationColor === 'string' ? parsed.romanizationColor : DEFAULT_SETTINGS.romanizationColor,
      targetLang: typeof parsed.targetLang === 'string' ? parsed.targetLang : DEFAULT_SETTINGS.targetLang,
      cinemaMode: typeof parsed.cinemaMode === 'boolean' ? parsed.cinemaMode : DEFAULT_SETTINGS.cinemaMode,
      cinemaOrbs: typeof parsed.cinemaOrbs === 'boolean' ? parsed.cinemaOrbs : DEFAULT_SETTINGS.cinemaOrbs,
    };
  } catch {
    return DEFAULT_SETTINGS;
  }
}

function saveToStorage(settings: Settings): void {
  try {
    localStorage.setItem('lyricsync-settings', JSON.stringify(settings));
  } catch {
    // Silently skip if storage unavailable
  }
}

function applySettings(settings: Settings): void {
  const root = document.documentElement;
  root.style.setProperty('--font-size-lyrics', settings.fontSize + 'px');
  root.style.setProperty('--font-family-lyrics', FONT_FAMILIES[settings.fontFamily]);
  root.style.setProperty('--line-spacing-lyrics', String(settings.lineSpacing));
  root.style.setProperty('--color-romanization', settings.romanizationColor);
  root.style.setProperty('--color-translation', settings.translationColor);
  root.setAttribute('data-theme', settings.theme);

  // Native fullscreen via Wails runtime (guarded — only active inside Wails WebView).
  if (settings.cinemaMode) {
    window.runtime?.WindowFullscreen();
  } else {
    window.runtime?.WindowUnfullscreen();
  }
}

export interface UseSettingsReturn {
  settings: Settings;
  updateSetting: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
}

export function useSettings(): UseSettingsReturn {
  const [settings, setSettings] = useState<Settings>(loadFromStorage);

  // Apply all settings on mount
  useEffect(() => {
    applySettings(settings);
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Keep CSS in sync with state
  useEffect(() => {
    applySettings(settings);
  }, [settings]);

  const updateSetting = useCallback(<K extends keyof Settings>(key: K, value: Settings[K]) => {
    setSettings(prev => {
      const next = { ...prev, [key]: value };
      saveToStorage(next);

      // Sync target language to backend
      if (key === 'targetLang') {
        fetch(apiUrl('/api/config'), {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ target_lang: value }),
        }).catch(() => {}); // non-blocking — backend will use default if unreachable
      }

      return next;
    });
  }, []);

  return { settings, updateSetting };
}
