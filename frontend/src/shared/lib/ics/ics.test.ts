import { describe, expect, it } from 'vitest';

import { alarmTrigger, buildIcs, escapeIcsText, foldLine } from './ics';

const base = {
  uid: 'appointment-7@ritme.app',
  title: 'سونوگرافی',
  start: '2026-10-05 10:30:00',
  stamp: new Date(Date.UTC(2026, 8, 26, 8, 0, 0)),
};

describe('buildIcs', () => {
  it('writes the start in Tehran time with an embedded +03:30 VTIMEZONE', () => {
    const ics = buildIcs(base);
    expect(ics).toContain('DTSTART;TZID=Asia/Tehran:20261005T103000');
    expect(ics).toContain('DTEND;TZID=Asia/Tehran:20261005T113000');
    expect(ics).toContain('TZID:Asia/Tehran');
    expect(ics).toContain('TZOFFSETTO:+0330');
    expect(ics).toContain('DTSTAMP:20260926T080000Z');
    expect(ics.split('\r\n')[0]).toBe('BEGIN:VCALENDAR');
    expect(ics.endsWith('END:VCALENDAR\r\n')).toBe(true);
  });

  it('rolls the end over midnight / month end', () => {
    const ics = buildIcs({ ...base, start: '2026-10-31 23:30', durationMinutes: 60 });
    expect(ics).toContain('DTEND;TZID=Asia/Tehran:20261101T003000');
  });

  it.each([
    [60, '-PT1H'],
    [180, '-PT3H'],
    [1440, '-P1D'],
    [2880, '-P2D'],
  ])('alarm %i min → %s', (minutes, trigger) => {
    const ics = buildIcs({ ...base, alarmMinutesBefore: minutes });
    expect(ics).toContain('BEGIN:VALARM');
    expect(ics).toContain(`TRIGGER:${trigger}`);
  });

  it('omits the alarm when none is asked for', () => {
    expect(buildIcs(base)).not.toContain('VALARM');
  });

  it('escapes text and includes location/description', () => {
    const ics = buildIcs({ ...base, location: 'Tehran, Vali-Asr; No 5', description: 'a\nb' });
    expect(ics).toContain('LOCATION:Tehran\\, Vali-Asr\; No 5');
    expect(ics).toContain('DESCRIPTION:a\\nb');
  });

  it('rejects a malformed start', () => {
    expect(() => buildIcs({ ...base, start: 'tomorrow' })).toThrow();
  });
});

describe('helpers', () => {
  it('alarmTrigger falls back to minutes', () => {
    expect(alarmTrigger(90)).toBe('-PT90M');
  });
  it('escapeIcsText escapes backslashes first', () => {
    expect(escapeIcsText('a\\b,c')).toBe('a\\\\b\\,c');
  });
  it('foldLine keeps each line within 75 octets and never splits a character', () => {
    const folded = foldLine('SUMMARY:' + 'سونوگرافی '.repeat(20));
    const enc = new TextEncoder();
    for (const line of folded.split('\r\n')) expect(enc.encode(line).length).toBeLessThanOrEqual(75);
    expect(folded.replace(/\r\n /g, '')).toBe('SUMMARY:' + 'سونوگرافی '.repeat(20));
  });
});
