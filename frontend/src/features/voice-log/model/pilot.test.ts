import { describe, expect, it } from 'vitest';

import type { LogCategory, LogDayValues } from '@/entities/health-log';
import { MANIFEST_GUIDE, parseAutopilotScript, type ManifestTarget } from '@/shared/pilot';

import { logSheetManifest, MAX_VOICE_STEPS, voiceScript } from './pilot';

const targets: ManifestTarget[] = [
  { id: 'log.save', kind: 'action', label: 'Save' },
  { id: 'log.cat:mood', kind: 'section', label: 'Mood' },
  { id: 'log.param:mood.moods', kind: 'multi', label: 'Mood', options: [{ value: 'happy', label: 'Happy' }] },
  { id: 'log.param:symptoms.general', kind: 'multi', label: 'General', options: [{ value: 'fatigue', label: 'Fatigue' }] },
  { id: 'log.level:symptoms.general', kind: 'single', label: 'General', options: [{ value: 'mild', label: 'Mild' }] },
  { id: 'log.param:bleeding.flow', kind: 'single', label: 'Flow', options: [{ value: 'heavy', label: 'Heavy' }] },
  { id: 'log.panel-param:pain.score', kind: 'single', label: 'Score' },
  { id: 'account.delete', kind: 'action', label: 'Delete' },
];

const categories = [{ code: 'mood' }, { code: 'symptoms' }, { code: 'bleeding' }] as unknown as LogCategory[];

describe('logSheetManifest', () => {
  it('keeps only the fields of the log sheet and offers nothing but set', () => {
    const m = logSheetManifest(targets);
    expect(m.targets.map((t) => t.id)).toEqual([
      'log.param:mood.moods',
      'log.param:symptoms.general',
      'log.level:symptoms.general',
      'log.param:bleeding.flow',
    ]);
    expect(m.actions).toEqual(['set']);
    expect(m.routes).toEqual([]);
    expect(m.sheets).toEqual([]);
    expect(m.guide).toBe(MANIFEST_GUIDE);
  });
});

describe('voiceScript', () => {
  const manifest = logSheetManifest(targets);
  const values = {
    mood: { moods: ['calm'] },
    symptoms: { general: { dizziness: { level: 'mild', score: null } } },
  } as unknown as LogDayValues;

  it('drops everything that is not a set on a listed field', () => {
    const script = voiceScript(
      {
        steps: [
          { do: 'open', route: '/profile' },
          { do: 'tap', target: 'log.save' },
          { do: 'set', target: 'account.delete', value: true },
          { do: 'set', target: { css: '.danger' }, value: true },
          { do: 'set', target: 'log.param:bleeding.flow', value: null },
          { do: 'set', target: 'log.param:bleeding.flow', value: 'heavy', say: '  خونریزی زیاد  ' },
          'nonsense',
        ],
      },
      manifest,
      values,
      categories,
    );
    expect(script).toEqual({
      version: 1,
      steps: [{ do: 'set', target: 'log.param:bleeding.flow', value: 'heavy', optional: true }],
    });
    expect(parseAutopilotScript(script).ok).toBe(true);
  });

  it('adds to a multi-select instead of replacing what is already picked', () => {
    const script = voiceScript(
      {
        steps: [
          { do: 'set', target: 'log.param:mood.moods', value: ['happy', 'calm'] },
          { do: 'set', target: 'log.param:symptoms.general', value: ['fatigue'] },
        ],
      },
      manifest,
      values,
      categories,
    );
    expect(script?.steps.map((s) => ('value' in s ? s.value : null))).toEqual([
      ['calm', 'happy'],
      ['dizziness', 'fatigue'],
    ]);
  });

  it('is null when nothing is left, and capped', () => {
    expect(voiceScript({ steps: [{ do: 'close' }] }, manifest, values, categories)).toBeNull();
    const many = Array.from({ length: MAX_VOICE_STEPS + 10 }, () => ({ do: 'set', target: 'log.param:bleeding.flow', value: 'heavy' }));
    expect(voiceScript({ steps: many }, manifest, values, categories)?.steps).toHaveLength(MAX_VOICE_STEPS);
  });
});
