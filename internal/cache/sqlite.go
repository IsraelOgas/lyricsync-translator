package cache

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	// Ensure the directory exists
	if dir := path.Dir(dbPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating db directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS songs (
		id TEXT PRIMARY KEY,
		hash_key TEXT UNIQUE NOT NULL,
		artist TEXT NOT NULL,
		title TEXT NOT NULL,
		album TEXT,
		duration_ms INTEGER,
		offset_ms INTEGER NOT NULL DEFAULT 0,
		source TEXT NOT NULL DEFAULT 'lrclib',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE TABLE IF NOT EXISTS lyric_lines (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		song_id TEXT NOT NULL REFERENCES songs(id),
		line_num INTEGER NOT NULL,
		time_ms INTEGER,
		original TEXT NOT NULL,
		lang TEXT,
		UNIQUE(song_id, line_num)
	);
	CREATE TABLE IF NOT EXISTS translations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		lyric_line_id INTEGER NOT NULL REFERENCES lyric_lines(id),
		romanized TEXT,
		translated_text TEXT,
		target_lang TEXT NOT NULL DEFAULT 'es',
		UNIQUE(lyric_line_id, target_lang)
	);`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Migrations for existing databases (add columns that may be missing)
	migrations := []string{
		"ALTER TABLE songs ADD COLUMN offset_ms INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE translations ADD COLUMN translated_text TEXT",
		"ALTER TABLE translations ADD COLUMN target_lang TEXT NOT NULL DEFAULT 'es'",
	}
	for _, m := range migrations {
		s.db.Exec(m) // ignore errors — column may already exist
	}

	return nil
}

func HashKey(artist, title, album string) string {
	h := sha256.New()
	h.Write([]byte(artist + "|" + title + "|" + album))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Store) GetSongByHash(hashKey string) (*Song, error) {
	row := s.db.QueryRow("SELECT id, hash_key, artist, title, album, duration_ms, offset_ms, source, created_at FROM songs WHERE hash_key = ?", hashKey)
	var song Song
	var createdAt string
	err := row.Scan(&song.ID, &song.HashKey, &song.Artist, &song.Title, &song.Album, &song.DurationMs, &song.OffsetMs, &song.Source, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	song.CreatedAt, _ = time.Parse("2026-01-01 13:00:00", createdAt)
	return &song, nil
}

func (s *Store) SaveSong(song *Song) error {
	if song.ID == "" {
		song.ID = uuid.New().String()
	}
	if song.HashKey == "" {
		song.HashKey = HashKey(song.Artist, song.Title, song.Album)
	}
	if song.Source == "" {
		song.Source = "lrclib"
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO songs (id, hash_key, artist, title, album, duration_ms, offset_ms, source, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		song.ID, song.HashKey, song.Artist, song.Title, song.Album, song.DurationMs, song.OffsetMs, song.Source,
	)
	return err
}

func (s *Store) GetLyricLines(songID string) ([]LyricLine, error) {
	rows, err := s.db.Query("SELECT id, song_id, line_num, time_ms, original, lang FROM lyric_lines WHERE song_id = ? ORDER BY line_num", songID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []LyricLine
	for rows.Next() {
		var l LyricLine
		err := rows.Scan(&l.ID, &l.SongID, &l.LineNum, &l.TimeMs, &l.Original, &l.Lang)
		if err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

func (s *Store) SaveLyricLines(songID string, lines []LyricLine) error {
	stmt, err := s.db.Prepare("INSERT OR REPLACE INTO lyric_lines (song_id, line_num, time_ms, original, lang) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, l := range lines {
		_, err := stmt.Exec(songID, l.LineNum, l.TimeMs, l.Original, l.Lang)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetTranslation(lyricLineID int) (*Translation, error) {
	row := s.db.QueryRow("SELECT id, lyric_line_id, romanized, translated_text, target_lang FROM translations WHERE lyric_line_id = ?", lyricLineID)
	var t Translation
	err := row.Scan(&t.ID, &t.LyricLineID, &t.Romanized, &t.TranslatedText, &t.TargetLang)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) SaveTranslation(t *Translation) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO translations (lyric_line_id, romanized, translated_text, target_lang) VALUES (?, ?, ?, ?)",
		t.LyricLineID, t.Romanized, t.TranslatedText, t.TargetLang,
	)
	return err
}

func (s *Store) GetTranslationsBySong(songID, targetLang string) (map[int]*Translation, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.lyric_line_id, t.romanized, t.translated_text, t.target_lang
		FROM translations t
		JOIN lyric_lines l ON t.lyric_line_id = l.id
		WHERE l.song_id = ? AND t.target_lang = ?
	`, songID, targetLang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]*Translation)
	for rows.Next() {
		var t Translation
		err := rows.Scan(&t.ID, &t.LyricLineID, &t.Romanized, &t.TranslatedText, &t.TargetLang)
		if err != nil {
			return nil, err
		}
		result[t.LyricLineID] = &t
	}
	return result, rows.Err()
}

func (s *Store) GetSongOffset(hashKey string) (int, error) {
	var offset int
	err := s.db.QueryRow("SELECT offset_ms FROM songs WHERE hash_key = ?", hashKey).Scan(&offset)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return offset, err
}

func (s *Store) ListSongs(search string) ([]Song, error) {
	rows, err := s.db.Query(
		`SELECT id, hash_key, artist, title, album, duration_ms, offset_ms, source, created_at
		 FROM songs
		 WHERE ? = '' OR artist LIKE ? OR title LIKE ? OR album LIKE ?
		 ORDER BY created_at DESC`,
		search, "%"+search+"%", "%"+search+"%", "%"+search+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []Song
	for rows.Next() {
		var song Song
		var createdAt string
		err := rows.Scan(&song.ID, &song.HashKey, &song.Artist, &song.Title, &song.Album,
			&song.DurationMs, &song.OffsetMs, &song.Source, &createdAt)
		if err != nil {
			return nil, err
		}
		song.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		songs = append(songs, song)
	}
	if songs == nil {
		songs = []Song{}
	}
	return songs, rows.Err()
}

func (s *Store) UpdateSongOffset(hashKey string, offsetMs int) error {
	_, err := s.db.Exec("UPDATE songs SET offset_ms = ? WHERE hash_key = ?", offsetMs, hashKey)
	return err
}
