import React from 'react';
import type { Settings } from '../types';
import styles from './SettingsPanel.module.css';

interface Props {
  isOpen: boolean;
  settings: Settings;
  onUpdateSetting: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
  onClose: () => void;
}

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

        <div className={styles.field}>
          <label className={styles.label} htmlFor="font-size-slider">Font Size</label>
          <input
            id="font-size-slider"
            type="range"
            className={styles.slider}
            min={14}
            max={40}
            value={settings.fontSize}
            onChange={e => onUpdateSetting('fontSize', Number(e.target.value))}
          />
          <span className={styles.sliderValue}>{settings.fontSize}px</span>
        </div>

        <div className={styles.toggleRow}>
          <label className={styles.label} htmlFor="romanization-toggle">Show Romanization</label>
          <input
            id="romanization-toggle"
            type="checkbox"
            className={styles.checkbox}
            checked={settings.showRomanization}
            onChange={e => onUpdateSetting('showRomanization', e.target.checked)}
          />
        </div>
      </div>
    </>
  );
};
