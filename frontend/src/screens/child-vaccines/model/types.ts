import type { VaccineSummary, VaccineVisit } from '@/entities/child';

/** One dose of a visit (`doses[]` of GET /children/{id}/vaccines). */
export interface VaccineDose {
  code: string;
  title: string;
  protectsAgainst: string | null;
  status: string;
  statusLabel: string | null;
  /** `YYYY-MM-DD` when recorded as given. */
  givenOn: string | null;
  note: string | null;
}

/** A visit of the schedule with its doses. */
export interface VaccineVisitDetail extends VaccineVisit {
  doses: VaccineDose[];
}

/** GET /children/{id}/vaccines (also the body of every mark / unmark). */
export interface VaccineSchedule {
  summary: VaccineSummary;
  visits: VaccineVisitDetail[];
  reminderLabel: string | null;
  note: string | null;
}

/** What the «تزریق شد» sheet records: a whole visit or one dose. */
export type MarkTarget =
  | { kind: 'visit'; visit: VaccineVisitDetail }
  | { kind: 'dose'; visit: VaccineVisitDetail; dose: VaccineDose };
