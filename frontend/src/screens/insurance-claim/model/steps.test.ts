import { describe, expect, it } from 'vitest';

import type { ClaimEvent } from '@/entities/insurance';

import { buildSteps, nextStatuses, stepProblem, stepToPayload } from './steps';

const ev = (id: number, status: ClaimEvent['status'], note = ''): ClaimEvent => ({ id, status, occurredOn: '2026-10-0' + id, deadlineOn: null, amount: null, note });

describe('claim steps', () => {
  it('marks the latest moving step current and keeps payment ahead', () => {
    const steps = buildSteps([ev(1, 'submitted'), ev(2, 'reviewing'), ev(3, 'docs_requested')]);
    expect(steps.map((s) => s.state)).toEqual(['done', 'done', 'current', 'todo']);
    expect(steps[2].tone).toBe('bloom');
    expect(steps[3].status).toBe('paid');
  });

  it('has no todo step once paid or rejected', () => {
    expect(buildSteps([ev(1, 'submitted'), ev(2, 'paid')]).map((s) => s.state)).toEqual(['done', 'done']);
    expect(buildSteps([ev(1, 'submitted'), ev(2, 'rejected')]).map((s) => s.state)).toEqual(['done', 'done']);
  });

  it('an appeal after a rejection is moving again', () => {
    const steps = buildSteps([ev(1, 'submitted'), ev(2, 'rejected'), ev(3, 'appealed')]);
    expect(steps.map((s) => s.state)).toEqual(['done', 'done', 'current', 'todo']);
  });

  it('offers every status but the current', () => {
    expect(nextStatuses('reviewing')).not.toContain('reviewing');
    expect(nextStatuses('reviewing')).toHaveLength(5);
  });

  it('a paid step needs an amount and sends it only then', () => {
    expect(stepProblem({ status: 'paid', occurredOn: '2026-10-05', amount: undefined, note: '' })).toBe('amount');
    expect(stepProblem({ status: 'reviewing', occurredOn: '2026-10-05', amount: undefined, note: '' })).toBeNull();
    expect(stepToPayload({ status: 'reviewing', occurredOn: '2026-10-05', amount: 5, note: ' hi ' })).toEqual({
      status: 'reviewing',
      occurred_on: '2026-10-05',
      amount: null,
      note: 'hi',
    });
  });
});
