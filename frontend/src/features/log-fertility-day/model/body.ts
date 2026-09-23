import { type FertilityDayInput, FERTILITY_SYMPTOMS, roundBbt } from '@/entities/fertility';

/**
 * camelCase day input → the `PUT /fertility/days/{date}` body.
 *
 * - `undefined` fields are omitted (leave as is), explicit `null` is sent
 *   (clear) — the README contract;
 * - `bbt` is rounded to 2 decimals (`decimal(4,2)`); the 35.00–38.50 range is
 *   the server's to enforce (422, localized) — the form pre-checks it with
 *   `isBbtInRange` for instant feedback;
 * - symptoms are de-duplicated and kept in canonical order; `[]` clears them;
 * - a blank note becomes `null`.
 */
export function toFertilityDayBody(input: FertilityDayInput): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (input.lh !== undefined) body.lh = input.lh;
  if (input.mucus !== undefined) body.mucus = input.mucus;
  if (input.bbt !== undefined) {
    body.bbt = input.bbt === null || !Number.isFinite(input.bbt) ? null : roundBbt(input.bbt);
  }
  if (input.bbtTime !== undefined) body.bbt_time = input.bbtTime || null;
  if (input.intercourse !== undefined) body.intercourse = input.intercourse;
  if (input.symptoms !== undefined) {
    const chosen = new Set(input.symptoms);
    body.symptoms = FERTILITY_SYMPTOMS.filter((s) => chosen.has(s));
  }
  if (input.note !== undefined) body.note = input.note?.trim() || null;
  return body;
}
