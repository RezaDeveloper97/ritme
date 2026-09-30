'use client';

import { clsx } from 'clsx';
import { useLocale } from 'next-intl';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

import { formatNumber } from '@/shared/lib/date';

import { Icon } from '../Icon';
import {
  DEFAULT_CENTER,
  DEFAULT_ZOOM,
  LOCATE_ZOOM,
  boundsToBbox,
  mapTypeFor,
  toLngLat,
} from './geo';
import { loadNeshanSdk, type NeshanSdk, type NmpMap, type NmpMarker } from './sdk';
import type { MapBbox, MapPin, MapPoint, MapTheme, MapUnavailableReason } from './types';

export interface NeshanMapProps {
  /** Accessible name of the map region (translated by the caller). */
  ariaLabel: string;
  pins: readonly MapPin[];
  center?: MapPoint;
  zoom?: number;
  selectedId?: string | null;
  /** Pin tap → its id (tapping the selected pin again → `null`); map tap → `null`. */
  onSelect?: (id: string | null) => void;
  /** Custom pin body (e.g. a category icon); the wrapper supplies the button. */
  renderPin?: (pin: MapPin, selected: boolean) => ReactNode;
  /** Overlay pinned to the top — search field, filter chips. */
  top?: ReactNode;
  /** Bottom row, beside my-location — e.g. the «فهرست» toggle. */
  controls?: ReactNode;
  /** Selected-place card slot, under the bottom row. */
  card?: ReactNode;
  /** Shows «search this area» after the user moves the map; both are needed. */
  searchAreaLabel?: string;
  onSearchArea?: (bbox: MapBbox) => void;
  /** Shows the my-location button when given (and geolocation exists). */
  myLocationLabel?: string;
  /** The position is handed over and never stored or logged here (§11). */
  onLocate?: (point: MapPoint) => void;
  onLocateError?: () => void;
  /** The map could not appear — the caller shows its list instead. */
  onUnavailable?: (reason: MapUnavailableReason) => void;
  /** Sets the size; the map fills it. */
  className?: string;
}

/**
 * The map colour scheme cannot be a CSS variable (it is a vector style), so
 * this is the one place that reads the resolved theme — the attribute
 * `applyTheme` writes — rather than a token (CLAUDE.md §10.3).
 */
function readTheme(): MapTheme {
  return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light';
}

interface PinHost {
  marker: NmpMarker;
  el: HTMLElement;
}

const floatingButton =
  'pointer-events-auto grid size-11 place-items-center rounded-full border border-(--line) ' +
  'bg-(--surface) text-(--ink) shadow-(--shadow-float) focus-visible:outline-2 ' +
  'focus-visible:outline-(--brand)';

export function NeshanMapView({
  mapKey,
  ariaLabel,
  pins,
  center,
  zoom = DEFAULT_ZOOM,
  selectedId = null,
  onSelect,
  renderPin,
  top,
  controls,
  card,
  searchAreaLabel,
  onSearchArea,
  myLocationLabel,
  onLocate,
  onLocateError,
  onUnavailable,
  className,
}: NeshanMapProps & { mapKey: string }) {
  const locale = useLocale();
  const containerRef = useRef<HTMLDivElement>(null);
  const [sdk, setSdk] = useState<NeshanSdk | null>(null);
  const [map, setMap] = useState<NmpMap | null>(null);
  const [failed, setFailed] = useState(false);
  const [areaMoved, setAreaMoved] = useState(false);
  const [hosts, setHosts] = useState<ReadonlyMap<string, HTMLElement>>(new Map());
  const markersRef = useRef(new Map<string, PinHost>());
  const meMarkerRef = useRef<NmpMarker | null>(null);

  // Latest callbacks without re-creating the map when a parent re-renders.
  const cb = useRef({ onSelect, onUnavailable });
  cb.current = { onSelect, onUnavailable };

  // Initial view only; later `center` changes fly (effect below).
  const initialView = useRef({ center: center ?? DEFAULT_CENTER, zoom });

  // 1. Load the SDK lazily and create the map once.
  useEffect(() => {
    let cancelled = false;
    let created: NmpMap | null = null;
    loadNeshanSdk()
      .then((loaded) => {
        if (cancelled || !containerRef.current) return;
        created = new loaded.Map({
          container: containerRef.current,
          mapKey,
          mapType: mapTypeFor(readTheme()),
          center: toLngLat(initialView.current.center),
          zoom: initialView.current.zoom,
          poi: false,
          traffic: false,
          mapTypeControllerOptions: { show: false },
        });
        created.on('moveend', (e) => {
          if (e.originalEvent) setAreaMoved(true);
        });
        created.on('click', () => cb.current.onSelect?.(null));
        setSdk(loaded);
        setMap(created);
      })
      .catch(() => {
        if (cancelled) return;
        setFailed(true);
        cb.current.onUnavailable?.('load-failed');
      });
    const markers = markersRef.current;
    return () => {
      cancelled = true;
      markers.forEach((m) => m.marker.remove());
      markers.clear();
      meMarkerRef.current?.remove();
      meMarkerRef.current = null;
      created?.remove();
    };
  }, [mapKey]);

  // 2. Follow the app theme (Neshan day/night vector style).
  useEffect(() => {
    if (!map?.setMapType) return;
    const observer = new MutationObserver(() => map.setMapType?.(mapTypeFor(readTheme())));
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['data-theme'],
    });
    return () => observer.disconnect();
  }, [map]);

  // 3. A new `center` from the caller moves the view.
  const centerLat = center?.lat;
  const centerLng = center?.lng;
  useEffect(() => {
    if (!map || centerLat === undefined || centerLng === undefined) return;
    map.flyTo({ center: toLngLat({ lat: centerLat, lng: centerLng }) });
  }, [map, centerLat, centerLng]);

  // 4. Sync one DOM marker per pin; React renders into it through a portal.
  useEffect(() => {
    if (!map || !sdk) return;
    const markers = markersRef.current;
    const wanted = new Set(pins.map((p) => p.id));
    markers.forEach((m, id) => {
      if (!wanted.has(id)) {
        m.marker.remove();
        markers.delete(id);
      }
    });
    for (const pin of pins) {
      const existing = markers.get(pin.id);
      if (existing) {
        existing.marker.setLngLat(toLngLat(pin));
        continue;
      }
      const el = document.createElement('div');
      const marker = new sdk.Marker({ element: el, anchor: 'bottom' })
        .setLngLat(toLngLat(pin))
        .addTo(map);
      markers.set(pin.id, { marker, el });
    }
    setHosts(new Map(Array.from(markers, ([id, m]) => [id, m.el])));
  }, [map, sdk, pins]);

  if (failed) return null;

  const canLocate =
    Boolean(myLocationLabel) && typeof navigator !== 'undefined' && 'geolocation' in navigator;

  function locate(): void {
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const point = { lat: pos.coords.latitude, lng: pos.coords.longitude };
        if (map && sdk) {
          if (!meMarkerRef.current) {
            const el = document.createElement('div');
            el.className =
              'size-4 rounded-full border-2 border-(--surface) bg-(--data) shadow-(--shadow-float)';
            meMarkerRef.current = new sdk.Marker({ element: el, anchor: 'center' }).addTo(map);
          }
          meMarkerRef.current.setLngLat(toLngLat(point));
          map.flyTo({ center: toLngLat(point), zoom: LOCATE_ZOOM });
          setAreaMoved(true);
        }
        onLocate?.(point);
      },
      () => onLocateError?.(),
      { enableHighAccuracy: false, timeout: 10_000, maximumAge: 60_000 },
    );
  }

  function searchArea(): void {
    if (!map || !onSearchArea) return;
    setAreaMoved(false);
    onSearchArea(boundsToBbox(map.getBounds()));
  }

  const pinById = new Map(pins.map((p) => [p.id, p]));

  return (
    <div
      role="region"
      aria-label={ariaLabel}
      aria-busy={!map}
      className={clsx('relative isolate overflow-hidden bg-(--surface-2)', className ?? 'h-full w-full')}
    >
      <div ref={containerRef} className="absolute inset-0" />

      {Array.from(hosts, ([id, el]) => {
        const pin = pinById.get(id);
        if (!pin) return null;
        const selected = pin.id === selectedId;
        return createPortal(
          <button
            type="button"
            aria-label={pin.label}
            aria-pressed={selected}
            onClick={() => onSelect?.(selected ? null : pin.id)}
            className={clsx(
              'flex min-h-11 min-w-11 items-center justify-center gap-1.5 rounded-full border-2 px-2 text-sm font-bold',
              'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-(--brand)',
              selected
                ? 'border-(--brand-fill) bg-(--brand-fill) text-(--on-brand) shadow-(--shadow-cta)'
                : 'border-(--brand) bg-(--surface) text-(--brand)',
            )}
          >
            {renderPin ? (
              renderPin(pin, selected)
            ) : pin.count !== undefined ? (
              formatNumber(pin.count, locale)
            ) : selected ? (
              pin.label
            ) : (
              <span className="size-2.5 rounded-full bg-current" />
            )}
          </button>,
          el,
          id,
        );
      })}

      <div className="pointer-events-none absolute inset-x-0 top-0 flex flex-col gap-2 p-4">
        {top ? <div className="pointer-events-auto">{top}</div> : null}
        {areaMoved && searchAreaLabel && onSearchArea ? (
          <button
            type="button"
            onClick={searchArea}
            className="pointer-events-auto self-center rounded-full border border-(--line) bg-(--surface) px-4 py-2 text-sm font-bold text-(--brand) shadow-(--shadow-float) focus-visible:outline-2 focus-visible:outline-(--brand)"
          >
            {searchAreaLabel}
          </button>
        ) : null}
      </div>

      <div className="pointer-events-none absolute inset-x-0 bottom-0 flex flex-col gap-3 p-4">
        {controls || canLocate ? (
          <div className="flex items-center justify-between gap-3">
            <div className="pointer-events-auto">{controls}</div>
            {canLocate ? (
              <button
                type="button"
                aria-label={myLocationLabel}
                onClick={locate}
                className={floatingButton}
              >
                <Icon name="target" size={20} />
              </button>
            ) : null}
          </div>
        ) : null}
        {card ? <div className="pointer-events-auto">{card}</div> : null}
      </div>
    </div>
  );
}
