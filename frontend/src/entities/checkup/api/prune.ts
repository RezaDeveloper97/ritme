import type { LocalFileStore } from '@/shared/lib/local-files';

import { checkupAttachments } from '../model/attachments';
import type { CheckupRecordPage } from '../model/types';
import { fetchCheckupRecords } from './queries';

/*
 * Orphaned report files (security audit M3-M7 #1). Deleting a custom checkup
 * type cascades its records on the server, and a record can also be deleted
 * from another device — either way the photo / PDF stays in this device's
 * IndexedDB with nothing pointing at it. This sweep keeps only files whose
 * record the server still lists as having an attachment.
 *
 * Conservative by design: nothing is deleted unless EVERY page of the user's
 * attachment records was read; any failure (offline, 401, 5xx) leaves the
 * files alone. It only fetches when there is at least one file on the device.
 */

/** Past this many pages (20 per page) we'd rather keep a stray file than keep paging. */
const MAX_PAGES = 50;
/** Home and History both run it; once a minute is plenty. */
const MIN_INTERVAL_MS = 60_000;

type FetchPage = (page: number) => Promise<CheckupRecordPage>;

const defaultFetchPage: FetchPage = (page) => fetchCheckupRecords({ filter: 'with_attachment' }, page);

/** Deletes the files whose record is gone. Resolves to how many went; rejects if a page failed. */
export async function pruneCheckupAttachments(
  store: LocalFileStore = checkupAttachments,
  fetchPage: FetchPage = defaultFetchPage,
): Promise<number> {
  const stored = await store.list();
  if (stored.length === 0) return 0;

  const alive = new Set<string>();
  for (let page = 1; ; page++) {
    if (page > MAX_PAGES) return 0;
    const res = await fetchPage(page);
    for (const record of res.records) alive.add(String(record.id));
    if (res.page >= res.lastPage) break;
  }

  const orphans = stored.filter((meta) => !alive.has(meta.key)).map((meta) => meta.key);
  await store.deleteMany(orphans);
  return orphans.length;
}

let inFlight: Promise<number> | null = null;
let lastRun = 0;

/**
 * Fire-and-forget form for screens: single-flight, at most once a minute, and
 * never throws (the next run retries). Resolves to the number of files removed.
 */
export function pruneCheckupAttachmentsSoon(now: () => number = Date.now): Promise<number> {
  if (inFlight) return inFlight;
  if (now() - lastRun < MIN_INTERVAL_MS) return Promise.resolve(0);
  lastRun = now();
  inFlight = pruneCheckupAttachments()
    .catch(() => 0)
    .finally(() => {
      inFlight = null;
    });
  return inFlight;
}
