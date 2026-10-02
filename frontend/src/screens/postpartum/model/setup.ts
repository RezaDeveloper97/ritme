import { MAX_BIRTH_AGE_DAYS } from '@/entities/postpartum';
import { diffInDays, fromApiDate } from '@/shared/lib/date';

export type BirthDateProblem = 'future' | 'tooOld';

/** Client mirror of the activation rule (birth in the last 365 days, not in the future); the API decides. */
export function birthDateProblem(birthDate: string, today: string): BirthDateProblem | null {
  const diff = diffInDays(fromApiDate(today), fromApiDate(birthDate));
  if (diff < 0) return 'future';
  if (diff > MAX_BIRTH_AGE_DAYS) return 'tooOld';
  return null;
}
