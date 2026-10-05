import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { learnAge } from '../model/age';
import { learnViewSchema } from './schema';

/* Boundary contract of /children/{id}/learn against the Go goldens (B-N5-02). */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/contract/golden/children');
interface Golden {
  steps: { request: { method: string; url: string }; status: number; body: { data?: unknown } }[];
}
const step = (url: string): unknown =>
  (JSON.parse(readFileSync(resolve(GOLDEN_DIR, 'milestones_and_learn.fa.json'), 'utf8')) as Golden).steps.find(
    (s) => s.request.method === 'GET' && s.request.url === url,
  )?.body.data;

describe.skipIf(!existsSync(GOLDEN_DIR))('learn parser', () => {
  it('parses all topics with this week’s pick', () => {
    const v = learnViewSchema.parse(step('/api/v1/children/1/learn'));
    expect(v.ageMonths).toBe(3);
    expect(v.topics[0]).toEqual({ code: 'all', label: 'همه' });
    expect(v.topics).toHaveLength(6);
    expect(v.topic).toBeNull();
    expect(v.featured).toMatchObject({ code: 'night_sleep_3m', topic: 'sleep', minutes: 4, articleSlug: null });
    expect(v.tips.length).toBeGreaterThan(2);
    expect(v.disclaimer).toBeTruthy();
  });

  it('parses one topic without a pick', () => {
    const v = learnViewSchema.parse(step('/api/v1/children/1/learn?topic=feeding'));
    expect(v.topic).toBe('feeding');
    expect(v.featured).toBeNull();
    expect(v.tips.every((tip) => tip.topic === 'feeding')).toBe(true);
  });
});

describe('learn age', () => {
  it('reads months, then years', () => {
    expect(learnAge(3)).toEqual({ unit: 'months', n: 3 });
    expect(learnAge(30)).toEqual({ unit: 'years', n: 2 });
  });
});
