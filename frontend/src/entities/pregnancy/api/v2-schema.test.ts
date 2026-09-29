import { describe, expect, it } from 'vitest';

import { pregnancyKeys } from './keys';
import {
  datingPreviewSchema,
  pregnancyAlertsV2Schema,
  pregnancyCalendarSchema,
  pregnancyDaySchema,
  pregnancyReportSchema,
  pregnancyTodaySchema,
  pregnancyWeekSchema,
  weekStateSchema,
} from './v2-schema';

/*
 * Boundary contract for `/api/v1/pregnancy/v2/*` — fixtures follow
 * docs/pregnancy-v2/README.md and the artboards (week 8 + 3 days, due
 * 1406-02-11 = 2027-05-01). The frontend is written before the Go endpoints
 * (T-M7-02…05), so the parsers must also survive values this bundle has never
 * seen and the shapes the README leaves open.
 */

describe('datingPreviewSchema', () => {
  const preview = {
    source: 'lmp',
    weeks: 8,
    days: 3,
    trimester: 1,
    due_date: '2027-05-01',
    due_date_label: '۱۱ اردیبهشت ۱۴۰۶',
    range: { from: '2027-04-17', to: '2027-05-15' },
    uncertainty_days: 14,
    confidence: { level: 'medium', label: 'متوسط' },
    basis: 'مبنای این محاسبه اولین روز آخرین قاعدگیه (۳ مرداد).',
    range_label: 'بازهٔ معمول تولد: ۲۸ فروردین تا ۲۵ اردیبهشت.',
    copy: { lead: 'بر اساس داده‌های فعلی، احتمالاً', suffix: 'باردار هستی', due_label: null, confidence: 'دقت تخمین: {confidence}' },
  };

  it('maps the README shape', () => {
    expect(datingPreviewSchema.parse(preview)).toEqual({
      source: 'lmp',
      age: { weeks: 8, days: 3 },
      trimester: 1,
      dueDate: '2027-05-01',
      dueDateLabel: '۱۱ اردیبهشت ۱۴۰۶',
      range: { from: '2027-04-17', to: '2027-05-15' },
      rangeLabel: 'بازهٔ معمول تولد: ۲۸ فروردین تا ۲۵ اردیبهشت.',
      uncertaintyDays: 14,
      confidence: { level: 'medium', label: 'متوسط' },
      basis: 'مبنای این محاسبه اولین روز آخرین قاعدگیه (۳ مرداد).',
      copy: {
        lead: 'بر اساس داده‌های فعلی، احتمالاً',
        suffix: 'باردار هستی',
        dueLabel: null,
        confidence: 'دقت تخمین: {confidence}',
        primary: null,
        secondary: null,
      },
    });
  });

  it('accepts a nested age, a bare confidence and a timestamped due date', () => {
    const parsed = datingPreviewSchema.parse({
      source: 'ultrasound',
      gestational_age: { weeks: '7', days: '4' },
      due_date: '2027-05-01T00:00:00Z',
      confidence: 'high',
      basis_sentence: 'x',
    });
    expect(parsed.age).toEqual({ weeks: 7, days: 4 });
    expect(parsed.dueDate).toBe('2027-05-01');
    expect(parsed.confidence).toEqual({ level: 'high', label: null });
    expect(parsed.basis).toBe('x');
    expect(parsed.range).toBeNull();
  });

  it('nulls unknown enums and rejects a preview without weeks or due date', () => {
    const parsed = datingPreviewSchema.parse({ ...preview, source: 'ivf', confidence: { level: 'certain' } });
    expect(parsed.source).toBeNull();
    expect(parsed.confidence.level).toBeNull();
    expect(() => datingPreviewSchema.parse({ weeks: 8 })).toThrow();
    expect(() => datingPreviewSchema.parse({ due_date: '2027-05-01' })).toThrow();
  });
});

describe('pregnancyTodaySchema', () => {
  const today = {
    date: '2026-09-22',
    weeks: 8,
    days: 3,
    trimester: 1,
    confidence: { level: 'medium', label: 'متوسط' },
    uncertainty_days: 5,
    carousel: [
      { week: 7, trimester: 1, title: '۷ هفته', size_line: 'بلوبری', illustration_key: 'blueberry', relation: 'past' },
      { week: 8, trimester: 1, title: '۸ هفته و ۳ روز', size_line: 'تمشک', illustration_key: 'raspberry', relation: 'current' },
      { week: 9, trimester: 1, title: '۹ هفته', size_line: 'گیلاس', illustration_key: 'cherry', relation: 'future' },
    ],
    due: { date: '2027-05-01', date_label: '۱۱ اردیبهشت ۱۴۰۶', days_left: 221, range: { from: '2027-04-17', to: '2027-05-15' } },
    progress: {
      week: 8,
      percent: 21,
      trimesters: [
        { trimester: 2, start_date: '2026-10-31', start_label: '۹ آبان', percent: 0 },
        { trimester: 1, start_date: '2026-07-25', percent: 62 },
        { trimester: 3, start_date: '2027-02-06', percent: 0 },
      ],
    },
    next_visit: {
      appointment_id: 12,
      care_item_key: 'nt_scan',
      title: 'سونوگرافی NT',
      date: '2026-10-10',
      time: '10:30:00',
      week: 11,
      days_until: 18,
      stage: 'booked',
    },
    tip: { week: 8, title: 'ضربان قلب', body: 'متن', read_minutes: 3, link: '/pregnancy/weeks/8' },
    tasks: [
      { key: 'folic', text: 'مصرف روزانهٔ فولیک اسید', done: true },
      { key: 'book_nt', text: 'رزرو نوبت سونوگرافی NT', done: false },
    ],
    unread_alerts: 2,
  };

  it('maps the Main artboard payload', () => {
    const parsed = pregnancyTodaySchema.parse(today);
    expect(parsed.age).toEqual({ weeks: 8, days: 3 });
    expect(parsed.carousel.map((w) => [w.week, w.relation, w.illustrationKey])).toEqual([
      [7, 'past', 'blueberry'],
      [8, 'current', 'raspberry'],
      [9, 'future', 'cherry'],
    ]);
    expect(parsed.due).toEqual({
      date: '2027-05-01',
      dateLabel: '۱۱ اردیبهشت ۱۴۰۶',
      daysLeft: 221,
      range: { from: '2027-04-17', to: '2027-05-15' },
    });
    // Trimesters come back ordered 1 → 3.
    expect(parsed.progress.trimesters.map((t) => t.trimester)).toEqual([1, 2, 3]);
    expect(parsed.progress.trimesters[1]).toEqual({
      trimester: 2,
      startDate: '2026-10-31',
      startLabel: '۹ آبان',
      percent: 0,
    });
    expect(parsed.nextVisit).toMatchObject({ appointmentId: 12, time: '10:30', stage: 'booked', daysUntil: 18 });
    expect(parsed.tip).toEqual({ week: 8, title: 'ضربان قلب', body: 'متن', readMinutes: 3, link: '/pregnancy/weeks/8' });
    expect(parsed.tasks).toEqual([
      { key: 'folic', text: 'مصرف روزانهٔ فولیک اسید', done: true },
      { key: 'book_nt', text: 'رزرو نوبت سونوگرافی NT', done: false },
    ]);
    expect(parsed.unreadAlerts).toBe(2);
  });

  it('survives a minimal payload (new pregnancy, nothing booked, no tip)', () => {
    const parsed = pregnancyTodaySchema.parse({ date: '2026-09-22', gestational_age: { weeks: 4, days: 0 } });
    expect(parsed).toMatchObject({
      carousel: [],
      due: null,
      nextVisit: null,
      tip: null,
      tasks: [],
      unreadAlerts: 0,
      confidence: { level: null, label: null },
    });
    // Progress falls back to the age and 4/40 = 10 %.
    expect(parsed.progress).toEqual({ week: 4, percent: 10, trimesters: [] });
  });

  it('skips broken rows instead of failing the screen', () => {
    const parsed = pregnancyTodaySchema.parse({
      ...today,
      carousel: [{ title: 'no week' }, today.carousel[1]],
      tasks: [{ key: 'x' }, { text: 'no key' }, today.tasks[0]],
      next_visit: { date: '2026-10-10' },
      tip: { read_minutes: 3 },
      progress: { percent: 180, trimesters: [{ trimester: 4 }] },
      unread_alerts: -3,
    });
    expect(parsed.carousel).toHaveLength(1);
    expect(parsed.tasks.map((t) => t.key)).toEqual(['folic']);
    expect(parsed.nextVisit).toBeNull();
    expect(parsed.tip).toBeNull();
    expect(parsed.progress.percent).toBe(100);
    expect(parsed.progress.trimesters).toEqual([]);
    expect(parsed.unreadAlerts).toBe(0);
  });
});

describe('pregnancyWeekSchema', () => {
  const week = {
    week: 8,
    trimester: 1,
    relation: 'current',
    range: { from: '2026-09-19', to: '2026-09-25' },
    range_label: '۲۸ شهریور تا ۳ مهر',
    bookmarked: false,
    details: {
      size_label: 'تمشک',
      illustration_key: 'raspberry',
      length_cm: '~۱.۶',
      weight_g: 1,
      heart_rate: '۱۵۰ تا ۱۷۰',
      headline: 'انگشت‌ها دارن شکل می‌گیرن',
      highlights: [
        { icon: 'hand', title: 'دست و پا', body: 'انگشت‌ها…' },
        { icon: 'heart', title: 'قلب', body: 'قلب تند می‌زنه…' },
        { icon: 'eye', tone: 'brand', title: 'صورت', body: 'پلک‌ها…' },
      ],
      body_symptoms: ['تهوع صبحگاهی', 'خستگی', ''],
      body_text: 'این علائم از تغییرات هورمونی میان…',
      tasks: [
        { key: 'folic', text: 'فولیک اسید رو هر روز ادامه بده' },
        { key: 'small_meals', text: 'چند وعدهٔ کوچک' },
      ],
      warning: 'این موارد رو با پزشکت در میون بذار…',
      reviewer_name: '[نام متخصص زنان و زایمان]',
      reviewed_at: '2026-09-01T10:00:00Z',
      sources: [{ title: 'WHO', url: 'https://www.who.int' }, 'ACOG'],
    },
    done_task_keys: ['folic'],
    tip: null,
  };

  it('maps the Week artboard payload and merges done_task_keys into tasks', () => {
    const parsed = pregnancyWeekSchema.parse(week);
    expect(parsed).toMatchObject({
      week: 8,
      relation: 'current',
      range: { from: '2026-09-19', to: '2026-09-25' },
      rangeLabel: '۲۸ شهریور تا ۳ مهر',
      bookmarked: false,
      tip: null,
    });
    expect(parsed.tasks).toEqual([
      { key: 'folic', text: 'فولیک اسید رو هر روز ادامه بده', done: true },
      { key: 'small_meals', text: 'چند وعدهٔ کوچک', done: false },
    ]);
    expect(parsed.details).toMatchObject({
      sizeLabel: 'تمشک',
      illustrationKey: 'raspberry',
      length: '~۱.۶',
      weight: '1',
      heartRate: '۱۵۰ تا ۱۷۰',
      bodySymptoms: ['تهوع صبحگاهی', 'خستگی'],
      reviewedAt: '2026-09-01',
      sources: [
        { title: 'WHO', url: 'https://www.who.int' },
        { title: 'ACOG', url: null },
      ],
    });
  });

  it('gives each highlight icon its artboard tint unless the admin overrides it', () => {
    const tones = pregnancyWeekSchema.parse(week).details.highlights.map((h) => [h.icon, h.tone]);
    expect(tones).toEqual([
      ['hand', 'brand'],
      ['heart', 'pink'],
      ['eye', 'brand'],
    ]);
  });

  it('keeps a highlight with an unknown icon (null icon) and drops empty ones', () => {
    const parsed = pregnancyWeekSchema.parse({
      ...week,
      details: { ...week.details, highlights: [{ icon: 'unicorn', title: 't', body: 'b' }, { icon: 'hand' }] },
    });
    expect(parsed.details.highlights).toEqual([{ icon: null, tone: 'brand', title: 't', body: 'b' }]);
  });

  it('reads a flat payload (details at the top level) and an unseeded week', () => {
    const flat = pregnancyWeekSchema.parse({ week_number: 30, size_label: 'کلم', tasks: [] });
    expect(flat.week).toBe(30);
    expect(flat.details.sizeLabel).toBe('کلم');
    expect(flat.relation).toBe('future');
    const empty = pregnancyWeekSchema.parse({ week: 41 });
    expect(empty.details.highlights).toEqual([]);
    expect(empty.tasks).toEqual([]);
  });
});

describe('weekStateSchema', () => {
  it('maps the PUT response', () => {
    expect(weekStateSchema.parse({ week: 8, bookmarked: true, done_task_keys: ['folic', 7] })).toEqual({
      week: 8,
      bookmarked: true,
      doneTaskKeys: ['folic'],
    });
  });
});

describe('pregnancyDaySchema', () => {
  const day = {
    date: '2026-09-22',
    week: 8,
    mood: 4,
    symptoms: { nausea: 'mild', fatigue: 'moderate', spotting: 'mild' },
    water_glasses: 5,
    weight: '62.4',
    visit_note: 'از دیروز سردرد خفیف دارم',
    last_weight: { value: 62.1, date: '2026-09-16' },
    alerts: [],
  };

  it('maps the Log artboard day', () => {
    expect(pregnancyDaySchema.parse(day)).toEqual({
      date: '2026-09-22',
      week: 8,
      mood: 4,
      symptoms: { nausea: 'mild', fatigue: 'moderate', spotting: 'mild' },
      waterGlasses: 5,
      weight: 62.4,
      visitNote: 'از دیروز سردرد خفیف دارم',
      lastWeight: { value: 62.1, date: '2026-09-16' },
      alerts: [],
    });
  });

  it('drops unknown symptoms / severities, clamps water and rejects odd moods', () => {
    const parsed = pregnancyDaySchema.parse({
      ...day,
      mood: 9,
      symptoms: { nausea: 'extreme', hiccups: 'mild', vomiting: null, headache: true },
      water_glasses: 40,
      weight: 0,
      visit_note: '   ',
      last_weight: { value: null },
    });
    expect(parsed.mood).toBeNull();
    expect(parsed.symptoms).toEqual({ nausea: 'mild', headache: 'mild' });
    expect(parsed.waterGlasses).toBe(15);
    expect(parsed.weight).toBeNull();
    expect(parsed.visitNote).toBeNull();
    expect(parsed.lastWeight).toBeNull();
  });

  it('accepts a list-shaped symptom set and the alerts a save raised', () => {
    const parsed = pregnancyDaySchema.parse({
      log_date: '2026-09-22',
      symptoms: [{ key: 'spotting', severity: 'severe' }, 'nausea'],
      alerts: [{ id: 5, level: 'urgent', title: 'لکه‌بینی شدید', contact: { text: 'با ۱۱۵ تماس بگیر', phone: '115' } }],
    });
    expect(parsed.symptoms).toEqual({ spotting: 'severe', nausea: 'mild' });
    expect(parsed.alerts[0]).toMatchObject({ id: 5, level: 'urgent', contact: { text: 'با ۱۱۵ تماس بگیر', phone: '115' } });
  });
});

describe('pregnancyAlertsV2Schema', () => {
  const alerts = {
    window_days: 7,
    alerts: [
      {
        id: 31,
        rule_key: 'vomiting_streak',
        level: 'follow_up',
        title: 'استفراغ ۴ روز پشت سر هم ثبت شده',
        what_we_saw: 'استفراغ در ۴ ثبت اخیر',
        how_sure: 'این الگو از ثبت‌های خودته، نه تشخیص',
        advice: 'تهوع و استفراغ در این هفته‌ها رایجه…',
        actions: [{ key: 'add_to_visit_note', label: 'افزودن به یادداشت ویزیت' }, 'ack'],
        contact: { text: 'should be dropped for non-urgent' },
        created_at: '2026-09-22T08:00:00Z',
        date_label: 'امروز',
        is_read: false,
      },
      { id: 32, rule_key: 'weight_missing_week', payload: { level4: 'suggestion' }, title: 'وزن این هفته ثبت نشده', actions: ['log_weight'] },
      { id: 33, level: 'catastrophic', title: 'unknown level' },
      { id: 34, level: 'info', title: 'وارد هفتهٔ ۸ شدی', read: true },
    ],
    legend: [
      { level: 'urgent', label: 'پیگیری زودتر', text: 'شرایطی که…' },
      { level: 'info', label: 'اطلاع', text: 'اتفاق عادی…' },
      { level: 'nope' },
    ],
  };

  it('maps the Alerts artboard payload', () => {
    const parsed = pregnancyAlertsV2Schema.parse(alerts);
    expect(parsed.windowDays).toBe(7);
    expect(parsed.alerts.map((a) => [a.id, a.level])).toEqual([
      [31, 'follow_up'],
      [32, 'suggestion'],
      [34, 'info'],
    ]);
    expect(parsed.alerts[0]).toMatchObject({
      ruleKey: 'vomiting_streak',
      whatWeSaw: 'استفراغ در ۴ ثبت اخیر',
      howSure: 'این الگو از ثبت‌های خودته، نه تشخیص',
      actions: [
        { key: 'add_to_visit_note', label: 'افزودن به یادداشت ویزیت' },
        { key: 'ack', label: null },
      ],
      contact: null,
      factDate: '2026-09-22',
      isRead: false,
    });
    expect(parsed.alerts[2].isRead).toBe(true);
    // fact_date wins over the creation day when the server sends it.
    expect(pregnancyAlertsV2Schema.parse({ ...alerts, alerts: [{ ...alerts.alerts[0], fact_date: '2026-09-19' }] }).alerts[0].factDate).toBe(
      '2026-09-19',
    );
    // Legend ordered info → urgent; unknown levels dropped.
    expect(parsed.legend.map((l) => l.level)).toEqual(['info', 'urgent']);
  });

  it('accepts a bare array and an empty payload', () => {
    expect(pregnancyAlertsV2Schema.parse([alerts.alerts[3]]).alerts).toHaveLength(1);
    expect(pregnancyAlertsV2Schema.parse(null)).toEqual({ windowDays: 7, alerts: [], legend: [] });
  });
});

describe('pregnancyCalendarSchema', () => {
  const calendar = {
    month: '2026-10',
    month_label: 'مهر ۱۴۰۵',
    week_range: { from: 8, to: 12 },
    days: [
      { date: '2026-10-03', week_start: 9 },
      { date: '2026-10-10', has_visit: true, week_start: 11 },
      { date: 'bad' },
    ],
    visits: [
      {
        appointment_id: 12,
        care_item_key: 'nt_scan',
        title: 'سونوگرافی NT',
        date: '2026-10-10',
        time: '10:30',
        week: 11,
        week_label: 'هفتهٔ ۱۱ و ۱ روز',
        stage: 'booked',
        prep: 'ناشتایی لازم نیست.',
        doctor: '[نام پزشک]',
        place: '[نام مرکز تصویربرداری]',
        remind_before: '1d',
      },
    ],
    next_visit: null,
    care_plan: [
      { key: 'first_visit', title: 'اولین ویزیت پزشک', kind: 'visit', state: 'done', date: '2026-09-16' },
      { key: 'nt_scan', title: 'سونوگرافی NT و غربالگری اول', kind: 'scan', week_from: 11, week_to: 14, state: 'booked', appointment_id: 12 },
      { key: 'anomaly', title: 'سونوگرافی آنومالی', kind: 'scan', week_from: 18, week_to: 22, window: { from: '2026-11-21', to: '2026-12-25' }, suggested_date: '2026-12-15' },
      { key: 'tdap', title: 'واکسن سه‌گانه (Tdap)', kind: 'shot', week_from: 27, week_to: 36, state: 'weird' },
    ],
    source_note: 'زمان‌ها بر اساس برنامهٔ رایج…',
  };

  it('maps the Calendar artboard payload', () => {
    const parsed = pregnancyCalendarSchema.parse(calendar);
    expect(parsed.month).toBe('2026-10');
    expect(parsed.weekRange).toEqual({ from: 8, to: 12 });
    expect(parsed.days).toEqual([
      { date: '2026-10-03', hasVisit: false, weekStart: 9, isToday: false },
      { date: '2026-10-10', hasVisit: true, weekStart: 11, isToday: false },
    ]);
    expect(parsed.visits[0]).toMatchObject({ appointmentId: 12, stage: 'booked', doctor: '[نام پزشک]' });
    expect(parsed.carePlan.map((c) => [c.key, c.state, c.kind])).toEqual([
      ['first_visit', 'done', 'visit'],
      ['nt_scan', 'booked', 'scan'],
      ['anomaly', 'to_book', 'scan'],
      ['tdap', 'to_book', null],
    ]);
    expect(parsed.carePlan[2]).toMatchObject({
      window: { from: '2026-11-21', to: '2026-12-25' },
      suggestedDate: '2026-12-15',
    });
  });

  it('rejects a payload without a month', () => {
    expect(() => pregnancyCalendarSchema.parse({ days: [] })).toThrow();
  });
});

describe('pregnancyReportSchema', () => {
  it('maps and date-sorts the doctor report data', () => {
    const parsed = pregnancyReportSchema.parse({
      from: '2026-09-01',
      to: '2026-09-22',
      profile: { weeks: 8, days: 3, due_date: '2027-05-01' },
      days: [
        { date: '2026-09-22', week: 8, mood: 2, symptoms: { vomiting: 'severe' }, visit_note: 'x' },
        { date: '2026-09-20', week: 8, symptoms: {} },
      ],
      weights: [{ date: '2026-09-22', week: 8, value: '62.4' }, { date: '2026-09-16', value: 62.1 }, { date: '2026-09-10' }],
    });
    expect(parsed.range).toEqual({ from: '2026-09-01', to: '2026-09-22' });
    expect(parsed.age).toEqual({ weeks: 8, days: 3 });
    expect(parsed.dueDate).toBe('2027-05-01');
    expect(parsed.days.map((d) => d.date)).toEqual(['2026-09-20', '2026-09-22']);
    expect(parsed.days[1]).toMatchObject({ mood: 2, symptoms: { vomiting: 'severe' }, visitNote: 'x' });
    expect(parsed.weights).toEqual([
      { date: '2026-09-16', week: null, value: 62.1 },
      { date: '2026-09-22', week: 8, value: 62.4 },
    ]);
  });
});

describe('pregnancyKeys.v2', () => {
  it('nests every v2 key under pregnancy/v2 so v1 mode switches refresh it', () => {
    const v2 = pregnancyKeys.v2;
    for (const key of [
      v2.today(),
      v2.week(8),
      v2.day('2026-09-22'),
      v2.calendar('2026-10'),
      v2.alerts(),
      v2.datingPreview('{}'),
      v2.report({ from: '2026-09-01', to: '2026-09-22' }),
    ]) {
      expect(key.slice(0, 2)).toEqual(['pregnancy', 'v2']);
      expect(key.slice(0, 1)).toEqual(pregnancyKeys.all);
    }
    expect(v2.week(8).slice(0, 3)).toEqual(v2.weekAll());
    expect(v2.day('2026-09-22').slice(0, 3)).toEqual(v2.dayAll());
    expect(v2.calendar('2026-10').slice(0, 3)).toEqual(v2.calendarAll());
    expect(v2.report({ from: 'a', to: 'b' }).slice(0, 3)).toEqual(v2.reportAll());
  });
});
