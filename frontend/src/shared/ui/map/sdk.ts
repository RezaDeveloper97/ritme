import { NESHAN_SDK } from '@/shared/config';

/**
 * The slice of the Neshan (mapbox-gl 1.13) global this wrapper uses. Typed by
 * hand: the SDK ships as a CDN script with no types, and pulling the whole
 * mapbox-gl typings in for five methods is not worth a dependency.
 */
export interface NmpBounds {
  getSouth(): number;
  getWest(): number;
  getNorth(): number;
  getEast(): number;
}

export interface NmpEvent {
  /** Present only when the user caused the event (drag, pinch, wheel). */
  originalEvent?: unknown;
}

export interface NmpMap {
  on(type: string, listener: (e: NmpEvent) => void): void;
  getBounds(): NmpBounds;
  flyTo(options: { center: [number, number]; zoom?: number }): void;
  setMapType?(type: string): void;
  resize(): void;
  remove(): void;
}

export interface NmpMarker {
  setLngLat(lngLat: [number, number]): NmpMarker;
  addTo(map: NmpMap): NmpMarker;
  remove(): void;
}

export interface NmpMapOptions {
  container: HTMLElement;
  mapKey: string;
  mapType: string;
  center: [number, number];
  zoom: number;
  poi: boolean;
  traffic: boolean;
  mapTypeControllerOptions: { show: boolean };
}

export interface NeshanSdk {
  Map: new (options: NmpMapOptions) => NmpMap;
  Marker: new (options: { element: HTMLElement; anchor?: 'bottom' | 'center' }) => NmpMarker;
}

declare global {
  interface Window {
    nmp_mapboxgl?: NeshanSdk;
  }
}

const LOAD_TIMEOUT_MS = 15_000;

let pending: Promise<NeshanSdk> | null = null;

/**
 * Inject the SDK script + stylesheet once, on first use (never in the initial
 * bundle). A failure clears the cache so a later mount can retry; the caller
 * treats a rejection as "no map" and shows its list.
 */
export function loadNeshanSdk(): Promise<NeshanSdk> {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return Promise.reject(new Error('neshan-sdk: no DOM'));
  }
  if (window.nmp_mapboxgl) return Promise.resolve(window.nmp_mapboxgl);
  if (pending) return pending;

  pending = new Promise<NeshanSdk>((resolve, reject) => {
    if (!document.querySelector(`link[href="${NESHAN_SDK.styleUrl}"]`)) {
      const link = document.createElement('link');
      link.rel = 'stylesheet';
      link.href = NESHAN_SDK.styleUrl;
      document.head.appendChild(link);
    }

    const script = document.createElement('script');
    script.src = NESHAN_SDK.scriptUrl;
    script.async = true;

    const timer = window.setTimeout(() => fail(new Error('neshan-sdk: timeout')), LOAD_TIMEOUT_MS);
    function fail(err: Error): void {
      window.clearTimeout(timer);
      script.remove();
      pending = null;
      reject(err);
    }

    script.onload = () => {
      window.clearTimeout(timer);
      if (window.nmp_mapboxgl) resolve(window.nmp_mapboxgl);
      else fail(new Error('neshan-sdk: global missing'));
    };
    script.onerror = () => fail(new Error('neshan-sdk: load error'));
    document.head.appendChild(script);
  });
  return pending;
}
