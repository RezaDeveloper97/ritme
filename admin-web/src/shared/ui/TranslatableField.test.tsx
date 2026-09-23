import { NextIntlClientProvider } from 'next-intl';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import type { ContentLanguage } from '@/shared/i18n';

import fa from '../../../messages/fa.json';
import { checkImageFile } from './ImageUpload';
import { TranslatableInputs } from './TranslatableField';

// Rows as GET /api/v1/languages returns them — including a third, admin-added one.
const languages: ContentLanguage[] = [
  { code: 'en', name: 'English', english_name: 'English', direction: 'ltr', is_default: false },
  { code: 'fa', name: 'فارسی', english_name: 'Persian', direction: 'rtl', is_default: true },
  { code: 'ar', name: 'العربية', english_name: 'Arabic', direction: 'rtl', is_default: false },
];

function render(kind: 'text' | 'textarea', required = true, errors?: Record<string, string[]>) {
  return renderToStaticMarkup(
    <NextIntlClientProvider locale="fa" messages={fa} timeZone="Asia/Tehran">
      <TranslatableInputs
        name="title"
        label="عنوان"
        value={{ fa: 'سلام' }}
        onChange={() => undefined}
        kind={kind}
        required={required}
        errors={errors}
        languages={languages}
      />
    </NextIntlClientProvider>,
  );
}

const controls = (html: string, tag: 'input' | 'textarea') => html.match(new RegExp(`<${tag}[^>]*>`, 'g')) ?? [];

describe('TranslatableInputs', () => {
  it('renders one input per active language, default first', () => {
    const inputs = controls(render('text'), 'input');
    expect(inputs).toHaveLength(3);
    expect(inputs.map((i) => /name="title\[(\w+)\]"/.exec(i)?.[1])).toEqual(['fa', 'en', 'ar']);
  });

  it('marks only the default language as required', () => {
    const inputs = controls(render('text'), 'input');
    expect(inputs.filter((i) => / required=""/.test(i))).toHaveLength(1);
    expect(inputs[0]).toMatch(/name="title\[fa\]"[^>]* required=""|required=""[^>]*name="title\[fa\]"/);
  });

  it('marks nothing required when the field is optional', () => {
    expect(controls(render('textarea', false), 'textarea').filter((i) => / required=""/.test(i))).toHaveLength(0);
  });

  it("uses each language's own direction", () => {
    const dirs = controls(render('text'), 'input').map((i) => /dir="(\w+)"/.exec(i)?.[1]);
    expect(dirs).toEqual(['rtl', 'ltr', 'rtl']);
  });

  it('shows the API error under its language (`title.en`)', () => {
    const html = render('text', true, { 'title.en': ['too long'] });
    expect(html).toContain('too long');
    expect(controls(html, 'input')[1]).toContain('aria-invalid="true"');
  });
});

describe('checkImageFile', () => {
  it('checks type and size', () => {
    const types = ['image/png', 'image/webp'];
    expect(checkImageFile({ type: 'image/png', size: 1024 }, types, 4)).toBeNull();
    expect(checkImageFile({ type: 'image/gif', size: 1024 }, types, 4)).toBe('badType');
    expect(checkImageFile({ type: 'image/png', size: 5 * 1024 * 1024 }, types, 4)).toBe('tooLarge');
  });
});
