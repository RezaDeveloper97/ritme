/**
 * Store billing through the Android shell's bridge (`window.RitmeNative`,
 * bridgeVersion >= 2): Cafe Bazaar / Myket in-app purchases. The shell answers
 * asynchronously by calling `window.__ritmeBilling.onResult(result)`; this
 * module turns that into promises, keyed by `requestId` so several requests can
 * be in flight. Anywhere else (browser, PWA, old shell) the channel is `web`.
 */
export type BillingChannel = 'web' | 'bazaar' | 'myket';
export type StoreChannel = Exclude<BillingChannel, 'web'>;

export interface StorePurchase {
  signed_data: string;
  signature: string;
  sku: string;
  token: string;
}

export type BillingError = 'cancelled' | 'unavailable' | 'failed';

export interface BillingResult {
  requestId: string;
  ok: boolean;
  error?: BillingError;
  purchases?: StorePurchase[];
}

interface BillingBridge {
  bridgeVersion?: () => number;
  billingChannel?: () => string;
  purchase?: (requestId: string, sku: string, payload: string) => void;
  queryPurchases?: (requestId: string) => void;
}

interface BillingWindow {
  RitmeNative?: BillingBridge;
  __ritmeBilling?: { onResult: (result: BillingResult) => void };
}

export const PURCHASE_TIMEOUT_MS = 5 * 60_000;
export const QUERY_TIMEOUT_MS = 20_000;

/** A store request that did not complete; `reason` is the shell's code (or `timeout`). */
export class StoreBillingError extends Error {
  constructor(readonly reason: BillingError | 'timeout') {
    super(`store billing: ${reason}`);
    this.name = 'StoreBillingError';
  }
}

const pending = new Map<string, (result: BillingResult) => void>();

function win(): BillingWindow | null {
  return typeof window === 'undefined' ? null : (window as unknown as BillingWindow);
}

function bridge(): BillingBridge | null {
  const b = win()?.RitmeNative;
  if (!b) return null;
  try {
    return typeof b.bridgeVersion === 'function' && b.bridgeVersion() >= 2 ? b : null;
  } catch {
    return null;
  }
}

export function getBillingChannel(): BillingChannel {
  const b = bridge();
  if (!b || typeof b.billingChannel !== 'function') return 'web';
  try {
    const c = b.billingChannel();
    return c === 'bazaar' || c === 'myket' ? c : 'web';
  } catch {
    return 'web';
  }
}

/** Dispatches a shell result to the request that is waiting for it (unknown ids are ignored). */
function onResult(result: BillingResult): void {
  const resolve = result && pending.get(result.requestId);
  if (resolve) resolve(result);
}

function installReceiver(): void {
  const w = win();
  if (!w) return;
  // Re-assigned when something else replaced it; the dispatch itself is stateless.
  if (w.__ritmeBilling?.onResult !== onResult) w.__ritmeBilling = { onResult };
}

let counter = 0;
function newRequestId(): string {
  counter += 1;
  return `b${Date.now().toString(36)}${counter}`;
}

function request(
  timeoutMs: number,
  start: (b: BillingBridge, requestId: string) => void,
): Promise<BillingResult> {
  return new Promise((resolve, reject) => {
    const b = bridge();
    if (!b) {
      reject(new StoreBillingError('unavailable'));
      return;
    }
    installReceiver();
    const requestId = newRequestId();
    const timer = setTimeout(() => {
      pending.delete(requestId);
      reject(new StoreBillingError('timeout'));
    }, timeoutMs);
    pending.set(requestId, (result) => {
      clearTimeout(timer);
      pending.delete(requestId);
      if (result.ok) resolve(result);
      else reject(new StoreBillingError(result.error ?? 'failed'));
    });
    try {
      start(b, requestId);
    } catch {
      clearTimeout(timer);
      pending.delete(requestId);
      reject(new StoreBillingError('failed'));
    }
  });
}

/** Opens the store's purchase screen for `sku`; resolves with the purchases the shell reports. */
export async function storePurchase(sku: string, payload: string): Promise<StorePurchase[]> {
  const r = await request(PURCHASE_TIMEOUT_MS, (b, id) => b.purchase?.(id, sku, payload));
  return r.purchases ?? [];
}

/** The purchases the store account currently owns (for restore). */
export async function storeQueryPurchases(): Promise<StorePurchase[]> {
  const r = await request(QUERY_TIMEOUT_MS, (b, id) => b.queryPurchases?.(id));
  return r.purchases ?? [];
}
