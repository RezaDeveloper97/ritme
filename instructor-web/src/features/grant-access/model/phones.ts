import { normalizeMobile } from '@/shared/lib';

/** API cap per POST /grants (InstructorGrantInput.phones maxItems). */
export const MAX_PER_REQUEST = 200;

export interface ParsedPhones {
  /** Valid, normalised (`09…`), deduplicated, in input order. */
  valid: string[];
  /** Lines that are not a mobile number (as typed, trimmed). */
  invalid: string[];
  /** How many valid entries were repeats. */
  duplicates: number;
}

function collect(entries: string[]): ParsedPhones {
  const seen = new Set<string>();
  const out: ParsedPhones = { valid: [], invalid: [], duplicates: 0 };
  for (const raw of entries) {
    const entry = raw.trim();
    if (!entry) continue;
    const phone = normalizeMobile(entry);
    if (!phone) out.invalid.push(entry);
    else if (seen.has(phone)) out.duplicates += 1;
    else {
      seen.add(phone);
      out.valid.push(phone);
    }
  }
  return out;
}

/** Multi-line input: one number per line (commas / semicolons / tabs also separate). */
export function parsePhones(text: string): ParsedPhones {
  return collect(text.split(/[\n\r,;،؛\t]+/));
}

/**
 * CSV / text file: every row contributes its first cell that is a mobile number; a first row with no
 * number is a header and skipped. Rows without a number elsewhere count as invalid.
 */
export function parseCsv(text: string): ParsedPhones {
  const rows = text.replace(/^﻿/, '').split(/\r?\n/).filter((l) => l.trim());
  const picked: string[] = [];
  rows.forEach((row, i) => {
    const cells = row.split(/[,;\t،؛]/).map((c) => c.trim().replace(/^"|"$/g, ''));
    const hit = cells.find((c) => normalizeMobile(c));
    if (hit) picked.push(hit);
    else if (i > 0 || cells.every((c) => /\d/.test(c))) picked.push(cells.find((c) => c) ?? row);
  });
  return collect(picked);
}

export function chunk<T>(items: T[], size = MAX_PER_REQUEST): T[][] {
  const out: T[][] = [];
  for (let i = 0; i < items.length; i += size) out.push(items.slice(i, i + size));
  return out;
}

/** Merge two parse results (typed + file), deduplicating across them. */
export function mergePhones(a: ParsedPhones, b: ParsedPhones): ParsedPhones {
  const merged = collect([...a.valid, ...b.valid]);
  return {
    valid: merged.valid,
    invalid: [...a.invalid, ...b.invalid],
    duplicates: a.duplicates + b.duplicates + merged.duplicates,
  };
}
