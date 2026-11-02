import { NextIntlClientProvider } from 'next-intl';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import en from '../../../../messages/en.json';
import fa from '../../../../messages/fa.json';
import { ChainStatusCard } from './AuditLogScreen';

describe('ChainStatusCard', () => {
  it.each(['en', 'fa'] as const)('renders the verify button in %s', (locale) => {
    const html = renderToStaticMarkup(
      <QueryClientProvider client={new QueryClient()}>
        <NextIntlClientProvider locale={locale} messages={locale === 'fa' ? fa : en} timeZone="Asia/Tehran">
          <ChainStatusCard />
        </NextIntlClientProvider>
      </QueryClientProvider>,
    );
    expect(html).toContain(locale === 'fa' ? 'بررسی زنجیره' : 'Verify chain');
  });
});
