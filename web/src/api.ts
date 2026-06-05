import type { SongSummary } from './types';

export async function fetchSavedSongs(search?: string): Promise<SongSummary[]> {
  const params = search ? `?search=${encodeURIComponent(search)}` : '';
  const res = await fetch(`/api/songs${params}`);
  if (!res.ok) {
    throw new Error(`Failed to fetch songs: ${res.status} ${res.statusText}`);
  }
  const data = await res.json();
  return data.songs as SongSummary[];
}
