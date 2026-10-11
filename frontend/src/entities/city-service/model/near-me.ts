'use client';

import { useCallback, useState } from 'react';

export interface NearMe {
  point: { lat: number; lng: number } | null;
  status: 'idle' | 'asking' | 'denied';
  /** Asks the browser for the position. Call it from a tap only. */
  request: () => void;
  clear: () => void;
  /** A position the map's own locate button already obtained (same rules: memory only). */
  set: (point: { lat: number; lng: number }) => void;
}

/**
 * The user's position for «near me» / sort by distance. Asked only when she taps, kept in memory for this visit,
 * never stored on the device and sent only as `lat`/`lng` search params (the server does not keep it either).
 */
export function useNearMe(): NearMe {
  const [point, setPoint] = useState<NearMe['point']>(null);
  const [status, setStatus] = useState<NearMe['status']>('idle');
  const request = useCallback(() => {
    if (typeof navigator === 'undefined' || !navigator.geolocation) {
      setStatus('denied');
      return;
    }
    setStatus('asking');
    navigator.geolocation.getCurrentPosition(
      (p) => {
        setPoint({ lat: p.coords.latitude, lng: p.coords.longitude });
        setStatus('idle');
      },
      () => setStatus('denied'),
      { enableHighAccuracy: false, timeout: 10_000, maximumAge: 60_000 },
    );
  }, []);
  const clear = useCallback(() => {
    setPoint(null);
    setStatus('idle');
  }, []);
  const set = useCallback((p: { lat: number; lng: number }) => {
    setPoint({ lat: p.lat, lng: p.lng });
    setStatus('idle');
  }, []);
  return { point, status, request, clear, set };
}
