import { describe, expect, it } from 'vitest';

import { toCustomCheckupBody } from './body';

describe('toCustomCheckupBody', () => {
  it('maps the custom-checkup form to the API body', () => {
    expect(
      toCustomCheckupBody({
        title: '  چکاپ چشم ',
        intervalMonths: 12,
        performedBy: 'doctor',
        note: '',
        lastDoneOn: '2026-01-10',
      }),
    ).toEqual({
      title: 'چکاپ چشم',
      interval_months: 12,
      performed_by: 'doctor',
      note: null,
      last_done_on: '2026-01-10',
    });
  });

  it('keeps a PUT partial', () => {
    expect(toCustomCheckupBody({ intervalMonths: 6 })).toEqual({ interval_months: 6 });
    expect(toCustomCheckupBody({ lastDoneOn: null })).toEqual({ last_done_on: null });
    expect(toCustomCheckupBody({})).toEqual({});
  });
});
