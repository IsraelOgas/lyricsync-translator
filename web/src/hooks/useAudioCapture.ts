import { useState, useRef, useCallback, useEffect } from 'react';

interface AudioCaptureState {
  isCapturing: boolean;
  analyser: AnalyserNode | null;
  error: string | null;
}

export function useAudioCapture(): AudioCaptureState & {
  startCapture: () => Promise<void>;
  stopCapture: () => void;
} {
  const [isCapturing, setIsCapturing] = useState(false);
  const [analyser, setAnalyser] = useState<AnalyserNode | null>(null);
  const [error, setError] = useState<string | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const audioCtxRef = useRef<AudioContext | null>(null);

  const stopCapture = useCallback(() => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach(t => t.stop());
      streamRef.current = null;
    }
    if (audioCtxRef.current) {
      audioCtxRef.current.close().catch(() => {});
      audioCtxRef.current = null;
    }
    setAnalyser(null);
    setIsCapturing(false);
  }, []);

  const startCapture = useCallback(async () => {
    try {
      setError(null);
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: false,
          noiseSuppression: false,
          autoGainControl: false,
        }
      });
      streamRef.current = stream;

      const ctx = new AudioContext();
      audioCtxRef.current = ctx;
      const source = ctx.createMediaStreamSource(stream);
      const analyserNode = ctx.createAnalyser();
      analyserNode.fftSize = 256;
      analyserNode.smoothingTimeConstant = 0.8;
      source.connect(analyserNode);

      setAnalyser(analyserNode);
      setIsCapturing(true);
    } catch (err: any) {
      setError(err.message || 'Failed to capture audio');
      setIsCapturing(false);
    }
  }, []);

  useEffect(() => {
    return () => stopCapture();
  }, [stopCapture]);

  return { isCapturing, analyser, error, startCapture, stopCapture };
}