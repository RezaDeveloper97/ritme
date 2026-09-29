import { describe, expect, it } from 'vitest';

import { checkupIcon } from '../model/icon';
import { checkupKeys } from './keys';
import {
  checkupDetailSchema,
  checkupHomeSchema,
  checkupItemSchema,
  checkupListSchema,
  checkupNextPreviewSchema,
  checkupRecordPageSchema,
  checkupRecordResultSchema,
  checkupRecordSchema,
} from './schema';

/*
 * Boundary contract for `/api/v1/checkups/*` — fixtures follow
 * docs/checkups/README.md and the v14 artboards. The frontend was written
 * before the Go endpoints, so the parsers must also survive values this bundle
 * has never seen and shapes the README leaves open.
 */

const papItem = {
  id: 3,
  key: 'pap_smear',
  title: 'پاپ‌اسمیر / HPV',
  subtitle: 'غربالگری دهانه رحم',
  category: 'multi_year',
  section: 'overdue',
  status: 'overdue',
  icon: 'shield-check',
  tone: 'rose',
  interval_label: 'هر ۳ سال',
  timing_label: 'روز ۱۰ تا ۲۰ سیکل',
  last_done_on: '2022-03-25',
  next_due_on: '2025-03-25',
  next_due_label: 'عقب‌افتاده از فروردین',
  is_custom: false,
};

const selfExamItem = {
  id: 1,
  key: 'breast_self_exam',
  title: 'خودآزمایی سینه',
  subtitle: null,
  category: 'monthly',
  section: 'this_month',
  status: 'due',
  icon: 'ribbon',
  tone: 'rose',
  interval_label: 'هر ماه',
  timing_label: 'روز ۷ تا ۱۰ سیکل',
  last_done_on: '2026-08-14',
  next_due_on: '2026-09-26',
  next_due_label: '۳ روز دیگر',
  is_custom: false,
};

describe('checkupItemSchema', () => {
  it('maps a README item to camelCase', () => {
    expect(checkupItemSchema.parse(papItem)).toEqual({
      id: 3,
      key: 'pap_smear',
      title: 'پاپ‌اسمیر / HPV',
      subtitle: 'غربالگری دهانه رحم',
      category: 'multi_year',
      section: 'overdue',
      status: 'overdue',
      icon: 'shield-check',
      tone: 'rose',
      intervalLabel: 'هر ۳ سال',
      timingLabel: 'روز ۱۰ تا ۲۰ سیکل',
      lastDoneOn: '2022-03-25',
      nextDueOn: '2025-03-25',
      nextDueLabel: 'عقب‌افتاده از فروردین',
      isCustom: false,
    });
  });

  it('falls back safely on unknown enum values', () => {
    const item = checkupItemSchema.parse({
      ...papItem,
      status: 'critical',
      tone: 'magenta',
      category: 'weekly',
      section: 'next_decade',
    });
    expect(item.status).toBe('up_to_date');
    expect(item.tone).toBe('neutral');
    expect(item.category).toBe('custom');
    // An unknown section falls back to the (fallen-back) category.
    expect(item.section).toBe('custom');
  });

  it('an unknown section falls back to a known category', () => {
    expect(checkupItemSchema.parse({ ...papItem, section: undefined }).section).toBe('multi_year');
  });

  it('tolerates a minimal row: string id, missing optional fields, ISO dates', () => {
    const item = checkupItemSchema.parse({
      id: '9',
      title: 'چکاپ چشم',
      category: 'custom',
      last_done_on: '2026-01-02T00:00:00Z',
    });
    expect(item).toMatchObject({
      id: 9,
      key: null,
      subtitle: null,
      status: 'up_to_date',
      tone: 'neutral',
      icon: null,
      lastDoneOn: '2026-01-02',
      nextDueOn: null,
      isCustom: true,
    });
  });

  it('reads a raw bilingual title instead of failing', () => {
    expect(checkupItemSchema.parse({ ...papItem, title: { fa: 'پاپ‌اسمیر', en: 'Pap smear' } }).title).toBe(
      'پاپ‌اسمیر',
    );
    expect(checkupItemSchema.parse({ ...papItem, title: { fa: '', en: 'Pap smear' } }).title).toBe(
      'Pap smear',
    );
  });

  it('rejects a row without a title', () => {
    expect(checkupItemSchema.safeParse({ ...papItem, title: '' }).success).toBe(false);
  });
});

describe('checkupListSchema', () => {
  it('parses GET /checkups and drops only the malformed row', () => {
    const list = checkupListSchema.parse({
      age: 34,
      summary: { total: 6, up_to_date: 4, due: 1, overdue: 1 },
      items: [selfExamItem, { id: 'x' }, papItem],
    });
    expect(list.age).toBe(34);
    expect(list.summary).toEqual({ total: 6, upToDate: 4, due: 1, overdue: 1 });
    expect(list.items.map((i) => i.id)).toEqual([1, 3]);
  });

  it('derives the summary when the server omits it (soon counts as up to date, not_yet is excluded)', () => {
    const list = checkupListSchema.parse({
      items: [
        selfExamItem,
        papItem,
        { ...papItem, id: 4, status: 'soon' },
        { ...papItem, id: 5, status: 'up_to_date' },
        { ...papItem, id: 6, status: 'not_yet' },
      ],
    });
    expect(list.age).toBeNull();
    expect(list.summary).toEqual({ total: 4, upToDate: 2, due: 1, overdue: 1 });
  });

  it('survives an empty payload', () => {
    expect(checkupListSchema.parse({})).toEqual({
      age: null,
      summary: { total: 0, upToDate: 0, due: 0, overdue: 0 },
      items: [],
    });
  });
});

describe('checkupHomeSchema', () => {
  it('parses the home card and caps highlights at two', () => {
    const home = checkupHomeSchema.parse({
      summary: { total: 6, up_to_date: 4, due: 1, overdue: 1 },
      highlights: [selfExamItem, papItem, { ...papItem, id: 7 }],
    });
    expect(home?.summary.total).toBe(6);
    expect(home?.highlights.map((h) => h.id)).toEqual([1, 3]);
  });

  it('is null when nothing applies (no data, or total 0)', () => {
    expect(checkupHomeSchema.parse(null)).toBeNull();
    expect(checkupHomeSchema.parse(undefined)).toBeNull();
    expect(
      checkupHomeSchema.parse({ summary: { total: 0, up_to_date: 0, due: 0, overdue: 0 }, highlights: [] }),
    ).toBeNull();
  });
});

const record = {
  id: 55,
  checkup_type_id: 3,
  done_on: '2022-03-25',
  result: 'normal',
  findings: null,
  note: null,
  has_attachment: true,
  next_due_on: null,
};

describe('checkupRecordSchema', () => {
  it('maps a record', () => {
    expect(checkupRecordSchema.parse(record)).toEqual({
      id: 55,
      checkupTypeId: 3,
      checkupTitle: null,
      checkupKey: null,
      checkupIcon: null,
      checkupTone: 'neutral',
      doneOn: '2022-03-25',
      result: 'normal',
      findings: [],
      note: null,
      hasAttachment: true,
      nextDueOn: null,
    });
  });

  it('unknown result → pending; findings as a JSON string; flags as 0/1', () => {
    const r = checkupRecordSchema.parse({
      ...record,
      result: 'abnormal',
      findings: '["lump","skin","lump"]',
      has_attachment: 0,
      checkup_title: 'خودآزمایی سینه',
    });
    expect(r.result).toBe('pending');
    expect(r.findings).toEqual(['lump', 'skin', 'lump']);
    expect(r.hasAttachment).toBe(false);
    expect(r.checkupTitle).toBe('خودآزمایی سینه');
  });

  it('history rows carry the type key, icon and tone (unknown tone → neutral)', () => {
    const r = checkupRecordSchema.parse({ ...record, checkup_key: 'dentist', checkup_icon: 'tooth', checkup_tone: 'green' });
    expect([r.checkupKey, r.checkupIcon, r.checkupTone]).toEqual(['dentist', 'tooth', 'green']);
    expect(checkupRecordSchema.parse({ ...record, checkup_tone: 'purple' }).checkupTone).toBe('neutral');
  });

  it('rejects a record without a date', () => {
    expect(checkupRecordSchema.safeParse({ ...record, done_on: null }).success).toBe(false);
  });
});

describe('checkupRecordPageSchema', () => {
  it('reads {items, pagination}', () => {
    const page = checkupRecordPageSchema.parse({
      items: [record, { id: 56 }],
      pagination: { current_page: 1, last_page: 3, per_page: 20, total: 41 },
    });
    expect(page.records.map((r) => r.id)).toEqual([55]);
    expect(page).toMatchObject({ page: 1, lastPage: 3, total: 41 });
  });

  it('reads {records, meta} and a bare array', () => {
    expect(checkupRecordPageSchema.parse({ records: [record], meta: { current_page: 2, last_page: 2 } })).toEqual(
      expect.objectContaining({ page: 2, lastPage: 2, total: 1 }),
    );
    expect(checkupRecordPageSchema.parse([record])).toMatchObject({ page: 1, lastPage: 1, total: 1 });
    expect(checkupRecordPageSchema.parse(null)).toEqual({ records: [], page: 1, lastPage: 1, total: 0 });
  });
});

describe('checkupDetailSchema', () => {
  const detail = {
    ...selfExamItem,
    why: 'چون…',
    performed_by: 'self',
    interval_months: 1,
    cycle_day_from: 7,
    cycle_day_to: 10,
    prep_steps: ['بهترین زمان: وسط سیکل', { text: 'پرهیز از کرم واژینال' }, 42, null],
    guide_steps: [
      { title: 'جلوی آینه', body: 'با دست‌ها پایین…' },
      'دراز کشیده',
      { body: 'no title' },
    ],
    finding_options: [
      { key: 'none', label: 'چیزی متفاوت نبود' },
      { key: 'lump', label: 'توده یا سفتی' },
      { key: 'skin', label: 'تغییر پوست', exclusive: false },
      'discharge',
      {},
    ],
    records: [
      { ...record, id: 1, done_on: '2025-01-01' },
      { ...record, id: 3, done_on: '2026-08-14' },
      { ...record, id: 2, done_on: '2026-07-10' },
    ],
    settings: { enabled: true, remind: false },
  };

  it('parses the detail with steps, findings, latest 2 records and settings', () => {
    const d = checkupDetailSchema.parse(detail);
    expect(d.performedBy).toBe('self');
    expect(d.cycleDayFrom).toBe(7);
    expect(d.cycleDayTo).toBe(10);
    expect(d.prepSteps).toEqual(['بهترین زمان: وسط سیکل', 'پرهیز از کرم واژینال', '42']);
    expect(d.guideSteps).toEqual([
      { title: 'جلوی آینه', body: 'با دست‌ها پایین…' },
      { title: 'دراز کشیده', body: null },
    ]);
    expect(d.findingOptions).toEqual([
      { key: 'none', label: 'چیزی متفاوت نبود', exclusive: true },
      { key: 'lump', label: 'توده یا سفتی', exclusive: false },
      { key: 'skin', label: 'تغییر پوست', exclusive: false },
      { key: 'discharge', label: 'discharge', exclusive: false },
    ]);
    expect(d.records.map((r) => r.id)).toEqual([3, 2]);
    expect(d.settings).toEqual({ enabled: true, remind: false });
  });

  it('defaults what is missing', () => {
    const d = checkupDetailSchema.parse({ id: 4, title: 'دندان‌پزشکی', performed_by: 'barber' });
    expect(d).toMatchObject({
      why: null,
      performedBy: 'doctor',
      intervalMonths: null,
      prepSteps: [],
      guideSteps: [],
      findingOptions: [],
      records: [],
      settings: { enabled: true, remind: true },
    });
  });
});

describe('checkupRecordResultSchema', () => {
  it('reads {item, record}', () => {
    const r = checkupRecordResultSchema.parse({ item: papItem, record });
    expect(r.item?.id).toBe(3);
    expect(r.record?.id).toBe(55);
  });

  it('reads a bare item (README) or a bare record', () => {
    expect(checkupRecordResultSchema.parse(papItem)).toEqual({
      item: expect.objectContaining({ id: 3 }),
      record: null,
    });
    expect(checkupRecordResultSchema.parse(record)).toEqual({
      item: null,
      record: expect.objectContaining({ id: 55 }),
    });
    expect(checkupRecordResultSchema.parse(undefined)).toEqual({ item: null, record: null });
  });
});

describe('checkupNextPreviewSchema', () => {
  it('maps the MarkDone banner', () => {
    expect(
      checkupNextPreviewSchema.parse({
        next_due_on: '2029-09-16',
        next_due_label: 'شهریور ۱۴۰۸',
        interval_label: '۳ سال بعد',
        reminder_label: 'یادآوری ۱ ماه قبل',
      }),
    ).toEqual({
      nextDueOn: '2029-09-16',
      nextDueLabel: 'شهریور ۱۴۰۸',
      intervalLabel: '۳ سال بعد',
      reminderLabel: 'یادآوری ۱ ماه قبل',
    });
    expect(checkupNextPreviewSchema.parse({})).toEqual({
      nextDueOn: null,
      nextDueLabel: null,
      intervalLabel: null,
      reminderLabel: null,
    });
  });
});

describe('checkupKeys', () => {
  it('nests every variant under its prefix', () => {
    expect(checkupKeys.list('action').slice(0, 2)).toEqual([...checkupKeys.listAll()]);
    expect(checkupKeys.detail(3).slice(0, 2)).toEqual([...checkupKeys.detailAll()]);
    expect(checkupKeys.records({ type: 3 }).slice(0, 2)).toEqual([...checkupKeys.recordsAll()]);
    expect(checkupKeys.records()).toEqual(['checkups', 'records', { filter: 'all', type: null }]);
    expect(checkupKeys.attachment(5).slice(0, 2)).toEqual([...checkupKeys.attachmentsAll()]);
  });
});

describe('checkupIcon', () => {
  it('resolves kebab-case, camelCase and aliases, with fallbacks', () => {
    expect(checkupIcon('shield-check')).toBe('shield');
    expect(checkupIcon('ribbon')).toBe('ribbon');
    expect(checkupIcon('Tooth')).toBe('tooth');
    expect(checkupIcon('unknown', { performedBy: 'lab' })).toBe('flask');
    expect(checkupIcon(null, { category: 'monthly' })).toBe('ribbon');
    expect(checkupIcon(undefined)).toBe('stetho');
  });
});
