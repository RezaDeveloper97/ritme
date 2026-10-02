import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

// The screen's only outside inputs: the translator, the locale and the Link. Keys come back as text.
vi.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string) => key,
}));
vi.mock('@/shared/i18n', () => ({
  Link: ({ href, children, className }: { href: string; children: unknown; className?: string }) =>
    createElement('a', { href, className }, children as never),
}));

const { SafetyScreen } = await import('./SafetyScreen');

type Props = Parameters<typeof SafetyScreen>[0];
const render = (props: Partial<Props>) =>
  renderToStaticMarkup(
    createElement(SafetyScreen, { safety: null, saved: true, retrying: false, t: ((k: string) => k) as never, ...props }),
  );

describe('EPDS urgent safety screen', () => {
  it('renders the call buttons before any copy, from the server actions', () => {
    const html = render({
      safety: {
        level: 'urgent',
        title: 'Please talk to someone',
        body: 'You are not alone',
        actions: [
          { type: 'call', number: '115', label: 'Emergency 115' },
          { type: 'call', number: '123', label: 'Social 123' },
          { type: 'call', number: '1480', label: 'Counsel 1480' },
        ],
      },
    });
    const firstCall = html.indexOf('href="tel:115"');
    expect(firstCall).toBeGreaterThan(-1);
    expect(firstCall).toBeLessThan(html.indexOf('Please talk to someone'));
    expect(html.indexOf('tel:115')).toBeLessThan(html.indexOf('tel:123'));
    expect(html.indexOf('tel:123')).toBeLessThan(html.indexOf('tel:1480'));
    expect(html).toContain('Emergency 115');
    expect(html).toContain('role="alert"');
  });

  it('without any server message (request failed) still offers 115 / 123 / 1480 and the fallback copy', () => {
    const html = render({ safety: null, saved: false, onRetry: () => undefined });
    for (const n of ['115', '123', '1480']) expect(html).toContain(`href="tel:${n}"`);
    expect(html).toContain('mood.safety.title');
    expect(html).toContain('mood.safety.notSaved');
    expect(html).toContain('mood.safety.retrySave');
  });

  it('a saved urgent result shows no retry', () => {
    const html = render({ saved: true });
    expect(html).not.toContain('mood.safety.retrySave');
    expect(html).toContain('href="/postpartum"');
  });
});
