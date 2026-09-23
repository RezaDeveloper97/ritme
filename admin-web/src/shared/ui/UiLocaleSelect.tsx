'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { setUiLocaleCookie, uiLocaleOptions } from '@/shared/i18n';

import { Icon } from './Icon';

/** Admin UI language picker (bundled UI translations, not content languages). */
export function UiLocaleSelect() {
  const t = useTranslations('shell');
  const locale = useLocale();
  const router = useRouter();
  const locales = uiLocaleOptions();
  if (locales.length < 2) return null;
  return (
    <label className="flex items-center gap-1.5 text-muted">
      <Icon name="globe" size={16} className="hidden sm:block" />
      <span className="sr-only">{t('language')}</span>
      <select
        className="input min-h-8 w-auto py-0.5 text-[13px]"
        value={locale}
        onChange={(e) => {
          setUiLocaleCookie(e.target.value);
          router.refresh();
        }}
      >
        {locales.map((l) => (
          <option key={l.code} value={l.code}>
            {l.name}
          </option>
        ))}
      </select>
    </label>
  );
}
