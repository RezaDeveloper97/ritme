/*
 * Endometriosis pain diary (CB-COND-03 on `GET|PUT /conditions/pain/{date}`, CB-COND-01). Health data (§11): never
 * log a day, a payload or an error body.
 */

/** «مسکن کمک کرد؟» answers, in the API's order. */
export const PAIN_ANALGESIC_EFFECTS = ['no', 'a_little', 'helped'] as const;
export type PainAnalgesicEffect = (typeof PAIN_ANALGESIC_EFFECTS)[number];

/** Catalog groups of the diary chips: «چه جور دردی؟» and «همراه با». */
export const PAIN_CATALOG_GROUPS = ['pain_types', 'pain_associated'] as const;
export type PainCatalogGroup = (typeof PAIN_CATALOG_GROUPS)[number];

/** One pain-diary day. */
export interface PainDay {
  date: string;
  /** 0–10; null when nothing (or only a level without a score) was logged. */
  score: number | null;
  /** Log taxonomy `pain.location` items. */
  locations: string[];
  /** Log taxonomy `pain.relief` options (written by the log sheet; the diary leaves them alone). */
  relief: string[];
  /** Catalog `pain_types` codes. */
  types: string[];
  /** Catalog `pain_associated` codes. */
  associated: string[];
  /** «نتوانستم سر کار یا درس بروم». */
  missedActivity: boolean | null;
  analgesic: string | null;
  /** `HH:MM`. */
  analgesicTime: string | null;
  analgesicEffect: PainAnalgesicEffect | null;
}

/** The `PUT /conditions/pain/{date}` body the diary sends (relief omitted: it stays as the log sheet left it). */
export interface PainDayInput {
  score: number | null;
  locations: string[] | null;
  types: string[];
  associated: string[];
  missed_activity: boolean;
  analgesic: string | null;
  analgesic_time: string | null;
  analgesic_effect: PainAnalgesicEffect | null;
}
