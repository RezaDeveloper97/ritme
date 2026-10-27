import { describe, expect, it } from 'vitest';

import en from '../../../../messages/en/home.json';
import fa from '../../../../messages/fa/home.json';

// Ring limits (E-17): the centre of the ring is small, so label ≤ 14, unit ≤ 10, subtitle ≤ 18 characters.
const LIMITS = { label: 14, unit: 10, subtitle: 18 } as const;

const KEYS: Record<keyof typeof LIMITS, string[]> = {
  label: ['ring.nextPeriod', 'ring.period', 'ring.noData', 'nb.hero.late', 'nb.hero.periodIn'],
  unit: ['ring.daysLeft', 'nb.hero.lateUnit'],
  subtitle: ['ring.usually', 'ring.cycleDay'],
};

/** Longest rendering of an ICU string: the `other` plural branch, numbers as two digits. */
function longest(message: string): string {
  const plural = message.replace(/\{\w+, plural,[^]*?other\{([^}]*)\}\}/g, '$1');
  return plural.replace(/#/g, '12').replace(/\{\w+(?:, \w+)?\}/g, '12');
}

function get(obj: unknown, path: string): string {
  const v = path.split('.').reduce<unknown>((o, k) => (o as Record<string, unknown>)?.[k], obj);
  if (typeof v !== 'string') throw new Error(`missing message ${path}`);
  return v;
}

describe('ring copy limits', () => {
  for (const [name, messages] of [['fa', fa], ['en', en]] as const) {
    for (const kind of Object.keys(LIMITS) as (keyof typeof LIMITS)[]) {
      for (const key of KEYS[kind]) {
        it(`${name} ${key} fits the ${kind} limit (${LIMITS[kind]})`, () => {
          expect([...longest(get(messages, key))].length).toBeLessThanOrEqual(LIMITS[kind]);
        });
      }
    }
  }
});
