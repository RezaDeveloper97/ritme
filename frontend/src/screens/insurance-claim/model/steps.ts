import type { ClaimEvent, ClaimEventPayload, ClaimStatus } from '@/entities/insurance';
import type { StepState, Tone } from '@/shared/ui';

/** A claim stops moving at these steps (she can still appeal a rejection or a short payment). */
const TERMINAL: readonly ClaimStatus[] = ['paid', 'rejected'];

export interface ClaimStep {
  id: string;
  status: ClaimStatus;
  state: StepState;
  tone: Tone;
  occurredOn: string | null;
  deadlineOn: string | null;
  amount: number | null;
  note: string;
}

/**
 * Her timeline as board steps (nbl_Ins_ClaimDetail): every step she recorded is done except the latest, which is the
 * current one while the claim is still moving; «waiting on you» (missing documents) is the bloom accent. While the
 * claim is not yet paid or rejected a last `paid` step stays ahead as the todo marker.
 */
export function buildSteps(events: readonly ClaimEvent[]): ClaimStep[] {
  if (events.length === 0) return [];
  const last = events[events.length - 1];
  const moving = !TERMINAL.includes(last.status);
  const steps: ClaimStep[] = events.map((e, i) => {
    const isLast = i === events.length - 1;
    const current = isLast && moving;
    return {
      id: `e${e.id}`,
      status: e.status,
      state: current ? 'current' : 'done',
      tone: current && e.status === 'docs_requested' ? 'bloom' : 'brand',
      occurredOn: e.occurredOn,
      deadlineOn: e.deadlineOn,
      amount: e.amount,
      note: e.note,
    };
  });
  if (moving && !events.some((e) => e.status === 'paid')) {
    steps.push({ id: 'todo-paid', status: 'paid', state: 'todo', tone: 'brand', occurredOn: null, deadlineOn: null, amount: null, note: '' });
  }
  return steps;
}

/** The statuses she can record next: anything but the current one. */
export function nextStatuses(current: ClaimStatus): ClaimStatus[] {
  return (['submitted', 'reviewing', 'docs_requested', 'paid', 'rejected', 'appealed'] as const).filter((s) => s !== current);
}

export interface StepDraft {
  status: ClaimStatus;
  /** `Y-m-d`. */
  occurredOn: string;
  amount: number | undefined;
  note: string;
}

export function stepProblem(d: StepDraft): 'amount' | null {
  return d.status === 'paid' && (!d.amount || d.amount < 1) ? 'amount' : null;
}

export function stepToPayload(d: StepDraft): ClaimEventPayload {
  const note = d.note.trim();
  return {
    status: d.status,
    occurred_on: d.occurredOn,
    amount: d.status === 'paid' ? (d.amount ?? null) : null,
    note: note === '' ? null : note,
  };
}
