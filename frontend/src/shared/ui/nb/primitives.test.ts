import { createElement as h, type ReactElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import {
  Accordion,
  Avatar,
  BarChart,
  Card,
  ChipGroup,
  DateStrip,
  EmptyState,
  HeaderButton,
  HeroCard,
  HubHeader,
  IconCircle,
  InfoNote,
  LineChart,
  ListGroup,
  ListRow,
  NumberStepper,
  PillChip,
  PlusLock,
  PrimaryButton,
  ProgressRing,
  ProgressSteps,
  ScreenHeader,
  SecondaryButton,
  SectionTitle,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  Switch,
  TileButton,
  UrgentCard,
} from './index';

/*
 * The a11y contract of each Night & Bloom primitive (role, accessible name,
 * state attribute), checked on the server-rendered markup — the same HTML the
 * first paint ships. Interaction is covered by the screens that use them.
 */
const html = (el: ReactElement) => renderToStaticMarkup(el);
const noop = () => undefined;

describe('Night & Bloom primitives — a11y contract', () => {
  it('ScreenHeader: h1 title, named 44px back button, spacer when no action', () => {
    const out = html(h(ScreenHeader, { title: 'تنظیمات سیکل', subtitle: 'پیش‌بینی', onBack: noop, backLabel: 'بازگشت' }));
    expect(out).toMatch(/<header class="nb-hdr"/);
    expect(out).toContain('<h1 class="nb-hdr-title">تنظیمات سیکل</h1>');
    expect(out).toMatch(/<button type="button" class="nb-hbtn" aria-label="بازگشت">/);
    expect(out).toContain('class="nb-hdr-spacer" aria-hidden="true"');
  });

  it('HeaderButton: icon-only button carries its label; icon is hidden', () => {
    const out = html(h(HeaderButton, { label: 'اعلان‌ها', icon: 'bell', variant: 'soft', badge: true }));
    expect(out).toMatch(/<button[^>]*aria-label="اعلان‌ها"/);
    expect(out).toContain('aria-hidden="true"');
  });

  it('HubHeader: greeting is the h1', () => {
    const out = html(h(HubHeader, { date: 'شنبه ۱۲ مهر', greeting: 'سلام مریم' }));
    expect(out).toContain('<h1 class="nb-hub-greeting">سلام مریم</h1>');
  });

  it('Card / HeroCard / SectionTitle: section semantics and h2 heading', () => {
    expect(html(h(Card, { as: 'section', 'aria-label': 'x', children: 'c' }))).toMatch(/^<section class="nb-card pad-md" aria-label="x">/);
    expect(html(h(HeroCard, { children: 'c' }))).toContain('nb-card is-hero');
    const out = html(h(SectionTitle, { title: 'امروز', actionLabel: 'همه', onAction: noop }));
    expect(out).toContain('<h2 class="nb-sect-title">امروز</h2>');
    expect(out).toMatch(/<button type="button" class="nb-sect-link">همه<\/button>/);
  });

  it('PrimaryButton / SecondaryButton: native buttons; loading = disabled + aria-busy', () => {
    const out = html(h(PrimaryButton, { loading: true, children: 'ادامه' }));
    expect(out).toMatch(/^<button type="button"[^>]*disabled=""[^>]*aria-busy="true"/);
    expect(out).toContain('ادامه');
    expect(html(h(SecondaryButton, { variant: 'text', children: 'هنوز نه' }))).toContain('nb-btn is-text');
  });

  it('TileButton: toggle tiles expose aria-pressed', () => {
    expect(html(h(TileButton, { icon: 'drop', label: 'پریود', pressed: true }))).toContain('aria-pressed="true"');
    expect(html(h(TileButton, { icon: 'drop', label: 'پریود' }))).not.toContain('aria-pressed');
  });

  it('PillChip / ChipGroup: aria-pressed chips inside a labelled group', () => {
    const out = html(
      h(ChipGroup, { label: 'محل درد', children: h(PillChip, { pressed: true, mode: 'multi', tone: 'bloom', children: 'شکم' }) }),
    );
    expect(out).toMatch(/<div role="group" aria-label="محل درد"/);
    expect(out).toMatch(/<button type="button" aria-pressed="true" class="nb-chip is-multi nb-tone-bloom"/);
  });

  it('SegmentedTabs: tablist with one selected, focusable tab', () => {
    const out = html(
      h(SegmentedTabs, {
        label: 'نما',
        value: 'month',
        onChange: noop,
        tabs: [
          { value: 'month', label: 'ماه' },
          { value: 'year', label: 'سال' },
        ],
      }),
    );
    expect(out).toMatch(/role="tablist" aria-label="نما"/);
    expect(out.match(/role="tab"/g)).toHaveLength(2);
    expect(out).toMatch(/aria-selected="true" tabindex="0"[^>]*>ماه/);
    expect(out).toMatch(/aria-selected="false" tabindex="-1"[^>]*>سال/);
  });

  it('NumberStepper: labelled group, named −/+ (disabled at bounds), live value in locale digits', () => {
    const out = html(
      h(NumberStepper, {
        value: 3,
        min: 3,
        max: 10,
        onChange: noop,
        label: 'طول پریود',
        unit: 'روز',
        decrementLabel: 'کم کردن',
        incrementLabel: 'زیاد کردن',
        locale: 'fa',
      }),
    );
    expect(out).toMatch(/role="group" aria-labelledby="([^"]+)"/);
    const id = /aria-labelledby="([^"]+)"/.exec(out)?.[1];
    expect(out).toContain(`id="${id}"`);
    expect(out).toMatch(/aria-label="کم کردن" disabled=""/);
    expect(out).toMatch(/aria-label="زیاد کردن"(?! disabled)/);
    expect(out).toMatch(/<output[^>]*aria-live="polite"/);
    expect(out).toContain('۳');
  });

  it('Switch: role=switch with aria-checked and a name', () => {
    expect(html(h(Switch, { checked: true, onCheckedChange: noop, label: 'محاسبه خودکار' }))).toMatch(
      /role="switch" aria-checked="true" aria-label="محاسبه خودکار"/,
    );
    const byRow = html(h(Switch, { checked: false, onCheckedChange: noop, labelledBy: 'r-title' }));
    expect(byRow).toContain('aria-labelledby="r-title"');
    expect(byRow).not.toContain('aria-label=');
  });

  it('StatusPill / PlusLock: teaser hidden + inert, lock explained by name', () => {
    expect(html(h(StatusPill, { tone: 'data', children: 'طبیعی' }))).toContain('طبیعی');
    const out = html(h(PlusLock, { label: 'پلاس', lockedText: 'در پلاس باز می‌شود', onUnlock: noop, children: 'راز' }));
    expect(out).toMatch(/class="nb-lock-body" aria-hidden="true" inert=""/);
    expect(out).toMatch(/<button type="button" class="nb-lock-hit" aria-label="در پلاس باز می‌شود">/);
    expect(html(h(PlusLock, { label: 'پلاس', lockedText: 'قفل', children: 'x' }))).toContain('<span class="sr-only">قفل</span>');
  });

  it('ListRow / ListGroup: action rows are buttons, static rows are not; group has a heading', () => {
    expect(html(h(ListRow, { title: 'زبان', value: 'فارسی', onClick: noop }))).toMatch(/^<button type="button" class="nb-row is-action">/);
    expect(html(h(ListRow, { title: 'زبان', id: 'lang' }))).toMatch(/^<div class="nb-row"><span class="nb-row-text"><span id="lang-title"/);
    expect(html(h(ListGroup, { title: 'یادآورها', children: 'x' }))).toContain('<h2 class="nb-list-title">یادآورها</h2>');
  });

  it('Accordion: button with aria-expanded controls a labelled region', () => {
    const closed = html(h(Accordion, { title: 'درد', summary: 'ثبت نشده', children: 'body' }));
    expect(closed).toMatch(/aria-expanded="false" aria-controls="([^"]+)"/);
    const panel = /aria-controls="([^"]+)"/.exec(closed)?.[1];
    expect(closed).toMatch(new RegExp(`id="${panel}" role="region" aria-labelledby="[^"]+" class="nb-acc-panel" hidden=""`));
    expect(html(h(Accordion, { title: 'درد', defaultOpen: true, children: 'b' }))).toContain('aria-expanded="true"');
  });

  it('IconCircle / Avatar: decorative unless labelled; avatar is a named img', () => {
    expect(html(h(IconCircle, { icon: 'drop' }))).toContain('aria-hidden="true"');
    expect(html(h(IconCircle, { icon: 'drop', label: 'پریود' }))).toMatch(/role="img" aria-label="پریود"/);
    expect(html(h(Avatar, { name: 'مریم' }))).toMatch(/role="img" aria-label="مریم"[^>]*><span aria-hidden="true">م</);
  });

  it('InfoNote / UrgentCard / EmptyState: note, alert, headed section', () => {
    expect(html(h(InfoNote, { source: 'WHO', children: 'n' }))).toMatch(/^<aside role="note"/);
    expect(html(h(UrgentCard, { title: 'تماس با ۱۱۵' }))).toMatch(/^<div role="alert"/);
    expect(html(h(EmptyState, { icon: 'note', title: 'هنوز چیزی نیست' }))).toContain('<h2 class="nb-empty-title">هنوز چیزی نیست</h2>');
  });

  it('Skeleton / SkeletonGroup: shapes hidden, group is a busy status with a label', () => {
    expect(html(h(Skeleton, { shape: 'card' }))).toMatch(/^<span aria-hidden="true"/);
    expect(html(h(SkeletonGroup, { label: 'در حال بارگذاری', children: 'x' }))).toMatch(
      /role="status" aria-busy="true" aria-live="polite"[^>]*><span class="sr-only">در حال بارگذاری<\/span>/,
    );
  });

  it('ProgressRing / ProgressSteps: progressbar with name and values', () => {
    const ring = html(h(ProgressRing, { value: 0.5, label: 'سیکل', valueText: 'روز ۱۴ از ۲۹' }));
    expect(ring).toMatch(/role="progressbar" aria-label="سیکل" aria-valuemin="0" aria-valuemax="100" aria-valuenow="50" aria-valuetext="روز ۱۴ از ۲۹"/);
    const steps = html(h(ProgressSteps, { total: 8, current: 3, label: 'مرحله ۳ از ۸' }));
    expect(steps).toMatch(/role="progressbar" aria-label="مرحله ۳ از ۸" aria-valuemin="0" aria-valuemax="8" aria-valuenow="3"/);
    expect(steps.match(/is-done/g)).toHaveLength(3);
  });

  it('DateStrip: labelled group of 7 named days; selected pressed, today = aria-current', () => {
    const day = new Date(2026, 8, 30);
    const out = html(h(DateStrip, { locale: 'fa', selected: day, today: day, onSelect: noop, label: 'این هفته', marker: () => 'period' as const, markedLabel: 'ثبت شده' }));
    expect(out).toMatch(/^<div role="group" aria-label="این هفته"/);
    expect(out.match(/<button/g)).toHaveLength(7);
    expect(out).toMatch(/aria-pressed="true" aria-current="date" aria-label="[^"]+ · ثبت شده"/);
  });

  it('LineChart / BarChart: named img, ltr plot', () => {
    const line = html(h(LineChart, { label: 'دمای پایه', series: [{ values: [36.4, null, 36.7] }], highlightIndex: 2 }));
    expect(line).toMatch(/^<svg role="img" aria-label="دمای پایه"/);
    expect(line).toContain('nb-chart-today');
    const bars = html(h(BarChart, { label: 'طول سیکل‌ها', bars: [{ label: '۱', value: 28 }] }));
    expect(bars).toMatch(/^<svg role="img" aria-label="طول سیکل‌ها"/);
  });

  it('SkyLayer: purely decorative', () => {
    expect(html(h(SkyLayer))).toBe('<div aria-hidden="true" class="nb-sky"></div>');
  });
});
