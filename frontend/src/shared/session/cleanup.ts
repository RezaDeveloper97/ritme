/**
 * Per-user data that must not outlive the session.
 *
 * Every session end — explicit logout, account deletion, or a 401 that
 * `endsSession()` trusts — goes through `clearAuthToken()`, which runs the
 * callbacks registered here. A slice that keeps per-user state on the device
 * (a persisted store, a queue) registers its own wipe, so `shared` never has
 * to know the slice's storage keys (CLAUDE.md §3). Device preferences (theme,
 * locale, dismissed prompts) are not per-user and must not register.
 *
 * Never `localStorage.clear()` here (CLAUDE.md §11.1): it would also take the
 * device preferences, and anything another origin-mate stored.
 */

import { clearHandoff } from '@/shared/lib/handoff';
import { clearAllLocalFiles } from '@/shared/lib/local-files';
import { getOutbox, type SendResult } from '@/shared/lib/outbox';

type Cleanup = () => void | Promise<void>;

const cleanups = new Set<Cleanup>();

/** Run `cleanup` whenever a session ends. Returns an unregister function. */
export function onSessionEnd(cleanup: Cleanup): () => void {
  cleanups.add(cleanup);
  return () => {
    cleanups.delete(cleanup);
  };
}

/**
 * Run every registered wipe. One failing wipe (blocked storage, a closed
 * IndexedDB) must not stop the others, nor the logout itself.
 */
export function runSessionCleanups(): void {
  for (const cleanup of cleanups) {
    try {
      const result = cleanup();
      if (result && typeof result.then === 'function') result.catch(() => undefined);
    } catch {
      // Keep going: the remaining per-user data still has to go.
    }
  }
}

/**
 * The offline outbox (`shared/lib/outbox`) holds the ended session's queued
 * health writes. Replayed after the next sign-in they would land in the new
 * account, so they go with the session. A `send` that answers `rejected`
 * removes every entry; the second pass covers a replay that was already in
 * flight (concurrent calls share that run instead of starting ours).
 */
async function dropQueuedWrites(): Promise<void> {
  const outbox = getOutbox();
  const drop = async (): Promise<SendResult> => 'rejected';
  await outbox.replay(drop);
  await outbox.replay(drop);
}

onSessionEnd(dropQueuedWrites);

/*
 * On-device files (checkup report photos / PDFs in `shared/lib/local-files`)
 * are the ended user's health data. The whole database goes, not one slice's
 * namespace: a slice that registered its own wipe would only be heard if its
 * module happened to be loaded in this page (security audit M3-M7 #1).
 */
onSessionEnd(clearAllLocalFiles);

/* A pending one-time navigation prefill (`shared/lib/handoff`) belongs to this session too. */
onSessionEnd(clearHandoff);
