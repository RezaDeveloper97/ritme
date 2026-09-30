import { createElement as h, type ReactElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import {
  Checkbox,
  CountdownRing,
  NumericScale,
  PrimaryButton,
  ProgressBar,
  RadioCardGroup,
  SearchField,
  SeverityScale,
  StepTimeline,
  UrgentCard,
  WeekDots,
  formatClock,
  severityTone,
  timerProgress,
} from './index';
import { barPercent } from './ProgressBar';
import { scaleValues } from './NumericScale';
import { radioStep } from './radio-group';

/*
 * CB-CORE-02 primitives: the a11y contract (role, name, state) on the
 * server-rendered markup, plus the pure helpers behind them.
 */
const html = (el: ReactElement) => renderToStaticMarkup(el);
const noop = () => undefined;

describe('radioStep — radiogroup keyboard (APG, RTL-aware)', () => {
  it('reading-forward arrow and ↓ go next, wrapping', () => {
    expect(radioStep('ArrowLeft', 0, 4, true)).toBe(1);
    expect(radioStep('ArrowRight', 0, 4, false)).toBe(1);
    expect(radioStep('ArrowDown', 3, 4, true)).toBe(0);
  });
  it('reading-backward arrow and ↑ go previous; Home/End jump', () => {
    expect(radioStep('ArrowRight', 0, 4, true)).toBe(3);
    expect(radioStep('ArrowUp', 2, 4, false)).toBe(1);
    expect(radioStep('Home', 2, 4, true)).toBe(0);
    expect(radioStep('End', 0, 4, true)).toBe(3);
    expect(radioStep('Enter', 0, 4, true)).toBeNull();
  });
});

describe('SeverityScale', () => {
  const options = [
    { value: 'none', label: 'ندارم' },
    { value: 'mild', label: 'خفیف' },
    { value: 'moderate', label: 'متوسط' },
    { value: 'severe', label: 'شدید' },
  ] as const;

  it('radiogroup named by the row title; one checked radio is the only tab stop', () => {
    const out = html(h(SeverityScale, { options, value: 'moderate', onChange: noop, label: 'گرگرفتگی' }));
    expect(out).toMatch(/<div id="([^"]+)" class="nb-scale-label">گرگرفتگی<\/div><div role="radiogroup" aria-labelledby="\1"/);
    expect(out.match(/role="radio"/g)).toHaveLength(4);
    expect(out.match(/aria-checked="true"/g)).toHaveLength(1);
    expect(out).toMatch(/aria-checked="true" tabindex="0" class="nb-sev-opt nb-tone-bloom">متوسط/);
    expect(out.match(/tabindex="0"/g)).toHaveLength(1);
  });

  it('nothing checked → first option is the tab stop; aria-label when no visible title', () => {
    const out = html(h(SeverityScale, { options, value: null, onChange: noop, ariaLabel: 'شدت' }));
    expect(out).toContain('role="radiogroup" aria-label="شدت"');
    expect(out).toMatch(/tabindex="0"[^>]*>ندارم/);
  });

  it('tone ramp: none-first vs the 4-level hot-flash scale', () => {
    expect([0, 1, 2, 3].map((i) => severityTone(i))).toEqual(['neutral', 'data', 'bloom', 'danger']);
    expect([0, 1, 2, 3].map((i) => severityTone(i, false))).toEqual(['data', 'warm', 'bloom', 'danger']);
  });
});

describe('NumericScale', () => {
  it('scaleValues is inclusive', () => {
    expect(scaleValues(0, 10)).toHaveLength(11);
    expect(scaleValues(1, 6)).toEqual([1, 2, 3, 4, 5, 6]);
  });

  it('0–10: radiogroup of 11 radios in locale digits, described by the end labels', () => {
    const out = html(
      h(NumericScale, {
        min: 0,
        max: 10,
        value: 7,
        onChange: noop,
        locale: 'fa',
        label: 'چقدر شدید؟',
        minLabel: 'بدون درد',
        maxLabel: 'بدترین درد ممکن',
        variant: 'solid',
        tone: 'danger',
      }),
    );
    expect(out.match(/role="radio"/g)).toHaveLength(11);
    expect(out).toMatch(/aria-describedby="([^"]+)"[\s\S]*<div id="\1" class="nb-nscale-ends"><span>بدون درد<\/span>/);
    expect(out).toMatch(/aria-checked="true" tabindex="0" class="nb-nscale-cell"><span class="nb-nscale-num">۷<\/span>/);
    expect(out).toContain('nb-nscale is-solid nb-tone-danger is-dense');
  });

  it('1–6 without end labels has no describedby', () => {
    const out = html(h(NumericScale, { min: 1, max: 6, value: null, onChange: noop, locale: 'en', ariaLabel: 'Mood' }));
    expect(out.match(/role="radio"/g)).toHaveLength(6);
    expect(out).not.toContain('aria-describedby');
    expect(out).not.toContain('is-dense');
  });
});

describe('StepTimeline', () => {
  const steps = [
    { id: 'a', title: 'آماده‌سازی', state: 'done' as const },
    { id: 'b', title: 'تحریک', meta: 'حدود ۱۰ روز', state: 'current' as const },
    { id: 'c', title: 'تخمک‌کشی', state: 'todo' as const },
  ];
  const stateLabels = { done: 'انجام شد', current: 'مرحله فعلی', todo: 'بعدی' };

  it('named ordered list; current step has aria-current=step; each step speaks its state', () => {
    const out = html(h(StepTimeline, { steps, label: 'مراحل درمان', stateLabels, locale: 'fa' }));
    expect(out).toMatch(/^<ol aria-label="مراحل درمان" class="nb-tl is-number">/);
    expect(out.match(/aria-current="step"/g)).toHaveLength(1);
    expect(out).toMatch(/aria-current="step" class="nb-tl-step is-current nb-tone-brand"/);
    expect(out).toContain('<span class="sr-only"> — مرحله فعلی</span>');
    // numbered markers in locale digits; done shows a check instead
    expect(out).toMatch(/nb-tl-marker" aria-hidden="true">۲</);
    expect(out).toMatch(/nb-tl-marker" aria-hidden="true"><svg/);
  });

  it('dot variant; done steps are data-toned, current can be bloom («waiting on you»)', () => {
    const out = html(
      h(StepTimeline, {
        steps: [steps[0], { ...steps[1], tone: 'bloom' as const }, steps[2]],
        label: 'روند',
        stateLabels,
        marker: 'dot',
        locale: 'fa',
      }),
    );
    expect(out).toContain('nb-tl is-dot');
    expect(out).toContain('nb-tl-step is-done nb-tone-data');
    expect(out).toContain('nb-tl-step is-current nb-tone-bloom');
  });
});

describe('CountdownRing (on ProgressRing)', () => {
  it('formatClock / timerProgress', () => {
    expect(formatClock(102_000)).toBe('01:42');
    expect(formatClock(3_725_000)).toBe('1:02:05');
    expect(formatClock(-5)).toBe('00:00');
    expect(timerProgress(2_000, 5_000, 'countdown')).toBeCloseTo(0.6);
    expect(timerProgress(2_000, 5_000, 'elapsed')).toBeCloseTo(0.4);
    expect(timerProgress(90_000, undefined, 'elapsed')).toBeCloseTo(0.5);
    expect(timerProgress(9_000, 5_000, 'countdown')).toBe(0);
  });

  it('elapsed: progressbar named, value text = locale clock, not a live region', () => {
    const out = html(h(CountdownRing, { elapsedMs: 102_000, label: 'زمان گرگرفتگی', locale: 'fa', caption: 'الان دارم', glow: true, tone: 'bloom' }));
    expect(out).toMatch(/role="progressbar" aria-label="زمان گرگرفتگی"[^>]*aria-valuetext="۰۱:۴۲"/);
    expect(out).toContain('nb-ring nb-tone-bloom nb-countdown is-glow');
    expect(out).toContain('<span class="nb-countdown-value">۰۱:۴۲</span>');
    expect(out).not.toContain('aria-live');
  });

  it('countdown: shows the time left, rounded up', () => {
    const out = html(h(CountdownRing, { elapsedMs: 1_200, totalMs: 5_000, mode: 'countdown', label: 'نگه دار', locale: 'en' }));
    expect(out).toContain('aria-valuetext="00:04"');
    expect(out).toContain('aria-valuenow="76"');
  });
});

describe('WeekDots', () => {
  it('named list of 7 days, each spoken as «weekday: state»', () => {
    const out = html(
      h(WeekDots, {
        days: ['done', 'done', 'done', 'done', 'missed', 'done', 'future'],
        locale: 'fa',
        label: 'این هفته',
        stateLabels: { done: 'مصرف شد', missed: 'ثبت نشده', future: 'هنوز نرسیده' },
        todayIndex: 5,
      }),
    );
    expect(out).toMatch(/^<ul aria-label="این هفته" class="nb-wdots nb-tone-data">/);
    expect(out.match(/<li /g)).toHaveLength(7);
    expect(out).toContain('aria-label="ش: مصرف شد"');
    expect(out).toContain('nb-wdot is-done is-today');
    expect(out).toMatch(/is-missed" aria-label="[^:]+: ثبت نشده"/);
  });
});

describe('ProgressBar', () => {
  it('barPercent clamps and guards max=0', () => {
    expect(barPercent(6, 20)).toBe(30);
    expect(barPercent(40, 20)).toBe(100);
    expect(barPercent(3, 0)).toBe(0);
  });

  it('progressbar labelled by the row title with min/max/now; fill width is the only inline datum', () => {
    const out = html(h(ProgressBar, { value: 6, max: 20, label: 'پاراکلینیک', valueLabel: '۱۴ از ۲۰ میلیون مانده', tone: 'brand' }));
    expect(out).toMatch(/<span id="([^"]+)" class="nb-pbar-label">پاراکلینیک<\/span>[\s\S]*role="progressbar" aria-labelledby="\1" aria-valuemin="0" aria-valuemax="20" aria-valuenow="6"/);
    expect(out).toContain('style="inline-size:30%"');
  });

  it('valueText replaces the visible value for screen readers', () => {
    const out = html(h(ProgressBar, { value: 90, max: 150, label: 'پیاده‌روی', valueLabel: '۹۰ از ۱۵۰', valueText: '۹۰ دقیقه از ۱۵۰' }));
    expect(out).toContain('class="nb-pbar-value" aria-hidden="true"');
    expect(out).toContain('aria-valuetext="۹۰ دقیقه از ۱۵۰"');
  });
});

describe('RadioCardGroup', () => {
  it('radiogroup of cards; title labels, description describes; disabled skipped by roving', () => {
    const out = html(
      h(RadioCardGroup, {
        label: 'مرحله',
        value: 'meno',
        onChange: noop,
        options: [
          { value: 'peri', title: 'پیش‌یائسگی', description: 'هنوز پریود دارم', icon: 'clock', iconTone: 'bloom' },
          { value: 'meno', title: 'یائسگی', description: '۱۲ ماه', icon: 'moon' },
          { value: 'x', title: 'غیرفعال', disabled: true },
        ],
      }),
    );
    expect(out).toMatch(/^<div role="radiogroup" aria-label="مرحله" class="nb-rcards">/);
    expect(out.match(/role="radio"/g)).toHaveLength(3);
    expect(out).toMatch(/role="radio" aria-checked="true" aria-labelledby="([^"]+)" aria-describedby="[^"]+" tabindex="0"[\s\S]*id="\1" class="nb-rcard-title">یائسگی/);
    expect(out).toMatch(/disabled="" tabindex="-1"/);
  });
});

describe('SearchField', () => {
  it('search landmark with a labelled type=search input; clear button only with text', () => {
    const empty = html(h(SearchField, { value: '', onValueChange: noop, label: 'جستجو', clearLabel: 'پاک کردن', placeholder: 'جستجو در ریتمی' }));
    expect(empty).toMatch(/^<div role="search" class="nb-search">/);
    expect(empty).toMatch(/<input type="search" aria-label="جستجو" enterKeyHint="search"/i);
    expect(empty).not.toContain('پاک کردن');
    const filled = html(h(SearchField, { value: 'قرص', onValueChange: noop, label: 'جستجو', clearLabel: 'پاک کردن' }));
    expect(filled).toMatch(/<button type="button" class="nb-search-clear" aria-label="پاک کردن">/);
  });
});

describe('Checkbox', () => {
  it('native checkbox inside its label (row is the hit target)', () => {
    const out = html(h(Checkbox, { checked: true, onCheckedChange: noop, label: 'شیشه شیر', description: '۲ عدد' }));
    expect(out).toMatch(/^<label class="nb-check"><input type="checkbox" class="nb-check-input" checked=""/);
    expect(out).toContain('<span class="nb-check-label">شیشه شیر</span>');
  });
});

describe('UrgentCard — DangerNote extension', () => {
  const hotlines = [
    { label: 'صدای مشاور', number: '1480', display: '۱۴۸۰' },
    { label: 'اورژانس اجتماعی', number: '123', display: '۱۲۳' },
  ];

  it('card (default) stays role=alert and keeps the B-N1-03 markup', () => {
    const out = html(h(UrgentCard, { title: 'تماس با ۱۱۵', action: h(PrimaryButton, { children: 'تماس' }) }));
    expect(out).toMatch(/^<div role="alert" class="nb-urgent">/);
    expect(out).toContain('nb-urgent-action');
  });

  it('note variant: role=note, no CTA, tel: hotline links in a named list', () => {
    const out = html(h(UrgentCard, { variant: 'note', title: 'اگر به آسیب زدن به خودت فکر می‌کنی…', hotlines, hotlinesLabel: 'شماره‌های کمک' }));
    expect(out).toMatch(/^<div role="note" class="nb-urgent is-note">/);
    expect(out).toContain('<ul class="nb-urgent-hotlines" aria-label="شماره‌های کمک">');
    expect(out).toContain('href="tel:1480"');
    expect(out).toContain('href="tel:123"');
    expect(out).not.toContain('nb-urgent-action"');
  });

  it('urgent flag overrides the role; extra actions render in their own row', () => {
    const out = html(h(UrgentCard, { variant: 'note', urgent: true, title: 'OHSS', actions: h('a', { href: '#' }, 'x') }));
    expect(out).toMatch(/^<div role="alert"/);
    expect(out).toContain('nb-urgent-actions');
  });
});
