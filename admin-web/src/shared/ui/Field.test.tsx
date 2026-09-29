import { NextIntlClientProvider } from 'next-intl';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import en from '../../../messages/en.json';
import fa from '../../../messages/fa.json';
import { TextInput } from './Field';

function render(locale: 'fa' | 'en', props: Parameters<typeof TextInput>[0]) {
  return renderToStaticMarkup(
    <NextIntlClientProvider locale={locale} messages={locale === 'fa' ? fa : en} timeZone="Asia/Tehran">
      <TextInput label="ترتیب" onChange={() => undefined} {...props} />
    </NextIntlClientProvider>,
  );
}

const input = (html: string) => html.match(/<input[^>]*>/)?.[0] ?? '';

describe('TextInput type="number"', () => {
  it('shows the value in fa digits on a numeric-keyboard text input', () => {
    const tag = input(render('fa', { type: 'number', value: '12', min: 1, max: 42 }));
    expect(tag).toContain('type="text"');
    expect(tag).toContain('inputMode="numeric"');
    expect(tag).toContain('value="۱۲"');
    expect(tag).not.toContain('min=');
  });

  it('keeps ASCII digits in en and a decimal keyboard for a fractional step', () => {
    const tag = input(render('en', { type: 'number', value: '1.5', step: '0.1' }));
    expect(tag).toContain('value="1.5"');
    expect(tag).toContain('inputMode="decimal"');
  });

  it('leaves other inputs alone', () => {
    const tag = input(render('fa', { value: '12' }));
    expect(tag).toContain('value="12"');
    expect(tag).not.toContain('inputMode');
  });
});
