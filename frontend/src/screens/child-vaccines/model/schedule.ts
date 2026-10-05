import type { VaccineDose, VaccineVisitDetail } from './types';

/** Visits that get the «ثبت نوبت» / «تزریق شد» actions: not done and due now or next. */
export function isActionable(visit: VaccineVisitDetail, nextCode: string | null): boolean {
  if (visit.status === 'done') return false;
  return visit.code === nextCode || visit.status === 'overdue' || visit.status === 'due' || visit.status === 'soon';
}

/**
 * Default «تاریخ تزریق» for a visit: its due date when that has passed (a late
 * entry of an old shot), else today — never before the birth.
 */
export function defaultGivenOn(visit: Pick<VaccineVisitDetail, 'dueDate'>, today: string, birthDate: string): string {
  const day = visit.dueDate < today ? visit.dueDate : today;
  return day < birthDate ? birthDate : day;
}

/** The given doses in schedule order, for the vaccine card tab. */
export function givenDoses(visits: readonly VaccineVisitDetail[]): Array<{ visit: VaccineVisitDetail; dose: VaccineDose }> {
  return visits.flatMap((visit) => visit.doses.filter((d) => d.givenOn !== null).map((dose) => ({ visit, dose })));
}

/** Doses with a note (newest shot first), for the notes tab. */
export function notedDoses(visits: readonly VaccineVisitDetail[]): Array<{ visit: VaccineVisitDetail; dose: VaccineDose }> {
  return givenDoses(visits)
    .filter(({ dose }) => !!dose.note)
    .sort((a, b) => (b.dose.givenOn ?? '').localeCompare(a.dose.givenOn ?? ''));
}
