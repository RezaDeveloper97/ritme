import { describe, expect, it } from 'vitest';

import type { CareItem } from '@/entities/pregnancy';

import { bookHref, careItemTopic, monthKey, nextStage, stageProgress } from './view';

describe('pregnancy calendar view helpers', () => {
  it('pads the month key', () => {
    expect(monthKey(1405, 7)).toBe('1405-07');
  });

  it('walks the visit stages', () => {
    expect(nextStage(null)).toBe('done');
    expect(nextStage('booked')).toBe('done');
    expect(nextStage('done')).toBe('result');
    expect(nextStage('result')).toBeNull();
    expect(stageProgress(null)).toBe(1);
    expect(stageProgress('result')).toBe(3);
  });

  it('prefills AddAppointment from a care item', () => {
    const item = {
      key: 'nt_scan',
      kind: 'scan',
      title: 'سونوگرافی NT',
      suggestedDate: '2026-10-10',
      date: null,
    } as unknown as CareItem;
    const url = new URL(bookHref(item), 'https://x');
    expect(url.pathname).toBe('/reminders/appointment/new');
    expect(url.searchParams.get('care_item_key')).toBe('nt_scan');
    expect(url.searchParams.get('title')).toBe('سونوگرافی NT');
    expect(url.searchParams.get('date')).toBe('2026-10-10');
    expect(url.searchParams.get('kind')).toBe('in_person');
    expect(url.searchParams.get('topic')).toBe('ultrasound');
  });

  it('maps every care-plan kind to its appointment topic', () => {
    expect(careItemTopic('scan')).toBe('ultrasound');
    expect(careItemTopic('test')).toBe('lab');
    expect(careItemTopic('vaccine')).toBe('vaccine');
    expect(careItemTopic('visit')).toBe('checkup');
    expect(careItemTopic(null)).toBe('checkup');
  });
});
