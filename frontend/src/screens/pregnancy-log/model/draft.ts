import {
  DAY_SYMPTOMS,
  type DaySymptom,
  type DaySymptoms,
  type Mood,
  type PregnancyDay,
  type PregnancyDayInput,
  type Severity,
  WATER_MAX,
  WATER_MIN,
} from "@/entities/pregnancy";

/** The Log screen's editable state for one day. */
export interface DayDraft {
  mood: Mood | null;
  symptoms: DaySymptoms;
  water: number;
  /** Raw text of the weight field (may hold Persian digits or a comma). */
  weight: string;
  visitNote: string;
}

export const EMPTY_DRAFT: DayDraft = {
  mood: null,
  symptoms: {},
  water: 0,
  weight: "",
  visitNote: "",
};

export function draftFromDay(day: PregnancyDay | null | undefined): DayDraft {
  if (!day) return EMPTY_DRAFT;
  return {
    mood: day.mood,
    symptoms: { ...day.symptoms },
    water: day.waterGlasses ?? 0,
    weight: day.weight != null ? String(day.weight) : "",
    visitNote: day.visitNote ?? "",
  };
}

/** Toggle a symptom on (at `mild`) or off. */
export function toggleSymptom(
  symptoms: DaySymptoms,
  key: DaySymptom,
): DaySymptoms {
  const next = { ...symptoms };
  if (next[key]) delete next[key];
  else next[key] = "mild";
  return next;
}

export function setSeverity(
  symptoms: DaySymptoms,
  key: DaySymptom,
  severity: Severity,
): DaySymptoms {
  return symptoms[key] ? { ...symptoms, [key]: severity } : symptoms;
}

/** Selected symptoms in canonical order. */
export function selectedSymptoms(symptoms: DaySymptoms): DaySymptom[] {
  return DAY_SYMPTOMS.filter((k) => !!symptoms[k]);
}

export function stepWater(water: number, delta: number): number {
  return Math.min(WATER_MAX, Math.max(WATER_MIN, water + delta));
}

const FA_DIGITS = "۰۱۲۳۴۵۶۷۸۹";
const AR_DIGITS = "٠١٢٣٤٥٦٧٨٩";

/** Weight text → kg, or `null` for blank / unreadable / out of a sane range. */
export function parseWeight(text: string): number | null {
  const ascii = text
    .trim()
    .replace(/[۰-۹]/g, (d) => String(FA_DIGITS.indexOf(d)))
    .replace(/[٠-٩]/g, (d) => String(AR_DIGITS.indexOf(d)))
    .replace(/[٫,]/g, ".");
  if (!ascii) return null;
  const n = Number(ascii);
  return Number.isFinite(n) && n >= 20 && n <= 300 ? n : null;
}

/** Whole-day input for the PUT — every field present, so cleared ones clear. */
export function inputFromDraft(draft: DayDraft): PregnancyDayInput {
  return {
    mood: draft.mood,
    symptoms: draft.symptoms,
    waterGlasses: draft.water,
    weight: parseWeight(draft.weight),
    visitNote: draft.visitNote,
  };
}
