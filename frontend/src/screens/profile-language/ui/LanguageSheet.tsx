'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';

import { useSwitchLocale } from '@/features/switch-locale';
import { Icon } from '@/shared/ui';

/**
 * The app's language picker, opened as a sheet from the profile screen.
 *
 * The list is whatever the backend currently ships (CLAUDE.md §6) — a language
 * an admin adds shows up here without a frontend release, so this renders the
 * fetched languages rather than a hardcoded pair. Each row is labelled in its
 * own language and laid out in that language's own direction, so a reader who
 * cannot read the current one can still find theirs.
 */
export function LanguageSheet() {
  const t = useTranslations('profile');
  const { locale, languages, switchLocale, isPending } = useSwitchLocale();

  return (
    <div className="lang-list">
      <p className="lang-hint">{t('language.hint')}</p>

      {languages.map((language) => {
        const active = language.code === locale;

        return (
          <button
            key={language.code}
            type="button"
            className={clsx('lang-row', active && 'is-active')}
            dir={language.direction}
            disabled={isPending || active}
            aria-current={active}
            onClick={() => switchLocale(language.code)}
          >
            <span className="lang-row-body">
              <span className="lang-row-name">{language.name}</span>
              <span className="lang-row-code">{language.englishName}</span>
            </span>
            {active ? <Icon name="check" size={18} className="lang-row-check" /> : null}
          </button>
        );
      })}
    </div>
  );
}
