import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { LockSnapshot } from '../model/controller';

// The gate's only inputs: the lock snapshot and the lock screen. Both are
// stubbed so the test pins the one guarantee — while locked, the app's tree
// (whatever route or history entry it is) is not rendered at all.
let snapshot: LockSnapshot | null = null;
/** What the page's controller knows (it exists only in the browser). */
let controllerState: LockSnapshot | null = null;
vi.mock('../model/store', () => ({
  useAppLock: () => snapshot,
  getLockController: () =>
    controllerState && {
      getSnapshot: () => controllerState,
      subscribe: () => () => undefined,
    },
}));
vi.mock('./LockScreen', () => ({ LockScreen: () => createElement('div', { 'data-testid': 'lock' }, 'LOCK') }));
vi.mock('@/shared/session', () => ({ isAuthenticated: () => true }));

const { AppLockGate } = await import('./AppLockGate');

const base: LockSnapshot = {
  enabled: true,
  locked: true,
  biometric: false,
  timeoutMin: 1,
  length: 4,
  blockedUntil: 0,
  lockedOut: false,
  hidePreview: false,
  hidden: false,
};

const render = () => renderToStaticMarkup(createElement(AppLockGate, null, createElement('main', null, 'HEALTH-DATA')));

describe('AppLockGate', () => {
  beforeEach(() => {
    snapshot = null;
    controllerState = null;
  });

  it('undecided (hydration) on a locked device: the app tree does not render', () => {
    controllerState = base; // useAppLock() still reports null during hydration
    expect(render()).not.toContain('HEALTH-DATA');
  });

  it('undecided on the server (no controller): the server HTML is rendered, hidden by the pre-paint marker', () => {
    expect(render()).toContain('HEALTH-DATA');
  });

  it('renders only the lock screen while locked', () => {
    snapshot = base;
    const html = render();
    expect(html).toContain('LOCK');
    expect(html).not.toContain('HEALTH-DATA');
  });

  it('renders the app once unlocked', () => {
    snapshot = { ...base, locked: false };
    controllerState = snapshot;
    expect(render()).toContain('HEALTH-DATA');
  });

  it('shows the preview veil while hidden when the option is on', () => {
    snapshot = { ...base, locked: false, hidePreview: true, hidden: true };
    controllerState = snapshot;
    expect(render()).toContain('app-veil');
  });
});
