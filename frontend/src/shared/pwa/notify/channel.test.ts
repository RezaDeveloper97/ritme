import { describe, expect, it } from 'vitest';

import { detectChannel, type ChannelEnv, nativeState, needsInstallForPush, webPushState } from './channel';

const base: ChannelEnv = {
  userAgent: 'Mozilla/5.0 (X11; Linux x86_64) Chrome/130',
  hasNativeBridge: false,
  hasServiceWorker: true,
  hasPushManager: true,
  hasNotification: true,
  standalone: false,
};
const IPHONE = 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Safari/604.1';

describe('detectChannel', () => {
  it('uses the native bridge inside the shell', () => {
    expect(detectChannel({ ...base, hasNativeBridge: true, hasPushManager: false })).toBe('native');
  });
  it('never uses web push inside an old shell without the bridge', () => {
    expect(detectChannel({ ...base, userAgent: 'Mozilla/5.0 Chrome/130 RitmeApp/1.0' })).toBe('unsupported');
  });
  it('uses web push in a capable browser', () => {
    expect(detectChannel(base)).toBe('webpush');
  });
  it('is unsupported without PushManager, service workers or Notification', () => {
    expect(detectChannel({ ...base, hasPushManager: false })).toBe('unsupported');
    expect(detectChannel({ ...base, hasServiceWorker: false })).toBe('unsupported');
    expect(detectChannel({ ...base, hasNotification: false })).toBe('unsupported');
  });
});

describe('needsInstallForPush', () => {
  const ios = { ...base, userAgent: IPHONE, hasPushManager: false };
  it('asks iOS Safari tabs to install first', () => {
    expect(needsInstallForPush(ios)).toBe(true);
  });
  it('does not once installed, or off iOS, or in the shell', () => {
    expect(needsInstallForPush({ ...ios, standalone: true })).toBe(false);
    expect(needsInstallForPush({ ...base, hasPushManager: false })).toBe(false);
    expect(needsInstallForPush({ ...ios, hasNativeBridge: true })).toBe(false);
  });
  it('treats iPadOS (Mac UA + touch) as iOS', () => {
    const ipad = { ...base, userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605', hasPushManager: false };
    expect(needsInstallForPush(ipad, 5)).toBe(true);
    expect(needsInstallForPush(ipad, 0)).toBe(false);
  });
});

describe('state mapping', () => {
  it('maps web push permission + subscription', () => {
    expect(webPushState('denied', false)).toBe('blocked');
    expect(webPushState('default', false)).toBe('off');
    expect(webPushState('granted', false)).toBe('off');
    expect(webPushState('granted', true)).toBe('on');
  });
  it('maps the bridge state', () => {
    expect(nativeState('granted')).toBe('on');
    expect(nativeState('denied')).toBe('blocked');
    expect(nativeState('prompt')).toBe('off');
    expect(nativeState('whatever')).toBe('off');
  });
});
