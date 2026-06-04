import { useState, useEffect } from 'react';

// Module-scoped URL → color cache survives StrictMode double-mounts
const colorCache = new Map<string, string>();

function rgbToHsl(r: number, g: number, b: number): [number, number, number] {
  r /= 255;
  g /= 255;
  b /= 255;
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const l = (max + min) / 2;

  if (max === min) return [0, 0, l]; // achromatic

  const d = max - min;
  const s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
  let h = 0;
  switch (max) {
    case r:
      h = ((g - b) / d + (g < b ? 6 : 0)) / 6;
      break;
    case g:
      h = ((b - r) / d + 2) / 6;
      break;
    case b:
      h = ((r - g) / d + 4) / 6;
      break;
  }
  return [h, s, l];
}

function hslToRgb(h: number, s: number, l: number): [number, number, number] {
  if (s === 0) {
    const v = Math.round(l * 255);
    return [v, v, v];
  }

  const hueToRgb = (p: number, q: number, t: number): number => {
    if (t < 0) t += 1;
    if (t > 1) t -= 1;
    if (t < 1 / 6) return p + (q - p) * 6 * t;
    if (t < 1 / 2) return q;
    if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6;
    return p;
  };

  const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
  const p = 2 * l - q;
  return [
    Math.round(hueToRgb(p, q, h + 1 / 3) * 255),
    Math.round(hueToRgb(p, q, h) * 255),
    Math.round(hueToRgb(p, q, h - 1 / 3) * 255),
  ];
}

/** Clamp HSL lightness to ≤25% to guarantee dark output. */
function darkenRgb(r: number, g: number, b: number): string {
  const [h, s, l] = rgbToHsl(r, g, b);
  const clampedL = Math.min(l, 0.25);
  const [dr, dg, db] = hslToRgb(h, s, clampedL);
  return `rgb(${dr},${dg},${db})`;
}

/**
 * Draw the image at 10×10 px and average all pixel RGB channels.
 * Catches SecurityError (CORS-blocked getImageData) silently.
 */
function extractDominantColor(img: HTMLImageElement): string | null {
  const canvas = document.createElement('canvas');
  canvas.width = 10;
  canvas.height = 10;
  const ctx = canvas.getContext('2d');
  if (!ctx) return null;

  ctx.drawImage(img, 0, 0, 10, 10);
  try {
    const data = ctx.getImageData(0, 0, 10, 10).data;
    let r = 0;
    let g = 0;
    let b = 0;
    const pixelCount = 100; // 10×10
    for (let i = 0; i < data.length; i += 4) {
      r += data[i];
      g += data[i + 1];
      b += data[i + 2];
    }
    return darkenRgb(
      Math.round(r / pixelCount),
      Math.round(g / pixelCount),
      Math.round(b / pixelCount),
    );
  } catch {
    // SecurityError from getImageData — CORS blocked
    return null;
  }
}

/**
 * Extracts the dominant (HSL-darkened) color from a cover art URL.
 *
 * Strategy:
 * 1. Module-scoped Map cache (O(1) lookup, survives remounts).
 * 2. Try `crossOrigin='anonymous'` first (Spotify CDN).
 * 3. On load error → retry without crossOrigin (local file://).
 * 4. If Canvas read throws SecurityError → returns null silently.
 * 5. null/undefined URL → immediate null, no network request.
 */
export function useCoverColor(url: string | null | undefined): string | null {
  const [color, setColor] = useState<string | null>(null);

  useEffect(() => {
    // Null/undefined URL → bail immediately
    if (!url) {
      setColor(null);
      return;
    }

    // Cache hit — skip extraction
    const cached = colorCache.get(url);
    if (cached) {
      setColor(cached);
      return;
    }

    let cancelled = false;

    const tryExtract = (crossOrigin: boolean): void => {
      const img = new Image();
      if (crossOrigin) {
        img.crossOrigin = 'anonymous';
      }

      img.onload = () => {
        if (cancelled) return;
        const result = extractDominantColor(img);
        if (result) {
          colorCache.set(url, result);
          setColor(result);
        } else {
          setColor(null);
        }
      };

      img.onerror = () => {
        if (cancelled) return;
        if (crossOrigin) {
          // Retry without crossOrigin (local file://, misconfigured CORS)
          tryExtract(false);
        } else {
          // Both attempts failed — silent degradation
          setColor(null);
        }
      };

      img.src = url;
    };

    tryExtract(true);

    return () => {
      cancelled = true;
    };
  }, [url]);

  return color;
}
