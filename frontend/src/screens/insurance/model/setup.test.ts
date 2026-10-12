import { describe, expect, it } from 'vitest';

import { draftFromPolicy, draftProblems, draftToPayload, emptyDraft } from './setup';

describe('policy draft', () => {
  it('needs an insurer, ordered dates and named members', () => {
    const d = emptyDraft('Me');
    expect(draftProblems(d)).toEqual({ insurer: true });
    d.insurerId = 3;
    d.validFrom = '2026-03-21';
    d.validTo = '2026-03-01';
    d.members.push({ kind: 'child', name: ' ' });
    expect(draftProblems(d)).toEqual({ dates: true, member: true });
  });

  it('builds the PUT body: millions to toman, months to days, number only when typed', () => {
    const d = emptyDraft('Me');
    d.custom = true;
    d.insurerName = ' Some insurer ';
    d.coverages.delivery = { capMillions: 40, deductiblePct: undefined, waitingMonths: 9 };
    d.coverages.paraclinic = { capMillions: 1.5, deductiblePct: 30, waitingMonths: undefined };
    const body = draftToPayload(d);
    expect(body).toMatchObject({ insurer_id: null, insurer_name: 'Some insurer', plan_name: null });
    expect(body).not.toHaveProperty('policy_number');
    expect(body.coverages).toEqual([
      { category: 'delivery', status: 'covered', cap_amount: 40_000_000, deductible_pct: null, waiting_days: 270 },
      { category: 'paraclinic', status: 'covered', cap_amount: 1_500_000, deductible_pct: 30, waiting_days: null },
    ]);
    d.policyNumber = '12345-4821';
    expect(draftToPayload(d).policy_number).toBe('12345-4821');
  });

  it('round-trips an existing policy and keeps member ids', () => {
    const d = draftFromPolicy(
      {
        id: 1,
        status: 'current',
        readOnly: false,
        canRenew: true,
        insurerId: 7,
        insurerName: 'Catalog',
        planName: 'Gold',
        numberLast4: '4821',
        validFrom: '2026-03-21',
        validTo: '2027-03-20',
        claimDeadlineDays: 15,
        ibanLast4: null,
        notCoveredNote: null,
        members: [{ id: 9, kind: 'self', childId: null, name: 'Sara' }],
        coverages: [
          { category: 'inpatient', status: 'covered', capAmount: 300_000_000, deductiblePct: 10, waitingDays: null, note: null, used: 0, remaining: 300_000_000, waitingEndsOn: null, waitingOver: true },
        ],
      },
      'Me',
    );
    expect(d).toMatchObject({ insurerId: 7, custom: false, insurerName: '', planName: 'Gold', claimDeadlineDays: 15 });
    expect(d.coverages.inpatient).toEqual({ capMillions: 300, deductiblePct: 10, waitingMonths: undefined });
    const body = draftToPayload(d);
    expect(body).toMatchObject({ insurer_id: 7, insurer_name: null });
    expect(body.members).toEqual([{ id: 9, kind: 'self', name: 'Sara' }]);
  });
});
