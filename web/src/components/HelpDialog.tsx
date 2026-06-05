import React from 'react';
import { X } from 'lucide-react';
import styles from './HelpDialog.module.css';

interface Props {
  open: boolean;
  onClose: () => void;
}

interface ShortcutEntry {
  key: string;
  action: string;
}

const SHORTCUTS: ShortcutEntry[] = [
  { key: 'Space', action: 'Play / Pause' },
  { key: '← →', action: 'Seek -5s / +5s' },
  { key: 'N / P', action: 'Next / Previous track' },
  { key: '↑ ↓', action: 'Volume up / down' },
  { key: 'M', action: 'Mute / Unmute' },
  { key: 'S', action: 'Toggle shuffle' },
  { key: 'L', action: 'Cycle loop' },
  { key: 'R', action: 'Toggle romanization' },
  { key: 'C', action: 'Toggle cinema mode' },
  { key: 'Esc', action: 'Close this dialog' },
];

export const HelpDialog: React.FC<Props> = ({ open, onClose }) => {
  if (!open) return null;

  return (
    <div className={styles.backdrop} onClick={onClose}>
      <div className={styles.dialog} onClick={e => e.stopPropagation()}>
        <div className={styles.header}>
          <h2 className={styles.title}>Keyboard Shortcuts</h2>
          <button className={styles.closeBtn} onClick={onClose} aria-label="Close">
            <X size={18} />
          </button>
        </div>
        <div className={styles.table}>
          {SHORTCUTS.map(entry => (
            <div key={entry.key} className={styles.row}>
              <kbd className={styles.kbd}>{entry.key}</kbd>
              <span className={styles.action}>{entry.action}</span>
            </div>
          ))}
        </div>
        <p className={styles.hint}>Press <kbd className={styles.kbd}>?</kbd> to open this dialog anytime.</p>
      </div>
    </div>
  );
};
