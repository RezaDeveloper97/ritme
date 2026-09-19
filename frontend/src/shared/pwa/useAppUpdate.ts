'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

import { compareVersions } from './semver';

export type UpdateStatus = 'none' | 'soft' | 'forced';

interface VersionInfo {
  version: string;
  minSupportedVersion: string;
  releaseNotes?: string;
  buildId?: string;
}

interface AppUpdateState {
  status: UpdateStatus;
  latestVersion: string | null;
  /** Identifies the announced update, for dismissal (version + build). */
  updateKey: string;
  releaseNotes: string;
  apply: () => Promise<void>;
}

const CURRENT_VERSION = process.env.NEXT_PUBLIC_APP_VERSION ?? '0.0.0';
// Stamped per build (scripts/generate-version.mjs → next.config.ts); empty in dev.
const CURRENT_BUILD_ID = process.env.NEXT_PUBLIC_BUILD_ID ?? '';
const POLL_INTERVAL_MS = 15 * 60 * 1000;

// Module-level so React StrictMode double-mounting can't cause a reload loop:
// controllerchange reloads at most once per page lifetime.
let reloadedOnControllerChange = false;

// Captured when this module first evaluates on the client — before the app
// registers the worker. A page that had no controller at load gets its first
// controller from the worker's `clients.claim()`; that is not an update, and
// reloading there wiped whatever the first-time visitor was typing
// (pwa-audit S-3). Only a page that was already controlled, or whose user
// pressed "update", reloads on controllerchange.
const controlledAtLoad =
  typeof navigator !== 'undefined' &&
  'serviceWorker' in navigator &&
  navigator.serviceWorker.controller !== null;
let userRequestedUpdate = false;

function isSwSupported(): boolean {
  return typeof window !== 'undefined' && 'serviceWorker' in navigator;
}

async function fetchVersionInfo(): Promise<VersionInfo | null> {
  try {
    const res = await fetch('/version.json', { cache: 'no-store' });
    if (!res.ok) return null;
    const data = (await res.json()) as Partial<VersionInfo>;
    if (typeof data.version !== 'string') return null;
    return {
      version: data.version,
      minSupportedVersion:
        typeof data.minSupportedVersion === 'string' ? data.minSupportedVersion : '0.0.0',
      releaseNotes: typeof data.releaseNotes === 'string' ? data.releaseNotes : '',
      buildId: typeof data.buildId === 'string' ? data.buildId : undefined,
    };
  } catch {
    return null;
  }
}

/**
 * Two-tier PWA update detection (soft banner vs forced blocking screen).
 *
 * Channels:
 *  1. Service worker: a new worker reaching `installed` while the page is
 *     already controlled means an update is waiting.
 *  2. Polling: /version.json (no-store) on start, on tab focus, and every
 *     15 minutes, compared numerically against the baked-in build version.
 *
 * Classification:
 *  - forced — running version < minSupportedVersion: blocking overlay.
 *  - soft — a newer version exists (either channel): dismissible banner.
 */
export function useAppUpdate(): AppUpdateState {
  const [waitingWorker, setWaitingWorker] = useState<ServiceWorker | null>(null);
  const [remote, setRemote] = useState<VersionInfo | null>(null);
  const registrationRef = useRef<ServiceWorkerRegistration | null>(null);

  useEffect(() => {
    if (!isSwSupported() || process.env.NODE_ENV !== 'production') return;

    let disposed = false;

    const trackWaiting = (reg: ServiceWorkerRegistration) => {
      if (reg.waiting && navigator.serviceWorker.controller) {
        setWaitingWorker(reg.waiting);
      }
    };

    navigator.serviceWorker
      .register('/sw.js')
      .then((reg) => {
        if (disposed) return;
        registrationRef.current = reg;
        trackWaiting(reg);
        reg.addEventListener('updatefound', () => {
          const installing = reg.installing;
          if (!installing) return;
          installing.addEventListener('statechange', () => {
            if (installing.state === 'installed' && navigator.serviceWorker.controller) {
              setWaitingWorker(reg.waiting ?? installing);
            }
          });
        });
      })
      .catch(() => {
        // Registration failure degrades to polling-only detection.
      });

    const onControllerChange = () => {
      if (!controlledAtLoad && !userRequestedUpdate) return;
      if (reloadedOnControllerChange) return;
      reloadedOnControllerChange = true;
      window.location.reload();
    };
    navigator.serviceWorker.addEventListener('controllerchange', onControllerChange);

    return () => {
      disposed = true;
      navigator.serviceWorker.removeEventListener('controllerchange', onControllerChange);
    };
  }, []);

  useEffect(() => {
    if (process.env.NODE_ENV !== 'production') return;

    let disposed = false;

    const poll = async () => {
      const info = await fetchVersionInfo();
      if (!disposed && info) setRemote(info);
    };

    void poll();
    const interval = window.setInterval(poll, POLL_INTERVAL_MS);

    const onVisible = () => {
      if (document.visibilityState !== 'visible') return;
      void poll();
      // Long-lived tabs must also discover new workers.
      void registrationRef.current?.update().catch(() => undefined);
    };
    document.addEventListener('visibilitychange', onVisible);

    return () => {
      disposed = true;
      window.clearInterval(interval);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, []);

  const forced =
    remote !== null && compareVersions(CURRENT_VERSION, remote.minSupportedVersion) < 0;
  const newerRemote = remote !== null && compareVersions(remote.version, CURRENT_VERSION) > 0;
  // Same version, different build: a deploy that forgot to bump package.json.
  // Still an update — the running HTML may point at chunks that are gone.
  const rebuiltRemote =
    remote !== null &&
    CURRENT_BUILD_ID !== '' &&
    remote.buildId !== undefined &&
    remote.buildId !== CURRENT_BUILD_ID &&
    compareVersions(remote.version, CURRENT_VERSION) === 0;
  const status: UpdateStatus = forced
    ? 'forced'
    : newerRemote || rebuiltRemote || waitingWorker !== null
      ? 'soft'
      : 'none';

  const apply = useCallback(async () => {
    userRequestedUpdate = true;
    const reg = registrationRef.current;
    const waiting = waitingWorker ?? reg?.waiting ?? null;

    if (forced) {
      // Stale runtime caches from an unsupported build must not survive.
      navigator.serviceWorker?.controller?.postMessage({ type: 'CLEAR_RUNTIME_CACHE' });
    }

    if (waiting) {
      // controllerchange handler performs the (single) reload.
      waiting.postMessage({ type: 'SKIP_WAITING' });
      // Safety net in case controllerchange never fires (e.g. first install).
      window.setTimeout(() => {
        window.location.reload();
      }, 2500);
      return;
    }

    // Newer version announced but the browser hasn't fetched the new worker
    // yet — force an update check, then reload to pick up fresh HTML.
    try {
      await reg?.update();
    } catch {
      // Offline or transient failure: the reload below retries from network.
    }
    window.location.reload();
  }, [forced, waitingWorker]);

  return {
    status,
    latestVersion: remote?.version ?? null,
    // SW-channel-only updates have no version string; 'sw' still lets the
    // user dismiss them (and re-arms when a different build appears).
    updateKey: remote ? `${remote.version}:${remote.buildId ?? ''}` : 'sw',
    releaseNotes: remote?.releaseNotes ?? '',
    apply,
  };
}
