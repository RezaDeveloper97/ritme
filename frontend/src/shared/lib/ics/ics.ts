/**
 * Minimal RFC 5545 builder for a single appointment event (VEVENT + VALARM).
 *
 * Times cross the API as Tehran wall-clock (`Y-m-d H:i:s`), so the event is
 * written in `TZID=Asia/Tehran` with an embedded VTIMEZONE (+03:30, no DST
 * since 2022) — calendar apps then show the visit at the time the user typed,
 * whatever zone the phone is in. Pure and domain-agnostic: the caller maps its
 * own "remind before" option to minutes.
 */

export const ICS_TIMEZONE = 'Asia/Tehran';

export interface IcsEvent {
  /** Stable unique id, e.g. `appointment-12@ritme.app`. */
  uid: string;
  title: string;
  description?: string | null;
  location?: string | null;
  /** Tehran wall-clock `Y-m-d H:i[:s]`. */
  start: string;
  /** Event length in minutes (default 60). */
  durationMinutes?: number;
  /** Alarm this many minutes before the start; omitted/null = no alarm. */
  alarmMinutesBefore?: number | null;
  /** DTSTAMP; defaults to now. */
  stamp?: Date;
}

const VTIMEZONE = [
  'BEGIN:VTIMEZONE',
  `TZID:${ICS_TIMEZONE}`,
  'BEGIN:STANDARD',
  'DTSTART:19700101T000000',
  'TZOFFSETFROM:+0330',
  'TZOFFSETTO:+0330',
  'TZNAME:+0330',
  'END:STANDARD',
  'END:VTIMEZONE',
];

const pad = (n: number): string => String(n).padStart(2, '0');

/** `Y-m-d H:i[:s]` → `{y,mo,d,h,mi,s}` (throws on garbage so no bogus file is made). */
function parseWallClock(value: string): number[] {
  const m = /^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?$/.exec(value.trim());
  if (!m) throw new Error('ics: invalid start');
  return m.slice(1).map((part) => Number(part ?? 0));
}

/** Local (floating, TZID-qualified) `YYYYMMDDTHHMMSS`, with minutes added. */
function wallClockPlus(value: string, minutes: number): string {
  const [y, mo, d, h, mi, s] = parseWallClock(value);
  // UTC arithmetic on the wall-clock fields: no zone is applied, only carried.
  const t = new Date(Date.UTC(y!, mo! - 1, d!, h!, mi! + minutes, s || 0));
  return (
    `${t.getUTCFullYear()}${pad(t.getUTCMonth() + 1)}${pad(t.getUTCDate())}` +
    `T${pad(t.getUTCHours())}${pad(t.getUTCMinutes())}${pad(t.getUTCSeconds())}`
  );
}

function utcStamp(date: Date): string {
  return date.toISOString().replace(/[-:]/g, '').replace(/\.\d{3}/, '');
}

/** TEXT escaping (RFC 5545 §3.3.11). */
export function escapeIcsText(value: string): string {
  return value
    .replace(/\\/g, '\\\\')
    .replace(/;/g, '\;')
    .replace(/,/g, '\\,')
    .replace(/\r?\n/g, '\\n');
}

/** Duration as `-PT1H` / `-P1D` / `-PT90M`. */
export function alarmTrigger(minutes: number): string {
  if (minutes > 0 && minutes % 1440 === 0) return `-P${minutes / 1440}D`;
  if (minutes > 0 && minutes % 60 === 0) return `-PT${minutes / 60}H`;
  return `-PT${Math.max(0, Math.round(minutes))}M`;
}

/** Folds a content line at 75 octets without splitting a UTF-8 character. */
export function foldLine(line: string): string {
  const encoder = new TextEncoder();
  const out: string[] = [];
  let current = '';
  let octets = 0;
  for (const ch of line) {
    const size = encoder.encode(ch).length;
    const limit = out.length === 0 ? 75 : 74; // continuation lines start with a space
    if (octets + size > limit) {
      out.push(current);
      current = '';
      octets = 0;
    }
    current += ch;
    octets += size;
  }
  out.push(current);
  return out.join('\r\n ');
}

/** Builds the `.ics` text (CRLF line endings). */
export function buildIcs(event: IcsEvent): string {
  const duration = event.durationMinutes ?? 60;
  const lines = [
    'BEGIN:VCALENDAR',
    'VERSION:2.0',
    'PRODID:-//Ritme//Care reminders//EN',
    'CALSCALE:GREGORIAN',
    'METHOD:PUBLISH',
    ...VTIMEZONE,
    'BEGIN:VEVENT',
    `UID:${event.uid}`,
    `DTSTAMP:${utcStamp(event.stamp ?? new Date())}`,
    `DTSTART;TZID=${ICS_TIMEZONE}:${wallClockPlus(event.start, 0)}`,
    `DTEND;TZID=${ICS_TIMEZONE}:${wallClockPlus(event.start, duration)}`,
    `SUMMARY:${escapeIcsText(event.title)}`,
  ];
  if (event.description?.trim()) lines.push(`DESCRIPTION:${escapeIcsText(event.description.trim())}`);
  if (event.location?.trim()) lines.push(`LOCATION:${escapeIcsText(event.location.trim())}`);
  if (event.alarmMinutesBefore != null) {
    lines.push(
      'BEGIN:VALARM',
      'ACTION:DISPLAY',
      `DESCRIPTION:${escapeIcsText(event.title)}`,
      `TRIGGER:${alarmTrigger(event.alarmMinutesBefore)}`,
      'END:VALARM',
    );
  }
  lines.push('END:VEVENT', 'END:VCALENDAR');
  return lines.map(foldLine).join('\r\n') + '\r\n';
}

/** Triggers a browser download of the `.ics` (client only). */
export function downloadIcs(filename: string, content: string): void {
  const blob = new Blob([content], { type: 'text/calendar;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename.endsWith('.ics') ? filename : `${filename}.ics`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}
