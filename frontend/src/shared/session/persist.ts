'use client';

/**
 * Ask the browser to keep this origin's storage (and with it the session token)
 * out of automatic eviction under storage pressure. Best effort: unsupported,
 * refused, or already granted all end the same way — nothing to do.
 */
let requested = false;

export function requestPersistentStorage(): void {
  if (requested || typeof navigator === 'undefined') return;
  requested = true;
  const storage = navigator.storage;
  if (!storage?.persist || !storage.persisted) return;
  void storage
    .persisted()
    .then((already) => (already ? true : storage.persist()))
    .catch(() => undefined);
}
