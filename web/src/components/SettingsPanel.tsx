import React from 'react';
import type { Settings } from '../types';
import styles from './SettingsPanel.module.css';

interface Props {
  isOpen: boolean;
  settings: Settings;
  onUpdateSetting: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
  onClose: () => void;
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

export const SettingsPanel: React.FC<Props> = ({ isOpen, settings, onUpdateSetting, onClose }) => {
  if (!isOpen) return null;

  return (
    <>
      <div className={styles.backdrop} onClick={onClose} />
      <div className={`${styles.panel} ${isOpen ? styles.panelOpen : styles.panelClosed}`}>
        <div className={styles.header}>
          <h2 className={styles.title}>Settings</h2>
          <button className={styles.closeBtn} onClick={onClose} aria-label="Close settings">
            ✕
          </button>
        </div>

        <div className={styles.body}>

          {/* Font Size */}
          <div className={styles.field}>
            <label className={styles.label}>Font Size</label>
            <div className={styles.sliderRow}>
              <input
                type="range"
                className={styles.slider}
                min={14}
                max={40}
                value={settings.fontSize}
                onChange={e => onUpdateSetting('fontSize', Number(e.target.value))}
              />
              <span className={styles.sliderValue}>{settings.fontSize}px</span>
            </div>
          </div>

          {/* Font Family */}
          <div className={styles.field}>
            <label className={styles.label}>Font</label>
            <div className={styles.chipRow}>
              {FONTS.map(f => (
                <button
                  key={f.value}
                  className={`${styles.chip} ${settings.fontFamily === f.value ? styles.chipActive : ''}`}
                  onClick={() => onUpdateSetting('fontFamily', f.value)}
                >
                  {f.label}
                </button>
              ))}
            </div>
          </div>

          {/* Target Language */}
          <div className={styles.field}>
            <label className={styles.label}>Translate to</label>
            <div className={styles.chipRow}>
              {LANGUAGES.map(l => (
                <button
                  key={l.value}
                  className={`${styles.chip} ${settings.targetLang === l.value ? styles.chipActive : ''}`}
                  onClick={() => onUpdateSetting('targetLang', l.value)}
                >
                  {l.label}
                </button>
              ))}
            </div>
          </div>

          {/* Line Spacing */}
          <div className={styles.field}>
            <label className={styles.label}>Line Spacing</label>
            <div className={styles.sliderRow}>
              <input
                type="range"
                className={styles.slider}
                min={1}
                max={2.5}
                step={0.1}
                value={settings.lineSpacing}
                onChange={e => onUpdateSetting('lineSpacing', Number(e.target.value))}
              />
              <span className={styles.sliderValue}>{settings.lineSpacing}</span>
            </div>
          </div>

          {/* Theme */}
          <div className={styles.field}>
            <label className={styles.label}>Theme</label>
            <div className={styles.chipRow}>
              {THEMES.map(t => (
                <button
                  key={t.value}
                  className={`${styles.chip} ${settings.theme === t.value ? styles.chipActive : ''}`}
                  onClick={() => onUpdateSetting('theme', t.value)}
                >
                  {t.label}
                </button>
              ))}
            </div>
          </div>

          {/* Romanization Color */}
          <div className={styles.field}>
            <label className={styles.label}>Romanization Color</label>
            <div className={styles.colorRow}>
              <input
                type="color"
                className={styles.colorInput}
                value={settings.romanizationColor}
                onChange={e => onUpdateSetting('romanizationColor', e.target.value)}
              />
              <span className={styles.colorValue}>{settings.romanizationColor}</span>
            </div>
          </div>

          {/* Translation Color */}
          <div className={styles.field}>
            <label className={styles.label}>Translation Color</label>
            <div className={styles.colorRow}>
              <input
                type="color"
                className={styles.colorInput}
                value={settings.translationColor}
                onChange={e => onUpdateSetting('translationColor', e.target.value)}
              />
              <span className={styles.colorValue}>{settings.translationColor}</span>
            </div>
          </div>

          {/* Show Romanization */}
          <div className={styles.toggleRow}>
            <label className={styles.label}>Show Romanization</label>
            <input
              type="checkbox"
              className={styles.checkbox}
              checked={settings.showRomanization}
              onChange={e => onUpdateSetting('showRomanization', e.target.checked)}
            />
          </div>

        </div>
      </div>
    </>
  );
};
