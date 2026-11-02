import { describe, expect, it, vi } from 'vitest';

const redirect = vi.hoisted(() => vi.fn());
vi.mock('next/navigation', () => ({ redirect }));

import PregnancyAlertsRoute from './page';

describe('/pregnancy/alerts', () => {
  it('redirects the old URL to the generic /alerts inbox, keeping the locale', async () => {
    await PregnancyAlertsRoute({ params: Promise.resolve({ locale: 'fa' }) });
    expect(redirect).toHaveBeenCalledWith('/fa/alerts');
  });
});
