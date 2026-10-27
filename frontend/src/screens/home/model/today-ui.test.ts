import { describe, expect, it } from 'vitest';

import type { CycleTodayUi } from '@/entities/cycle';

import type { HeroInput } from './hero-state';
import { heroState } from './hero-state';
import { bannerActions, isEnginePayload, latenessTier, todayHero } from './today-ui';

const base: HeroInput = {
  isToday: true,
  inPeriod: false,
  daysLeft: 11,
  daysLate: 0,
  periodOpen: false,
  cycleDay: 18,
  periodLength: 5,
  startDismissed: false,
  endDismissed: false,
};

const ui = (over: Partial<CycleTodayUi> = {}): CycleTodayUi => ({
  ringAction: 'edit',
  banner: null,
  showPregnancyTestButton: false,
  predictionCard: null,
  ...over,
});

const hero = (input: Partial<HeroInput>, todayUi: CycleTodayUi | null) =>
  todayHero({ ...base, ...input, todayUi });

describe('latenessTier', () => {
  it('splits at 7/8 and 14/15', () => {
    expect([0, 1, 7, 8, 14, 15, 30].map(latenessTier)).toEqual([
      'none', 'calm', 'calm', 'overdue', 'overdue', 'ask', 'ask',
    ]);
    expect(latenessTier(null)).toBe('none');
  });
});

describe('todayHero with a v6 today_ui', () => {
  it('late 7 is calm (late variant, no banner); late 8 carries the overdue banner and test button', () => {
    const calm = hero({ daysLeft: null, daysLate: 7, cycleDay: 35 }, ui({ ringAction: 'start' }));
    expect(calm).toMatchObject({ variant: 'late', startPrompt: true, banner: null, tier: 'calm' });
    const overdue = hero(
      { daysLeft: null, daysLate: 8 },
      ui({ ringAction: 'start', banner: 'overdue', showPregnancyTestButton: true }),
    );
    expect(overdue).toMatchObject({ variant: 'late', banner: 'overdue', showPregnancyTest: true, tier: 'overdue' });
  });

  it('late 15 is the ask tier; the overdue banner hides the test button when the engine does not suggest it', () => {
    expect(hero({ daysLeft: null, daysLate: 15 }, ui({ banner: 'overdue' }))).toMatchObject({
      tier: 'ask',
      banner: 'overdue',
      showPregnancyTest: false,
    });
    expect(hero({ daysLeft: null, daysLate: 20 }, ui({ banner: 'ask_last_period' }))).toMatchObject({
      banner: 'ask_last_period',
      showPregnancyTest: false,
    });
  });

  it('open period: the end prompt follows ring_action (EPL−2 edit, EPL−1 end, hard cap end, cap+1 edit)', () => {
    const during = { inPeriod: true, periodOpen: true, daysLeft: 25 };
    expect(hero({ ...during, cycleDay: 3 }, ui({ ringAction: 'edit' }))).toMatchObject({ variant: 'during', endPrompt: false });
    expect(hero({ ...during, cycleDay: 4 }, ui({ ringAction: 'end' }))).toMatchObject({ variant: 'during', endPrompt: true });
    expect(hero({ ...during, cycleDay: 14 }, ui({ ringAction: 'end', banner: 'end_missing' }))).toMatchObject({
      endPrompt: true,
      endMissing: true,
      banner: 'end_missing',
    });
    expect(hero({ ...during, cycleDay: 15 }, ui({ ringAction: 'edit', banner: 'end_missing' }))).toMatchObject({
      endPrompt: false,
      banner: 'end_missing',
    });
  });

  it('start window: the prompt follows ring_action at next−4/−3/+14/+15', () => {
    expect(hero({ daysLeft: 4 }, ui({ ringAction: 'edit' })).startPrompt).toBe(false);
    expect(hero({ daysLeft: 3 }, ui({ ringAction: 'start' })).startPrompt).toBe(true);
    expect(hero({ daysLeft: null, daysLate: 14 }, ui({ ringAction: 'start' })).startPrompt).toBe(true);
    expect(hero({ daysLeft: null, daysLate: 15 }, ui({ ringAction: 'edit' })).startPrompt).toBe(false);
  });

  it('«هنوز نه» / «هنوز ادامه دارد» stay local; the end dismissal also silences end_missing', () => {
    expect(hero({ daysLeft: 1, startDismissed: true }, ui({ ringAction: 'start' })).startPrompt).toBe(false);
    const open = { inPeriod: true, periodOpen: true, cycleDay: 9 };
    const shown = ui({ ringAction: 'end', banner: 'end_missing' });
    expect(hero({ ...open, endDismissed: true }, shown)).toMatchObject({ endPrompt: false, banner: null, endMissing: false });
  });

  it('shows nothing on a tapped (non-today) day', () => {
    expect(hero({ isToday: false, daysLeft: 1 }, ui({ ringAction: 'start', banner: 'overdue' }))).toMatchObject({
      startPrompt: false,
      banner: null,
    });
  });

  it('one banner only, an unknown code degrades to none', () => {
    expect(hero({}, ui({ banner: 'something_new' })).banner).toBeNull();
  });

  it('unknown main phase stays neutral: no period, not in the period variant', () => {
    expect(hero({ daysLeft: null, daysLate: 0, cycleDay: null }, ui())).toMatchObject({ variant: 'normal', endPrompt: false });
  });
});

describe('todayHero without today_ui (Laravel payload)', () => {
  it('is identical to heroState, with no banner', () => {
    const cases: Partial<HeroInput>[] = [
      {},
      { daysLeft: 1 },
      { daysLeft: null, daysLate: 4 },
      { inPeriod: true, periodOpen: true, cycleDay: 6 },
      { inPeriod: true, periodOpen: true, cycleDay: 3, endDismissed: true },
    ];
    for (const c of cases) {
      const input = { ...base, ...c };
      const { banner, showPregnancyTest, fromEngine, tier, ...rest } = todayHero({ ...input, todayUi: null });
      expect(rest).toEqual(heroState(input));
      expect([banner, showPregnancyTest, fromEngine]).toEqual([null, false, false]);
      expect(tier).toBe(latenessTier(input.daysLate));
    }
  });
});

describe('isEnginePayload', () => {
  it('detects v6 by today_ui or limits', () => {
    expect(isEnginePayload(null)).toBe(false);
    expect(isEnginePayload({ todayUi: null, limits: null })).toBe(false);
    expect(isEnginePayload({ todayUi: ui(), limits: null })).toBe(true);
    expect(
      isEnginePayload({ todayUi: null, limits: { fertilitySuppressed: false, suppressionReason: null, engineSuspended: false, promptPregnancyMode: false } }),
    ).toBe(true);
  });
});

describe('bannerActions (which action each button triggers)', () => {
  it('end_missing opens the period end editor', () => {
    expect(bannerActions('end_missing', 'none', false)).toEqual(['edit_end']);
  });
  it('overdue offers the pregnancy test only when the engine says so; the long tier adds the date editor', () => {
    expect(bannerActions('overdue', 'overdue', true)).toEqual(['log_pregnancy_test']);
    expect(bannerActions('overdue', 'overdue', false)).toEqual([]);
    expect(bannerActions('overdue', 'ask', true)).toEqual(['log_pregnancy_test', 'edit_last_period']);
  });
  it('ask_last_period opens the PeriodDateEditor', () => {
    expect(bannerActions('ask_last_period', 'ask', false)).toEqual(['edit_last_period']);
  });
});
