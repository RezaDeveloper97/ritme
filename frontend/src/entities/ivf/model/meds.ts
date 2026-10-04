import type { IvfCycle, IvfDoseDay, IvfRole, IvfRoute } from './types';

/*
 * The injection schedule (`GET /ivf/meds`, CB-IVF-01; screen CB-IVF-03,
 * nbl_IVF_Meds): trigger, today/tomorrow, the 8-site rotation and every
 * medicine with its inventory. Health data — never log it (CLAUDE.md §11).
 */

/** How the inventory is counted (`ivf_meds.stock_unit`; «قلم» = pen). */
export const IVF_STOCK_UNITS = ['pen', 'vial', 'ampoule', 'prefilled_syringe', 'box', 'other'] as const;
export type IvfStockUnit = (typeof IVF_STOCK_UNITS)[number];

/** Routes that take an injection site (the API answers 422 `site` for the rest). */
export const IVF_INJECTED_ROUTES: readonly IvfRoute[] = ['subcutaneous', 'intramuscular'];

/** At most this many daily dose times (API `times` maxItems). */
export const IVF_MAX_TIMES = 4;

export interface IvfInventory {
  /** Units as counted at `countedAt`. */
  stockUnits: number;
  stockUnit: IvfStockUnit;
  dosesPerUnit: number;
  countedAt: string;
  dosesLeft: number;
  /** Units left today — what an edit sends back to keep the count. */
  unitsLeft: number;
  /** At today's doses per day; null when not scheduled today. */
  daysLeft: number | null;
  runsOutOn: string | null;
  /** Lasts 3 days or fewer and runs out before the medicine ends [needs clinical review]. */
  low: boolean;
}

export interface IvfMed {
  id: number;
  /** The care medication reminder (CB-IVF-06b: its `is_active` switch is `PUT /care/medications/{id}`). */
  reminderId: number | null;
  name: string;
  role: IvfRole;
  route: IvfRoute;
  isTrigger: boolean;
  /** Tehran wall clock `Y-m-d H:i:s` (the trigger only). */
  triggerAt: string | null;
  dose: string | null;
  unit: string | null;
  /** `HH:MM` daily times (empty for the trigger). */
  times: string[];
  startsOn: string | null;
  endsOn: string | null;
  isActive: boolean;
  notes: string | null;
  inventory: IvfInventory | null;
}

/** The cycle's latest trigger medicine («تزریق تریگر»). */
export interface IvfTrigger {
  medId: number;
  name: string;
  dose: string | null;
  unit: string | null;
  triggerAt: string;
  taken: boolean;
}

export interface IvfSiteUse {
  site: string;
  date: string;
  slot: string;
}

export interface IvfSites {
  /** Active `ivf_injection_sites` codes in rotation order. */
  codes: string[];
  /** The most recent logged site (across cycles). */
  last: IvfSiteUse | null;
  /** Never used yet in rotation order, else the least recently used. */
  suggested: string | null;
}

/** `GET /ivf/meds` (and every meds / dose write's answer). */
export interface IvfMedsView {
  cycle: IvfCycle | null;
  trigger: IvfTrigger | null;
  today: IvfDoseDay;
  tomorrow: IvfDoseDay;
  sites: IvfSites;
  meds: IvfMed[];
}

/** `POST /ivf/meds`, `PUT /ivf/meds/{id}` body (camelCase; the API maps it). */
export interface IvfMedInput {
  name: string;
  role: IvfRole;
  route: IvfRoute;
  dose: string | null;
  unit: string | null;
  /** 1–4 `HH:MM`, null for the trigger. */
  times: string[] | null;
  /** `Y-m-d H:i`, the trigger only. */
  triggerAt: string | null;
  startsOn: string | null;
  endsOn: string | null;
  notes: string | null;
  stockUnits: number | null;
  stockUnit: IvfStockUnit | null;
  dosesPerUnit: number | null;
}

/** Catalog `ivf_injection_sites` row (admin-editable, request locale). */
export interface IvfInjectionSite {
  code: string;
  title: string | null;
  /** «حداقل ۵ سانت دور از ناف…». */
  body: string | null;
  region: 'abdomen' | 'thigh' | 'arm' | null;
  side: 'right' | 'left' | null;
}

/** Catalog `ivf_med_presets` row — «افزودن دارو از روی نسخه» pre-fill (classes, no brands). */
export interface IvfMedPreset {
  code: string;
  title: string | null;
  role: IvfRole;
  route: IvfRoute;
  unit: string | null;
  times: string[];
  stockUnit: IvfStockUnit | null;
}

/** Catalog `ivf_guidance` row (`trigger_timing`, `site_rotation`, …). */
export interface IvfGuidance {
  code: string;
  title: string | null;
  body: string | null;
}

/** Roles a stage change past stimulation offers to stop (CB-IVF-06b) [needs clinical review]. */
export const IVF_STIMULATION_ROLES: readonly IvfRole[] = ['stimulation', 'suppression'];
