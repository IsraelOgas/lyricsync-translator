import React, { useState } from 'react';
import { X, Eye, EyeOff } from 'lucide-react';
import type { Settings } from '../types';
import styles from './SettingsPanel.module.css';

interface Props {
  isOpen: boolean;
  settings: Settings;
  onUpdateSetting: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
  onClose: () => void;
  offsetMs: number;
  onUpdateOffset: (offsetMs: number) => void;
  deepseekApiKey: string;
  onUpdateDeepseekKey: (key: string) => void;
}

const THEMES: { value: Settings['theme']; label: string }[] = [
  { value: 'dark-purple', label: 'Dark Purple' },
  { value: 'dark-blue', label: 'Dark Blue' },
  { value: 'warm-amber', label: 'Warm Amber' },
  { value: 'minimal-mono', label: 'Minimal Mono' },
];

const FONTS: { value: Settings['fontFamily']; label: string }[] = [
  { value: 'sans', label: 'Sans-serif' },
  { value: 'serif', label: 'Serif' },
  { value: 'mono', label: 'Monospace' },
  { value: 'rounded', label: 'Rounded' },
];

const ALIGNMENTS: { value: Settings['textAlignment']; label: string }[] = [
  { value: 'left', label: 'Left' },
  { value: 'center', label: 'Center' },
  { value: 'right', label: 'Right' },
];

const LANGUAGES: { value: string; label: string }[] = [
  { value: 'es', label: 'Español' },
  { value: 'en', label: 'English' },
  { value: 'pt', label: 'Português' },
  { value: 'fr', label: 'Français' },
  { value: 'de', label: 'Deutsch' },
  { value: 'it', label: 'Italiano' },
  { value: 'ja', label: '日本語' },
  { value: 'ko', label: '한국어' },
  { value: 'zh', label: '中文' },
];

export const SettingsPanel: React.FC<Props> = ({ isOpen, settings, onUpdateSetting, onClose, offsetMs, onUpdateOffset, deepseekApiKey, onUpdateDeepseekKey }) => {
  if (!isOpen) return null;

  const [showKey, setShowKey] = useState(false);
  const [localKey, setLocalKey] = useState('');
  const [saved, setSaved] = useState(false);

  return (
    <>
      <div className={styles.backdrop} onClick={onClose} />
      <div className={`${styles.panel} ${isOpen ? styles.panelOpen : styles.panelClosed}`}>
        <div className={styles.header}>
          <h2 className={styles.title}>Settings</h2>
          <button className={styles.closeBtn} onClick={onClose} aria-label="Close settings">
            <X size={18} />
          </button>
        </div>

        <div className={styles.body}>

          {/* Sync Offset */}
          <div className={styles.field}>
            <label className={styles.label} title="Adjust lyrics timing relative to the audio, in milliseconds. Positive values delay the lyrics; negative values show them earlier.">Sync Offset</label>
            <div className={styles.sliderRow}>
              <input
                type="range"
                className={styles.slider}
                min={-5000}
                max={5000}
                step={100}
                value={offsetMs}
                onChange={e => onUpdateOffset(Number(e.target.value))}
                title="Drag to fine-tune when each line appears (milliseconds)."
              />
              <span className={styles.sliderValue}>{offsetMs > 0 ? '+' : ''}{(offsetMs / 1000).toFixed(1)}s</span>
            </div>
          </div>

          {/* Font Size */}
          <div className={styles.field}>
            <label className={styles.label} title="Size of the lyrics text, in pixels.">Font Size</label>
            <div className={styles.sliderRow}>
              <input
                type="range"
                className={styles.slider}
                min={14}
                max={40}
                value={settings.fontSize}
                onChange={e => onUpdateSetting('fontSize', Number(e.target.value))}
                title="Bigger values make the lyrics easier to read from a distance."
              />
              <span className={styles.sliderValue}>{settings.fontSize}px</span>
            </div>
          </div>

          {/* Font Family */}
          <div className={styles.field}>
            <label className={styles.label} title="Font family used for the lyrics text.">Font</label>
            <div className={styles.chipRow}>
              {FONTS.map(f => (
                <button
                  key={f.value}
                  className={`${styles.chip} ${settings.fontFamily === f.value ? styles.chipActive : ''}`}
                  onClick={() => onUpdateSetting('fontFamily', f.value)}
                  title={`Use the "${f.label}" font for lyrics.`}
                >
                  {f.label}
                </button>
              ))}
            </div>
          </div>

          {/* Text Alignment */}
          <div className={styles.field}>
            <label className={styles.label} title="Horizontal alignment of the lyrics lines on screen.">Text Alignment</label>
            <div className={styles.chipRow}>
              {ALIGNMENTS.map(a => (
                <button
                  key={a.value}
                  className={`${styles.chip} ${settings.textAlignment === a.value ? styles.chipActive : ''}`}
                  onClick={() => onUpdateSetting('textAlignment', a.value)}
                  title={`Align lyrics to the ${a.label.toLowerCase()}.`}
                >
                  {a.label}
                </button>
              ))}
            </div>
          </div>

          {/* Target Language */}
          <div className={styles.field}>
            <label className={styles.label} title="Language the lyrics are translated into. Uses the active translation provider (DeepSeek or LibreTranslate).">Translate to</label>
            <div className={styles.chipRow}>
              {LANGUAGES.map(l => (
                <button
                  key={l.value}
                  className={`${styles.chip} ${settings.targetLang === l.value ? styles.chipActive : ''}`}
                  onClick={() => onUpdateSetting('targetLang', l.value)}
                  title={`Translate lyrics to ${l.label}.`}
                >
                  {l.label}
                </button>
              ))}
            </div>
          </div>

          {/* DeepSeek API Key */}
          <div className={styles.field}>
            <label className={styles.label} title="API key for the DeepSeek translation provider. Saved to ~/.config/lyricsync/config.yaml and applied without restarting.">
              DeepSeek API Key
              {deepseekApiKey && !localKey && (
                <span className={styles.configuredBadge}>✓ configured</span>
              )}
              {saved && (
                <span className={styles.configuredBadge}>✓ saved</span>
              )}
            </label>
            <div className={styles.inputRow}>
              <input
                type={showKey ? 'text' : 'password'}
                className={styles.textInput}
                placeholder={deepseekApiKey ? 'Key is configured' : 'Paste your API key'}
                value={localKey || deepseekApiKey}
                onFocus={e => {
                  if (!localKey) e.target.select();
                }}
                onChange={e => {
                  setLocalKey(e.target.value);
                  setSaved(false);
                }}
                onBlur={() => {
                  if (localKey && localKey !== '••••••••') {
                    onUpdateDeepseekKey(localKey);
                    setSaved(true);
                    setTimeout(() => setSaved(false), 2000);
                  }
                }}
                onKeyDown={e => {
                  if (e.key === 'Enter' && localKey && localKey !== '••••••••') {
                    onUpdateDeepseekKey(localKey);
                    setSaved(true);
                    setTimeout(() => setSaved(false), 2000);
                  }
                }}
                title="Required when the translation provider is DeepSeek. Press Enter or blur to save."
              />
              <button
                type="button"
                className={styles.iconBtn}
                onClick={() => setShowKey(!showKey)}
                aria-label={showKey ? 'Hide API key' : 'Show API key'}
                title={showKey ? 'Hide the API key' : 'Reveal the API key'}
              >
                {showKey ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
          </div>

          {/* Line Spacing */}
          <div className={styles.field}>
            <label className={styles.label} title="Vertical space between lyric lines. Higher values reduce visual crowding.">Line Spacing</label>
            <div className={styles.sliderRow}>
              <input
                type="range"
                className={styles.slider}
                min={1}
                max={2.5}
                step={0.1}
                value={settings.lineSpacing}
                onChange={e => onUpdateSetting('lineSpacing', Number(e.target.value))}
                title="Adjust the gap between lines (1 = tight, 2.5 = airy)."
              />
              <span className={styles.sliderValue}>{settings.lineSpacing}</span>
            </div>
          </div>

          {/* Theme */}
          <div className={styles.field}>
            <label className={styles.label} title="Color theme applied to the whole app.">Theme</label>
            <div className={styles.chipRow}>
              {THEMES.map(t => (
                <button
                  key={t.value}
                  className={`${styles.chip} ${settings.theme === t.value ? styles.chipActive : ''}`}
                  onClick={() => onUpdateSetting('theme', t.value)}
                  title={`Switch to the "${t.label}" theme.`}
                >
                  {t.label}
                </button>
              ))}
            </div>
          </div>

          {/* Romanization Color */}
          <div className={styles.field}>
            <label className={styles.label} title="Color of the romanized text (transliteration of Japanese, Chinese, Korean).">Romanization Color</label>
            <div className={styles.colorRow}>
              <input
                type="color"
                className={styles.colorInput}
                value={settings.romanizationColor}
                onChange={e => onUpdateSetting('romanizationColor', e.target.value)}
                title="Pick the color used for romanized lyrics."
              />
              <span className={styles.colorValue}>{settings.romanizationColor}</span>
            </div>
          </div>

          {/* Translation Color */}
          <div className={styles.field}>
            <label className={styles.label} title="Color of the translated lyrics text.">Translation Color</label>
            <div className={styles.colorRow}>
              <input
                type="color"
                className={styles.colorInput}
                value={settings.translationColor}
                onChange={e => onUpdateSetting('translationColor', e.target.value)}
                title="Pick the color used for translated lyrics."
              />
              <span className={styles.colorValue}>{settings.translationColor}</span>
            </div>
          </div>

          {/* Show Romanization */}
          <div className={styles.toggleRow}>
            <label className={styles.label} title="Show the romanized (transliterated) text under non-Latin lyrics (Japanese, Chinese, Korean). ON shows it; OFF hides it.">Show Romanization</label>
            <input
              type="checkbox"
              className={styles.checkbox}
              checked={settings.showRomanization}
              onChange={e => onUpdateSetting('showRomanization', e.target.checked)}
            />
          </div>

          {/* Translate Lyrics */}
          <div className={styles.toggleRow}>
            <label className={styles.label} title="Enable or disable the translation pipeline. ON translates lyrics via the configured provider (uses API tokens); OFF makes zero provider calls (no cost).">Translate Lyrics</label>
            <input
              type="checkbox"
              className={styles.checkbox}
              checked={settings.translationEnabled}
              onChange={e => onUpdateSetting('translationEnabled', e.target.checked)}
            />
          </div>

          {/* Cinema Mode */}
          <div className={styles.toggleRow}>
            <label className={styles.label} title="Fullscreen immersive mode with an animated background, floating track info, and hidden UI bars. ON enables it; OFF returns to the normal view.">Cinema Mode</label>
            <input
              type="checkbox"
              className={styles.checkbox}
              checked={settings.cinemaMode}
              onChange={e => onUpdateSetting('cinemaMode', e.target.checked)}
            />
          </div>

          {/* Karaoke */}
          <div className={styles.toggleRow}>
            <label className={styles.label} title="Karaoke highlighting. ON requests word-level timestamps from lrcmux and paints word-by-word when available (falls back to line fill otherwise); OFF requests line-level (cached, free) with no painting.">Karaoke</label>
            <input
              type="checkbox"
              className={styles.checkbox}
              checked={settings.karaokeMode}
              onChange={e => onUpdateSetting('karaokeMode', e.target.checked)}
            />
          </div>

        </div>
      </div>
    </>
  );
};
