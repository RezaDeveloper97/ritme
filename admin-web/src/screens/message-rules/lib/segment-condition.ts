import { emptyCond, type CondForm, type ConditionsForm } from '@/features/rule-conditions';

/** A rule targets a saved segment through the `segment` set fact (T-M10-32): `segment contains <id>` in the «all of» group. */
export const SEGMENT_FACT = 'segment';

const isSegmentRow = (c: CondForm) => c.fact.trim() === SEGMENT_FACT && c.op === 'contains';

/** Segment ids named by the «all of» rows (as stored: text). */
export function segmentIds(f: ConditionsForm): string[] {
  return f.allOf.filter(isSegmentRow).map((c) => c.value.trim()).filter((v) => v !== '');
}

/** The form without the segment rows, for the generic builder (never leaves it with zero rows). */
export function withoutSegments(f: ConditionsForm): ConditionsForm {
  const allOf = f.allOf.filter((c) => !isSegmentRow(c));
  return { ...f, allOf: allOf.length === 0 && f.anyOf.length === 0 ? [emptyCond()] : allOf };
}

/** Put the segment rows (from `ids`) back in front of the builder's «all of» rows; a lone blank builder row is dropped when a segment exists. */
export function withSegments(builder: ConditionsForm, ids: readonly string[]): ConditionsForm {
  const seg: CondForm[] = ids.map((id) => ({ fact: SEGMENT_FACT, op: 'contains', value: id }));
  const rest = builder.allOf.filter((c) => !isSegmentRow(c));
  const blank = rest.length === 1 && rest[0]?.fact.trim() === '';
  const allOf = [...seg, ...(seg.length > 0 && blank ? [] : rest)];
  return { ...builder, allOf: allOf.length === 0 ? [emptyCond()] : allOf };
}

export const addSegment = (ids: readonly string[], id: string): string[] => (ids.includes(id) ? [...ids] : [...ids, id]);
export const removeSegment = (ids: readonly string[], id: string): string[] => ids.filter((x) => x !== id);
