import { describe, expect, it, vi } from 'vitest';

import { BATCH_SIZE, createTracker, OPT_OUT_MS } from './track';

function setup(tracking = true, authed = true) {
  let t = 1_000_000;
  const send = vi.fn(async () => ({ tracking }));
  const mem = new Map<string, string>();
  const tr = createTracker({
    send,
    isAuthenticated: () => authed,
    now: () => t,
    storage: { getItem: (k) => mem.get(k) ?? null, setItem: (k, v) => void mem.set(k, v), removeItem: (k) => void mem.delete(k) },
  });
  return { tr, send, advance: (ms: number) => (t += ms) };
}

describe('analytics tracker', () => {
  it('batches events and sends names plus enumerated props only', async () => {
    const { tr, send } = setup();
    tr.track({ name: 'app_open' });
    tr.track({ name: 'log_saved', props: { source: 'quick' } });
    await tr.flush();
    expect(send).toHaveBeenCalledTimes(1);
    const sent = (send.mock.calls[0] as unknown[])[0] as { name: string; props?: unknown }[];
    expect(sent.map((e) => e.name)).toEqual(['app_open', 'log_saved']);
    expect(sent[1].props).toEqual({ source: 'quick' });
    expect(Object.keys(sent[0]).sort()).toEqual(['name', 'occurred_at']);
  });

  it('flushes at the batch size', () => {
    const { tr, send } = setup();
    for (let i = 0; i < BATCH_SIZE; i++) tr.track({ name: 'app_open' });
    expect(send).toHaveBeenCalledTimes(1);
  });

  it('stops for six hours after tracking:false, and again after a consent change', async () => {
    const { tr, send, advance } = setup(false);
    tr.track({ name: 'app_open' });
    await tr.flush();
    tr.track({ name: 'app_open' });
    await tr.flush();
    expect(send).toHaveBeenCalledTimes(1);
    tr.consentChanged();
    tr.track({ name: 'app_open' });
    await tr.flush();
    expect(send).toHaveBeenCalledTimes(2);
    advance(OPT_OUT_MS + 1);
    tr.track({ name: 'app_open' });
    await tr.flush();
    expect(send).toHaveBeenCalledTimes(3);
  });

  it('sends nothing without a session and never throws on a failing send', async () => {
    const a = setup(true, false);
    a.tr.track({ name: 'app_open' });
    await a.tr.flush();
    expect(a.send).not.toHaveBeenCalled();
    const { tr, send } = setup();
    send.mockRejectedValueOnce(new Error('boom'));
    tr.track({ name: 'app_open' });
    await expect(tr.flush()).resolves.toBeUndefined();
  });
});
