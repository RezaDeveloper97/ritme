/*
 * The companion's side of «همدم» (B-N4-05 on B-N4-02/03): the links a companion
 * account holds and the companion panel home (`GET /api/v1/companion/home`).
 * Camel-cased from the payloads; every section view is `null` without a grant.
 */
import type { CompanionGrants, CompanionSection, CompanionType } from './types';

/** Length of a companion invite code (backend `internal/companion/code.go`). */
export const COMPANION_CODE_LENGTH = 6;

/** One link as the companion sees it (`POST /companions/accept`, `GET /companions/links`). */
export interface ViewerLink {
  id: number;
  type: CompanionType;
  owner: { id: number; name: string | null };
  acceptedAt: string | null;
  grants: CompanionGrants;
  /** Sections the owner lets the companion record for (edit grants). */
  canRecordFor: CompanionSection[];
  family: { id: number; sharedChildIds: number[] } | null;
}

/** Her cycle today, minimised (`companion/shared` cycle view). */
export interface SharedCycleView {
  date: string;
  hasData: boolean;
  cycleDay: number | null;
  cycleLength: number | null;
  mainPhase: string | null;
  daysToPeriod: number | null;
  daysLate: number;
  predictedNextPeriodStart: string | null;
}

export interface SharedPregnancyView {
  isActive: boolean;
  currentWeek: number | null;
  trimester: number | null;
}

export interface SharedMedication {
  id: number;
  title: string;
  /** «HH:MM» dose times. */
  times: string[];
}

export interface SharedAppointment {
  id: number;
  title: string;
  /** ISO date-time with offset. */
  scheduledAt: string | null;
  daysUntil: number | null;
}

/** Tip phase of `companion/home` tips.go. */
export const COMPANION_PHASES = ['menstrual', 'follicular', 'fertile', 'luteal', 'pregnancy', 'general'] as const;
export type CompanionPhase = (typeof COMPANION_PHASES)[number];

export interface CompanionTip {
  key: string;
  title: string;
  body: string | null;
}

/** One partner card set on the companion home. */
export interface CompanionPartner {
  link: ViewerLink;
  partnerName: string | null;
  cycle: SharedCycleView | null;
  pregnancy: SharedPregnancyView | null;
  /** Number of days with logged symptoms in the last week; null = not shared. */
  symptomDays: number | null;
  meds: SharedMedication[] | null;
  appointments: SharedAppointment[] | null;
  phase: CompanionPhase;
  note: string | null;
  tips: CompanionTip[];
  /** The shared child card (B-N5); always null for now. */
  child: unknown | null;
}

export interface CompanionArticle {
  id: number;
  slug: string;
  title: string | null;
  readTimeMinutes: number | null;
  imageUrl: string | null;
}

export interface CompanionHome {
  viewer: { id: number; name: string | null };
  hasPartners: boolean;
  partners: CompanionPartner[];
  /** Server copy for the no-link state; its action is always the code entry. */
  emptyState: { title: string; body: string; actionLabel: string } | null;
  articles: CompanionArticle[];
}
