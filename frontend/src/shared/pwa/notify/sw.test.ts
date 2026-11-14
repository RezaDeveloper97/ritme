import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import vm from 'node:vm';
import { describe, expect, it } from 'vitest';

/** Runs scripts/sw.template.js against a stub worker scope and drives its push handlers. */
function boot() {
  const handlers: Record<string, (event: unknown) => void> = {};
  const shown: Array<{ title: string; options: Record<string, unknown> }> = [];
  const opened: string[] = [];
  const navigated: string[] = [];
  let windows: Array<{ url: string; focus: () => Promise<unknown>; navigate: (u: string) => Promise<unknown> }> = [];
  const self = {
    location: { origin: 'https://ritme.app' },
    addEventListener: (type: string, fn: (event: unknown) => void) => {
      handlers[type] = fn;
    },
    registration: {
      showNotification: (title: string, options: Record<string, unknown>) => {
        shown.push({ title, options });
        return Promise.resolve();
      },
    },
    clients: {
      matchAll: () => Promise.resolve(windows),
      openWindow: (url: string) => {
        opened.push(url);
        return Promise.resolve();
      },
    },
  };
  const source = readFileSync(join(process.cwd(), 'scripts/sw.template.js'), 'utf8');
  vm.runInNewContext(source, { self, URL, caches: {}, Response: {}, Request: class {}, setTimeout, clearTimeout });
  return {
    handlers,
    shown,
    opened,
    navigated,
    setWindows(urls: string[]) {
      windows = urls.map((url) => ({
        url,
        focus: () => Promise.resolve(undefined),
        navigate: (u: string) => {
          navigated.push(u);
          return Promise.resolve();
        },
      }));
    },
  };
}

async function fire(handler: (event: unknown) => void, event: Record<string, unknown>) {
  let pending: Promise<unknown> = Promise.resolve();
  handler({ ...event, waitUntil: (p: Promise<unknown>) => (pending = p) });
  await pending;
}

describe('service worker push', () => {
  it('shows the payload with the in-app url and rtl for fa', async () => {
    const sw = boot();
    await fire(sw.handlers.push, {
      data: { json: () => ({ title: 'دارو', body: 'وقت قرص است', url: '/fa/reminders', tag: 'dose-1' }) },
    });
    expect(sw.shown).toHaveLength(1);
    expect(sw.shown[0].title).toBe('دارو');
    expect(sw.shown[0].options).toMatchObject({ body: 'وقت قرص است', tag: 'dose-1', dir: 'rtl', lang: 'fa', data: { url: '/fa/reminders' } });
  });
  it('still shows a neutral notification for a missing or malformed payload', async () => {
    const sw = boot();
    await fire(sw.handlers.push, { data: null });
    await fire(sw.handlers.push, {
      data: {
        json: () => {
          throw new Error('bad json');
        },
      },
    });
    expect(sw.shown).toHaveLength(2);
    expect(sw.shown[0].title).toBe('ریتمی');
    expect(sw.shown[0].options.body).toBe('');
  });
  it('drops an off-origin url from the payload', async () => {
    const sw = boot();
    await fire(sw.handlers.push, { data: { json: () => ({ title: 'x', url: '//evil.com/x' }) } });
    expect(sw.shown[0].options.data).toEqual({ url: '/' });
  });
});

describe('service worker notificationclick', () => {
  const click = (url: unknown) => ({ notification: { close: () => undefined, data: { url } } });

  it('focuses and navigates an existing window', async () => {
    const sw = boot();
    sw.setWindows(['https://ritme.app/fa/home']);
    await fire(sw.handlers.notificationclick, click('/fa/reminders'));
    expect(sw.navigated).toEqual(['https://ritme.app/fa/reminders']);
    expect(sw.opened).toEqual([]);
  });
  it('opens a window when none exists', async () => {
    const sw = boot();
    await fire(sw.handlers.notificationclick, click('/fa/reminders'));
    expect(sw.opened).toEqual(['https://ritme.app/fa/reminders']);
  });
  it('never navigates off origin', async () => {
    const sw = boot();
    await fire(sw.handlers.notificationclick, click('https://evil.com'));
    expect(sw.opened).toEqual(['https://ritme.app/']);
  });
});
