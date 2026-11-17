import { describe, expect, it } from 'vitest';

import { parseAutopilotScript, type PilotCatalog, type PilotScript, type TargetRef } from '@/shared/pilot';

import { defineAutopilotConfig, resolveConfig, type AutopilotConfigInput } from '../config';
import { fail, OK, RunControl, runScript, type ActResult, type PilotEnv } from './runner';

const CATALOG: PilotCatalog = {
  routes: [],
  sheets: [{ sheet: 'log', titleKey: 'x' }],
  targets: [
    { id: 'log.save', value: 'action', surface: { sheet: 'log' } },
    { id: 'log.cat:*', value: 'section', surface: { sheet: 'log' } },
    { id: 'log.param:*', value: 'single', surface: { sheet: 'log' }, reveal: (key) => [{ do: 'expand', target: `log.cat:${key.split('.')[0]}` }] },
  ],
};

/** A screen made of sets: which sheet is open, which sections are expanded, what was done to it. */
function fakeApp(over: Partial<PilotEnv> = {}) {
  const calls: string[] = [];
  const state = { sheet: false, open: new Set<string>(), locked: false, slept: 0 };
  const key = (t: TargetRef) => (typeof t === 'string' ? t : JSON.stringify(t));
  const exists = (target: TargetRef): boolean => {
    const id = key(target);
    if (id.startsWith('log.cat:')) return state.sheet && id !== 'log.cat:ghost';
    if (id.startsWith('log.param:')) return state.sheet && state.open.has(id.slice('log.param:'.length).split('.')[0]);
    if (id === 'log.save') return state.sheet;
    return id === 'nav.log';
  };
  const env: PilotEnv = {
    sleep: async (ms) => {
      state.slept += ms;
    },
    locked: () => state.locked,
    exists,
    waitFor: async (target, wanted) => exists(target) === (wanted === 'visible'),
    surfaceOpen: (surface) => (surface.sheet ? state.sheet : true),
    open: async (where) => {
      calls.push('sheet' in where ? `open sheet ${where.sheet}` : `open ${where.route}`);
      if ('sheet' in where) state.sheet = true;
      return OK;
    },
    close: async () => {
      calls.push('close');
      state.sheet = false;
      return OK;
    },
    act: async (verb, target, value): Promise<ActResult> => {
      const id = key(target);
      calls.push(`${verb} ${id}${verb === 'set' ? '=' + JSON.stringify(value) : ''}`);
      if (verb === 'expand') state.open.add(id.slice('log.cat:'.length));
      if (verb === 'set' && value === 'bad') return fail('invalid_value');
      return OK;
    },
    caption: (text) => {
      if (text) calls.push(`say ${text}`);
    },
    progress: () => {},
    confirm: async () => true,
    ...over,
  };
  return { env, calls, state };
}

function parse(steps: unknown[], extra: Record<string, unknown> = {}): PilotScript {
  const parsed = parseAutopilotScript({ version: 1, steps, ...extra });
  if (!parsed.ok) throw new Error(parsed.issues.join('; '));
  return parsed.script;
}

function start(steps: unknown[], opts: { config?: AutopilotConfigInput; env?: Partial<PilotEnv>; preset?: 'human' | 'instant' } = {}) {
  const app = fakeApp(opts.env);
  const control = new RunControl();
  const config = resolveConfig(defineAutopilotConfig(opts.config), { run: { preset: opts.preset ?? 'instant' } });
  const done = runScript({ script: parse(steps), config, catalog: CATALOG, env: app.env, control, rng: () => 0.5 });
  return { ...app, control, done };
}

describe('runScript', () => {
  it('reaches a field by itself: opens its sheet, expands its section, then sets it', async () => {
    const run = start([{ do: 'set', target: 'log.param:bleeding.flow', value: 'heavy' }]);
    const report = await run.done;
    expect(run.calls).toEqual(['open sheet log', 'expand log.cat:bleeding', 'set log.param:bleeding.flow="heavy"']);
    expect(report).toEqual({ status: 'done', steps: [{ index: 0, do: 'set', target: 'log.param:bleeding.flow', status: 'ok' }] });
  });

  it('does not reopen or re-expand what is already there', async () => {
    const run = start([
      { do: 'set', target: 'log.param:bleeding.flow', value: 'heavy' },
      { do: 'set', target: 'log.param:bleeding.color', value: 'red' },
      { do: 'tap', target: 'log.save' },
    ]);
    await run.done;
    expect(run.calls.filter((c) => c.startsWith('open') || c.startsWith('expand'))).toEqual(['open sheet log', 'expand log.cat:bleeding']);
  });

  it('runs a page-only script', async () => {
    const run = start([{ do: 'open', route: '/calendar' }]);
    expect((await run.done).status).toBe('done');
    expect(run.calls).toEqual(['open /calendar']);
  });

  it('stops at the first failed step and never reports the value', async () => {
    const run = start([
      { do: 'set', target: 'log.param:note.text', value: 'bad' },
      { do: 'tap', target: 'log.save' },
    ]);
    const report = await run.done;
    expect(report.status).toBe('failed');
    expect(report.code).toBe('step_failed');
    expect(report.steps).toEqual([{ index: 0, do: 'set', target: 'log.param:note.text', status: 'failed', code: 'invalid_value' }]);
    expect(JSON.stringify(report)).not.toContain('bad');
    expect(run.calls).not.toContain('tap log.save');
  });

  it('skips an optional step, or every failed step with onStepError: skip', async () => {
    const optional = start([
      { do: 'tap', target: 'log.cat:ghost', optional: true },
      { do: 'tap', target: 'log.save' },
    ]);
    const report = await optional.done;
    expect(report.status).toBe('done');
    expect(report.steps.map((s) => [s.status, s.code])).toEqual([['skipped', 'target_not_found'], ['ok', undefined]]);

    const lenient = start([{ do: 'tap', target: 'log.cat:ghost' }, { do: 'tap', target: 'log.save' }], { config: { errors: { onStepError: 'skip' } } });
    expect((await lenient.done).status).toBe('done');
  });

  it('retries a target that was not there yet', async () => {
    let tries = 0;
    const run = start([{ do: 'tap', target: 'nav.log' }], {
      config: { errors: { retries: 2 } },
      env: {
        exists: () => false,
        waitFor: async () => {
          tries += 1;
          return tries >= 3;
        },
      },
    });
    expect((await run.done).status).toBe('done');
    expect(tries).toBe(3);
  });

  it('refuses a blocked target at run time too', async () => {
    const run = start([{ do: 'tap', target: 'log.save' }], { config: { safety: { blockedTargets: ['log.save'] } } });
    const report = await run.done;
    expect(report.steps[0]).toMatchObject({ status: 'failed', code: 'blocked' });
    expect(run.calls).toEqual([]);
  });

  it('asks before a confirm target and honours a no', async () => {
    const asked: string[] = [];
    const run = start([{ do: 'tap', target: 'log.save' }], {
      config: { safety: { confirmTargets: ['log.save'] } },
      env: {
        confirm: async (id) => {
          asked.push(id);
          return false;
        },
      },
    });
    const report = await run.done;
    expect(asked).toEqual(['log.save']);
    expect(report.steps[0]).toMatchObject({ status: 'failed', code: 'declined' });
    expect(run.calls).not.toContain('tap log.save');
  });

  it('does not open a blocked section to reach a field inside it', async () => {
    const run = start([{ do: 'set', target: 'log.param:bleeding.flow', value: 'heavy' }], { config: { safety: { blockedTargets: ['log.cat:*'] } } });
    expect((await run.done).steps[0]).toMatchObject({ status: 'failed', code: 'blocked' });
    expect(run.calls).toEqual(['open sheet log']);
  });

  it('asks only once when the step is retried', async () => {
    let asked = 0;
    let acts = 0;
    const run = start([{ do: 'tap', target: 'log.save' }], {
      config: { safety: { confirmTargets: ['log.save'] }, errors: { retries: 2 } },
      env: {
        confirm: async () => {
          asked += 1;
          return true;
        },
        act: async () => (++acts < 3 ? fail('not_verified') : OK),
      },
    });
    expect((await run.done).status).toBe('done');
    expect([asked, acts]).toEqual([1, 3]);
  });

  it('stops when the app lock is closed', async () => {
    const run = start([{ do: 'tap', target: 'nav.log' }], { env: { locked: () => true } });
    expect(await run.done).toMatchObject({ status: 'failed', code: 'locked', steps: [] });
  });

  it('cancels mid-run, even inside a sleep', async () => {
    let release: () => void = () => {};
    const run = start([{ do: 'wait', ms: 5000 }, { do: 'tap', target: 'nav.log' }], {
      env: { sleep: () => new Promise<void>((resolve) => (release = resolve)) },
    });
    await Promise.resolve();
    run.control.cancel();
    const report = await run.done;
    release();
    expect(report).toMatchObject({ status: 'cancelled', code: 'user_cancelled', steps: [] });
    expect(run.calls).toEqual([]);
  });

  it('parks while paused and carries on after resume', async () => {
    const run = start([{ do: 'tap', target: 'nav.log' }, { do: 'open', route: '/calendar' }]);
    run.control.pause();
    await new Promise((resolve) => setTimeout(resolve, 5));
    expect(run.calls).toEqual([]);
    run.control.resume();
    expect((await run.done).status).toBe('done');
    expect(run.calls).toEqual(['tap nav.log', 'open /calendar']);
  });

  it('spends no time on pauses with the instant preset, and some with human', async () => {
    const instant = start([{ do: 'tap', target: 'nav.log', say: 'می‌زنم' }]);
    await instant.done;
    expect(instant.state.slept).toBe(0);
    expect(instant.calls).toEqual(['say می‌زنم', 'tap nav.log']);

    const human = start([{ do: 'tap', target: 'nav.log' }], { preset: 'human' });
    await human.done;
    expect(human.state.slept).toBeGreaterThan(400);

    const paced = start([{ do: 'tap', target: 'nav.log', pace: { beforeStepMs: 0, afterStepMs: 0 } }], { preset: 'human' });
    await paced.done;
    expect(paced.state.slept).toBe(0);
  });

  it('turns a thrown driver error into a failed step', async () => {
    const run = start([{ do: 'tap', target: 'nav.log' }], {
      env: {
        act: async () => {
          throw new Error('boom');
        },
      },
    });
    expect((await run.done).steps[0]).toMatchObject({ status: 'failed', code: 'error' });
  });
});
