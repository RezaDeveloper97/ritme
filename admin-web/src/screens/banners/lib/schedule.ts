/** Where a banner's display window stands at `now` (ISO bounds, either open). */
export type ScheduleState = 'live' | 'scheduled' | 'ended';

export function scheduleState(startsAt: string | null, endsAt: string | null, now: Date = new Date()): ScheduleState {
  const t = now.getTime();
  if (startsAt && new Date(startsAt).getTime() > t) return 'scheduled';
  if (endsAt && new Date(endsAt).getTime() < t) return 'ended';
  return 'live';
}
