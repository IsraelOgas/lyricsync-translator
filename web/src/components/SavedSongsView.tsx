import React, { useState, useEffect, useRef } from 'react';
import { ArrowLeft } from 'lucide-react';
import type { SongSummary, LyricLineData } from '../types';
import { fetchSavedSongs } from '../api';
import { LyricsViewer } from './LyricsViewer';
import styles from './SavedSongsView.module.css';

interface Props {
  showRomanization: boolean;
}

export const SavedSongsView: React.FC<Props> = ({ showRomanization }) => {
  const [songs, setSongs] = useState<SongSummary[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedSong, setSelectedSong] = useState<{
    lines: LyricLineData[];
  } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const loadSongs = async (search?: string) => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchSavedSongs(search);
      setSongs(data);
    } catch {
      setError('Unable to load saved songs. Check that the server is running.');
    } finally {
      setLoading(false);
    }
  };

  // Debounced search (handles initial load too — searchQuery starts as '')
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      loadSongs(searchQuery);
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [searchQuery]);

  const handleSongClick = async (hashKey: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/songs/${hashKey}/lyrics`);
      if (!res.ok) {
        throw new Error(`Failed to fetch song detail: ${res.status}`);
      }
      const data = await res.json();
      setSelectedSong({ lines: data.lines });
    } catch {
      setError('Unable to load song details. Check that the server is running.');
    } finally {
      setLoading(false);
    }
  };

  const handleBack = () => {
    setSelectedSong(null);
  };

  const formatDate = (dateStr: string) => {
    try {
      return new Date(dateStr).toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    } catch {
      return dateStr;
    }
  };

  // Detail view: show LyricsViewer in static mode
  if (selectedSong) {
    return (
      <div className={styles.container}>
        <button className={styles.backBtn} onClick={handleBack} aria-label="Back to song list">
          <ArrowLeft size={18} />
          Back to library
        </button>
        <div className={styles.detailPanel}>
          <LyricsViewer
            lines={selectedSong.lines}
            positionMs={0}
            offsetMs={0}
            paused={false}
            showRomanization={showRomanization}
            staticMode
          />
        </div>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.searchBar}>
        <input
          type="text"
          className={styles.searchInput}
          placeholder="Search by artist or title..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          aria-label="Search saved songs"
        />
      </div>

      {loading && (
        <div className={styles.empty}>
          <div className={styles.spinner} />
          <p className={styles.emptyText}>Loading saved songs...</p>
        </div>
      )}

      {!loading && error && (
        <div className={styles.empty}>
          <p className={styles.errorText}>{error}</p>
        </div>
      )}

      {!loading && !error && songs.length === 0 && !searchQuery && (
        <div className={styles.empty}>
          <p className={styles.emptyText}>No saved songs yet.</p>
          <p className={styles.emptySub}>Play a song to save it here.</p>
        </div>
      )}

      {!loading && !error && songs.length === 0 && searchQuery && (
        <div className={styles.empty}>
          <p className={styles.emptyText}>No songs match your search.</p>
        </div>
      )}

      {!loading && !error && songs.length > 0 && (
        <div className={styles.songList}>
          {songs.map((song) => (
            <button
              key={song.id}
              className={styles.songCard}
              onClick={() => handleSongClick(song.hash_key)}
            >
              <span className={styles.songArtist}>{song.artist}</span>
              <span className={styles.songTitle}>{song.title}</span>
              {song.album && <span className={styles.songAlbum}>{song.album}</span>}
              <span className={styles.songDate}>{formatDate(song.created_at)}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};
