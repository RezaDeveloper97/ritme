import { describe, expect, it } from 'vitest';

import { type ClaimDraft, type DraftDoc, draftProblems, draftToPayload, withDoc } from './draft';

const draft = (patch: Partial<ClaimDraft> = {}): ClaimDraft => ({
  memberId: 3,
  costType: 'paraclinic',
  centreName: '  Centre  ',
  serviceDate: '2026-10-02',
  amount: 1_800_000,
  docs: [],
  ...patch,
});
const doc = (id: number, label: DraftDoc['label']): DraftDoc => ({ id, label, title: null, kind: 'visit' });

describe('claim draft', () => {
  it('needs an amount and a date', () => {
    expect(draftProblems(draft())).toEqual([]);
    expect(draftProblems(draft({ amount: undefined }))).toEqual(['amount']);
    expect(draftProblems(draft({ amount: 0, serviceDate: '' }))).toEqual(['amount', 'date']);
  });

  it('keeps one document per named slot and never lists one twice', () => {
    let docs = withDoc([], doc(1, 'invoice'));
    docs = withDoc(docs, doc(2, 'invoice'));
    expect(docs.map((d) => d.id)).toEqual([2]);
    docs = withDoc(docs, doc(2, 'report'));
    expect(docs).toEqual([doc(2, 'report')]);
    docs = withDoc(withDoc(docs, doc(5, 'other')), doc(6, 'other'));
    expect(docs.map((d) => d.id)).toEqual([2, 5, 6]);
  });

  it('builds the API body with a trimmed centre and labelled documents', () => {
    const body = draftToPayload(draft({ docs: [doc(9, 'invoice')] }));
    expect(body).toMatchObject({ member_id: 3, cost_type: 'paraclinic', centre_name: 'Centre', service_date: '2026-10-02', amount: 1_800_000 });
    expect(body.documents).toEqual([{ id: 9, label: 'invoice' }]);
    expect(draftToPayload(draft({ centreName: ' ' })).centre_name).toBeNull();
  });
});
