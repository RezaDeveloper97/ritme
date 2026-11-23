import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  autopilotWanted,
  requestAutopilotRuntime,
  resetAutopilotWake,
  setAutopilotRuntimeReady,
  subscribeAutopilotWake,
  whenAutopilotRuntimeReady,
} from './wake';

afterEach(() => {
  resetAutopilotWake();
  vi.useRealTimers();
});

describe('autopilot lazy wake', () => {
  it('is not wanted until something asks, and wakes each listener exactly once', () => {
    const wake = vi.fn();
    subscribeAutopilotWake(wake);
    expect(autopilotWanted()).toBe(false);
    expect(wake).not.toHaveBeenCalled();
    requestAutopilotRuntime();
    requestAutopilotRuntime();
    expect(autopilotWanted()).toBe(true);
    expect(wake).toHaveBeenCalledTimes(1);
  });

  it('an unsubscribed listener is not woken', () => {
    const wake = vi.fn();
    subscribeAutopilotWake(wake)();
    requestAutopilotRuntime();
    expect(wake).not.toHaveBeenCalled();
  });

  it('resolves true when the runtime mounts', async () => {
    const p = whenAutopilotRuntimeReady(1000);
    setAutopilotRuntimeReady(true);
    await expect(p).resolves.toBe(true);
    await expect(whenAutopilotRuntimeReady(1000)).resolves.toBe(true);
  });

  it('resolves false (not_ready) when the chunk never arrives', async () => {
    vi.useFakeTimers();
    const p = whenAutopilotRuntimeReady(5000);
    vi.advanceTimersByTime(5000);
    await expect(p).resolves.toBe(false);
  });

  it('an unmount makes later waiters wait again', async () => {
    vi.useFakeTimers();
    setAutopilotRuntimeReady(true);
    setAutopilotRuntimeReady(false);
    const p = whenAutopilotRuntimeReady(10);
    vi.advanceTimersByTime(10);
    await expect(p).resolves.toBe(false);
  });
});
