import { NextIntlClientProvider } from 'next-intl';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import en from '../../../../messages/en.json';
import fa from '../../../../messages/fa.json';

vi.mock('@/features/auth', () => ({ useCan: () => true }));
vi.mock('@/entities/segment', () => ({
  useSegments: () => ({ isLoading: false, data: { items: [{ id: 7, name: 'Active TTC', filters: {} }, { id: 8, name: 'Other', filters: {} }] } }),
}));

import { SegmentPicker } from './SegmentPicker';

function render(locale: 'fa' | 'en', ids: string[]) {
  return renderToStaticMarkup(
    <QueryClientProvider client={new QueryClient()}>
      <NextIntlClientProvider locale={locale} messages={locale === 'fa' ? fa : en} timeZone="Asia/Tehran">
        <SegmentPicker ids={ids} onChange={() => undefined} />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  );
}

describe('SegmentPicker', () => {
  it('shows the chosen segment by name and offers only the others', () => {
    const html = render('en', ['7']);
    expect(html).toContain('Active TTC');
    expect(html).toContain('<option value="8">Other</option>');
    expect(html).not.toContain('<option value="7">');
  });

  it('flags a deleted segment id instead of crashing', () => {
    expect(render('en', ['99'])).toContain('Deleted segment (#99)');
    expect(render('fa', ['99'])).toContain('سگمنت حذف‌شده');
  });
});
