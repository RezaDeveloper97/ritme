import type { CreditPack } from '../model/types';

/** The pack the screen starts on: the highlighted one, else the first. Null with no packs. */
export function defaultPack(packs: readonly CreditPack[]): CreditPack | null {
  return packs.find((p) => p.isHighlighted) ?? packs[0] ?? null;
}

/** A `?pack=` value → the pack it names (null when absent / unknown / not a whole id). */
export function packFromParam(packs: readonly CreditPack[], raw: string | null): CreditPack | null {
  if (!raw || !/^\d+$/.test(raw)) return null;
  return packs.find((p) => p.id === Number(raw)) ?? null;
}
