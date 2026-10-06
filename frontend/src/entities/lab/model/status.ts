import type { LabStage, LabStatus } from './types';

/*
 * The lab status state machine as the screens see it (B-N6-06 `views.go` stage()):
 *
 *   queued → extracting (reading <30 %, then extracting) → needs_review (review)
 *          → [user verifies] → interpreting (explaining) → ready (done)
 *   any → failed
 *
 * The processing screen polls `/labs/{id}/status` while the lab is busy and
 * hands over to verify (needs_review), result (ready) or its failed state.
 */

/** The server is working on it — keep polling. */
export function isBusy(status: LabStatus): boolean {
  return status === 'queued' || status === 'extracting' || status === 'interpreting';
}

/** Where a lab with this status lives: processing, verify, result — or the failed state of processing. */
export type LabView = 'processing' | 'verify' | 'result' | 'failed';

export function viewOf(status: LabStatus): LabView {
  if (isBusy(status)) return 'processing';
  if (status === 'needs_review') return 'verify';
  if (status === 'ready') return 'result';
  return 'failed';
}

/** The route of a lab in its current state. */
export function labHref(id: number, status: LabStatus): string {
  switch (viewOf(status)) {
    case 'verify':
      return `/labs/${id}/verify`;
    case 'result':
      return `/labs/${id}`;
    default:
      return `/labs/${id}/processing`;
  }
}

/** Polls are fast at first (the fake provider finishes in a second or two), then back off. */
export const POLL_FAST_MS = 1500;
export const POLL_SLOW_MS = 4000;
export const POLL_FAST_COUNT = 20;
/** After this many polls (~10 minutes) the screen stops polling and offers a manual refresh. */
export const POLL_MAX_COUNT = 160;

/** Delay before the next status poll, or `false` to stop (not busy any more, or gave up). */
export function nextPollDelay(status: LabStatus | undefined, polls: number): number | false {
  if (status !== undefined && !isBusy(status)) return false;
  if (polls >= POLL_MAX_COUNT) return false;
  return polls < POLL_FAST_COUNT ? POLL_FAST_MS : POLL_SLOW_MS;
}

/** The four processing steps of nbl_Lab_Processing. */
export const PROCESS_STEPS = ['read', 'extract', 'compare', 'explain'] as const;
export type ProcessStep = (typeof PROCESS_STEPS)[number];
export type StepProgress = 'done' | 'current' | 'todo';

/** Which of the four steps are done / current for a stage. */
export function stepStates(stage: LabStage): Record<ProcessStep, StepProgress> {
  // index of the current step; 4 = all done
  const current: Record<LabStage, number> = {
    queued: 0,
    reading: 0,
    extracting: 1,
    review: 2,
    explaining: 3,
    done: 4,
    failed: -1,
  };
  const at = current[stage];
  const out = {} as Record<ProcessStep, StepProgress>;
  PROCESS_STEPS.forEach((step, i) => {
    out[step] = at < 0 ? 'todo' : i < at ? 'done' : i === at ? 'current' : 'todo';
  });
  return out;
}

/** 0–1 for the processing ring: the server's progress inside the extraction, a fixed share for the explanation. */
export function ringValue(stage: LabStage, progress: number): number {
  const p = Math.max(0, Math.min(100, Number.isFinite(progress) ? progress : 0)) / 100;
  switch (stage) {
    case 'queued':
      return 0.04;
    case 'reading':
    case 'extracting':
      return Math.max(0.08, p * 0.7);
    case 'review':
      return 0.75;
    case 'explaining':
      return 0.88;
    case 'done':
      return 1;
    default:
      return 0;
  }
}
