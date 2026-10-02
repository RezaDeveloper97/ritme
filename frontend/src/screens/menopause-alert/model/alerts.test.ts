import { describe, expect, it } from 'vitest';

import { alertLook, menoAlertsSchema, primaryAlert, tellEarlyAlerts } from './alerts';

const payload = {
  group: 'meno_alerts',
  items: [
    {
      code: 'postmenopausal_bleeding',
      title: 'خونریزی بعد از یائسگی',
      body: 'بدنه',
      meta: { severity: 'urgent', primary: true, stages: ['meno', 'post'], cta: 'این مورد را به پزشک بگو' },
    },
    { code: 'heavy_perimenopause_bleeding', title: 'زیاد', body: 'بیش از ۷ روز', meta: { severity: 'caution' } },
    { code: 'chest_pain_palpitations', title: 'قفسه سینه', body: '۱۱۵', meta: { severity: 'urgent', hotline: '115' } },
    { code: 'admin_added', title: 'جدید', body: null, meta: null },
    { code: 'blank', title: null, body: null, meta: null },
    { title: 'no code' },
  ],
};

describe('menoAlertsSchema', () => {
  const alerts = menoAlertsSchema.parse(payload);

  it('parses meta and drops malformed items', () => {
    expect(alerts.map((a) => a.code)).toEqual([
      'postmenopausal_bleeding',
      'heavy_perimenopause_bleeding',
      'chest_pain_palpitations',
      'admin_added',
      'blank',
    ]);
    expect(alerts[0]).toMatchObject({ primary: true, severity: 'urgent', cta: 'این مورد را به پزشک بگو' });
    expect(alerts[2].hotline).toBe('115');
    expect(alerts[3]).toMatchObject({ primary: false, severity: 'caution', hotline: null, cta: null });
  });

  it('rejects a non-numeric hotline', () => {
    const [a] = menoAlertsSchema.parse({ items: [{ code: 'x', title: 't', meta: { hotline: 'tel:115' } }] });
    expect(a.hotline).toBeNull();
  });

  it('survives a missing payload', () => {
    expect(menoAlertsSchema.parse(undefined)).toEqual([]);
  });

  it('splits the primary card from the tell-early list', () => {
    expect(primaryAlert(alerts)?.code).toBe('postmenopausal_bleeding');
    expect(tellEarlyAlerts(alerts).map((a) => a.code)).toEqual([
      'heavy_perimenopause_bleeding',
      'chest_pain_palpitations',
      'admin_added',
    ]);
  });

  it('falls back to the bleeding code when no item is flagged primary', () => {
    const items = menoAlertsSchema.parse({ items: [{ code: 'a', title: 'a' }, { code: 'postmenopausal_bleeding', title: 'b' }] });
    expect(primaryAlert(items)?.code).toBe('postmenopausal_bleeding');
    expect(primaryAlert([])).toBeNull();
  });
});

describe('alertLook', () => {
  it('has a look per seeded code and a severity fallback', () => {
    expect(alertLook({ code: 'chest_pain_palpitations', severity: 'urgent' })).toEqual({ icon: 'zap', tone: 'danger' });
    expect(alertLook({ code: 'new', severity: 'urgent' }).tone).toBe('danger');
    expect(alertLook({ code: 'new', severity: 'caution' })).toEqual({ icon: 'warning', tone: 'warm' });
  });
});
