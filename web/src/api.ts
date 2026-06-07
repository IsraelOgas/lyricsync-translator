import type { SongSummary } from './types';

/** Returns the API path, prepending the API base URL in production.
 *  In dev mode, the template {{.APIBase}} is not substituted, so
 *  we fall back to relative paths (Vite proxies /api to Go).
 *  In production, absolute URLs are used because the Wails webview
 *  loads from an internal scheme (wails://) and needs real HTTP
 *  for features like SSE streaming. CORS middleware on the server
 *  allows these cross-origin requests. */
export function apiUrl(path: string): string {
  const base = window.__API_BASE__ || '';
  return (base && !base.includes('{{')) ? base + path : path;
}

export async function fetchSavedSongs(search?: string): Promise<SongSummary[]> {
  const params = search ? `?search=${encodeURIComponent(search)}` : '';
  const res = await fetch(apiUrl(`/api/songs${params}`));
  if (!res.ok) {
    throw new Error(`Failed to fetch songs: ${res.status} ${res.statusText}`);
  }
  const data = await res.json();
  return data.songs as SongSummary[];
}
