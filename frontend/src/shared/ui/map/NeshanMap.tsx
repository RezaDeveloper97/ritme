'use client';

import dynamic from 'next/dynamic';
import { useEffect, useRef } from 'react';

import { env } from '@/shared/config';

import type { NeshanMapProps } from './NeshanMapView';

// The view (and, inside it, the Neshan SDK script) is split out of every route
// bundle and only fetched when a key exists and a map actually mounts.
const NeshanMapView = dynamic(() => import('./NeshanMapView').then((m) => m.NeshanMapView), {
  ssr: false,
});

/**
 * Neshan web map (DECISIONS #16). No `NEXT_PUBLIC_NESHAN_KEY` → renders nothing
 * and reports `'no-key'`; callers branch on `isMapEnabled()` up front and show
 * their list. A failed SDK load reports `'load-failed'` the same way.
 */
export function NeshanMap(props: NeshanMapProps) {
  const key = env.neshanKey;
  const onUnavailable = useRef(props.onUnavailable);
  onUnavailable.current = props.onUnavailable;

  useEffect(() => {
    if (!key) onUnavailable.current?.('no-key');
  }, [key]);

  if (!key) return null;
  return <NeshanMapView {...props} mapKey={key} />;
}

export type { NeshanMapProps };
