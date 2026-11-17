import { describe, expect, it } from 'vitest';

import { parseAutopilotScript, type PilotScript } from '@/shared/pilot';

import { defineAutopilotConfig, paceFor, resolveConfig } from '../config';
import { planChoice, readNumber, stepperDirection, stepperProgressed, textOf } from './plan';
import { checkScript, isSafeRoute, needsConfirm } from './safety';
import { cursorTravelMs, jittered, typingDelay, typingFrames } from './timing';

const base = defineAutopilotConfig();

describe('config', () => {
  it('has safe defaults', () => {
    expect(base.enabled).toBe(true);
    expect(base.defaultPreset).toBe('human');
    expect(base.safety.allowRawSelectors).toBe(false);
    expect(base.typing.focus).toBe('visual');
    expect(base.presets.instant.typeCharMs).toBe(0);
  });

  it('layers preset < script options < run options', () => {
    expect(resolveConfig(base).preset).toBe('human');
    expect(resolveConfig(base, { scriptOptions: { preset: 'brisk' } }).preset).toBe('brisk');
    expect(resolveConfig(base, { scriptOptions: { preset: 'brisk' }, run: { preset: 'instant' } }).preset).toBe('instant');
    expect(resolveConfig(base, { run: { pace: { typeCharMs: 3 } } }).timing.typeCharMs).toBe(3);
  });

  it('ignores script options the config does not allow', () => {
    const resolved = resolveConfig(base, { scriptOptions: { preset: 'nope', pace: { typeCharMs: 1 }, safety: { allowRawSelectors: true } } });
    expect(resolved.preset).toBe('human');
    expect(resolved.timing.typeCharMs).toBe(base.presets.human.typeCharMs);
    expect(resolved.safety.allowRawSelectors).toBe(false);
    const open = defineAutopilotConfig({ safety: { scriptMayOverride: ['preset', 'pace'] } });
    expect(resolveConfig(open, { scriptOptions: { pace: { typeCharMs: 1 } } }).timing.typeCharMs).toBe(1);
  });

  it('goes still under reduced motion unless told to keep', () => {
    expect(resolveConfig(base, { reducedMotion: true })).toMatchObject({ preset: 'instant', still: true });
    const keep = defineAutopilotConfig({ reducedMotion: 'keep' });
    expect(resolveConfig(keep, { reducedMotion: true }).still).toBe(false);
  });

  it('applies a step pace, dropping junk', () => {
    expect(paceFor(base.presets.human, { hoverMs: 1 }).hoverMs).toBe(1);
    expect(paceFor(base.presets.human, { hoverMs: -5 })).toEqual(base.presets.human);
    expect(paceFor(base.presets.human, undefined)).toBe(base.presets.human);
  });
});

describe('timing', () => {
  it('jitters within bounds and not at all when off', () => {
    expect(jittered(100, 0.25, () => 0)).toBe(75);
    expect(jittered(100, 0.25, () => 0.999999)).toBe(125);
    expect(jittered(100, 0, () => 0.3)).toBe(100);
    expect(jittered(0, 0.5, () => 0.3)).toBe(0);
  });

  it('clamps pointer travel', () => {
    const t = base.presets.human;
    expect(cursorTravelMs(1, t)).toBe(t.cursorMinMs);
    expect(cursorTravelMs(100_000, t)).toBe(t.cursorMaxMs);
    expect(cursorTravelMs(450, t)).toBe(500);
    expect(cursorTravelMs(450, base.presets.instant)).toBe(0);
  });

  it('types by character, by word or at once', () => {
    expect(typingFrames('سلام', 'perChar')).toEqual(['س', 'سل', 'سلا', 'سلام']);
    expect(typingFrames('a bc', 'perWord')).toEqual(['a ', 'a bc']);
    expect(typingFrames('abc', 'instant')).toEqual(['abc']);
    expect(typingFrames('', 'perChar')).toEqual(['']);
  });

  it('never splits a ZWNJ, a diacritic or an emoji from its letter', () => {
    expect(typingFrames('می‌شود', 'perChar')).toEqual(['م', 'می‌', 'می‌ش', 'می‌شو', 'می‌شود']);
    expect(typingFrames('😀a', 'perChar')).toEqual(['😀', '😀a']);
  });

  it('pauses longer after a word', () => {
    const t = { ...base.presets.human, jitter: 0 };
    expect(typingDelay('ab', t, () => 0.5)).toBe(t.typeCharMs);
    expect(typingDelay('ab ', t, () => 0.5)).toBe(t.typeCharMs + t.typeWordPauseMs);
  });
});

describe('plan', () => {
  it('reads numbers in any digit script', () => {
    expect(readNumber('۳۶٫۵ °C')).toBe(36.5);
    expect(readNumber('٧ روز')).toBe(7);
    expect(readNumber('12 days')).toBe(12);
    expect(readNumber('۳۶/۵')).toBe(36.5);
    expect(readNumber('—')).toBeNull();
  });

  it('plans a single choice: press only the wanted chip', () => {
    expect(planChoice(['a', 'b', 'c'], ['a'], ['b'], true)).toEqual({ press: ['b'], unknown: [] });
    expect(planChoice(['a', 'b'], ['a'], ['a'], true)).toEqual({ press: [], unknown: [] });
    expect(planChoice(['a', 'b'], ['a'], [], true)).toEqual({ press: ['a'], unknown: [] });
  });

  it('plans a multi choice as the exact set: off first, then on', () => {
    expect(planChoice(['a', 'b', 'c'], ['a', 'b'], ['b', 'c'], false)).toEqual({ press: ['a', 'c'], unknown: [] });
    expect(planChoice(['a', 'b'], [], ['a', 'b'], false).press).toEqual(['a', 'b']);
  });

  it('reports codes the group does not offer', () => {
    expect(planChoice(['a'], [], ['zzz'], true)).toEqual({ press: [], unknown: ['zzz'] });
  });

  it('steers a stepper', () => {
    expect(stepperDirection(5, 8)).toBe('up');
    expect(stepperDirection(5, 2)).toBe('down');
    expect(stepperDirection(36.5, 36.5)).toBeNull();
    expect(stepperProgressed(5, 6, 8)).toBe(true);
    expect(stepperProgressed(5, 5, 8)).toBe(false);
    expect(stepperProgressed(7.5, 8.5, 8)).toBe(false);
    expect(stepperProgressed(null, 36.5, 37)).toBe(true);
  });

  it('turns values into text', () => {
    expect(textOf(7.5)).toBe('7.5');
    expect(textOf(null)).toBe('');
    expect(textOf('a')).toBe('a');
  });
});

function scriptOf(steps: unknown[]): PilotScript {
  const parsed = parseAutopilotScript({ version: 1, steps });
  if (!parsed.ok) throw new Error(parsed.issues.join('; '));
  return parsed.script;
}

describe('safety', () => {
  it('only lets in-app routes through', () => {
    for (const ok of ['/calendar', '/fertility/log', '/']) expect(isSafeRoute(ok, base.safety)).toBe(true);
    for (const bad of ['https://evil.example', '//evil.example', 'calendar', '/a/../b', '/a?x=1', '/a#b', 'javascript:alert(1)', '/home/%2e%2e/otp', '/dev/autopilot', '/otp', '/onboarding/name', '/a b']) {
      expect(isSafeRoute(bad, base.safety)).toBe(false);
    }
    const narrow = defineAutopilotConfig({ safety: { allowedRoutes: ['/log', '/calendar'] } });
    expect(isSafeRoute('/log/customize', narrow.safety)).toBe(true);
    expect(isSafeRoute('/logs', narrow.safety)).toBe(false);
    expect(isSafeRoute('/profile', narrow.safety)).toBe(false);
  });

  it('flags unsafe steps and oversize scripts', () => {
    const config = defineAutopilotConfig({ safety: { maxSteps: 2, blockedTargets: ['profile.delete', 'pay.pay:*'] } });
    const issues = checkScript(
      scriptOf([
        { do: 'open', route: 'https://evil.example' },
        { do: 'tap', target: 'profile.delete' },
        { do: 'tap', target: 'pay.pay:now' },
        { do: 'tap', target: { css: 'button' } },
        { do: 'tap', target: { role: 'button', name: 'x' } },
        { do: 'tap', target: 'log.save' },
      ]),
      config.safety,
    );
    expect(issues).toEqual([
      { step: -1, code: 'too_many_steps' },
      { step: 0, code: 'route_not_allowed' },
      { step: 1, code: 'target_blocked' },
      { step: 2, code: 'target_blocked' },
      { step: 3, code: 'selector_not_allowed' },
      { step: 4, code: 'selector_not_allowed' },
    ]);
  });

  it('knows which steps need a yes', () => {
    const config = defineAutopilotConfig({ safety: { confirmTargets: ['log.save'] } });
    const [save, other] = scriptOf([{ do: 'tap', target: 'log.save' }, { do: 'tap', target: 'nav.log' }]).steps;
    expect(needsConfirm(save, config.safety)).toBe(true);
    expect(needsConfirm(other, config.safety)).toBe(false);
  });
});
