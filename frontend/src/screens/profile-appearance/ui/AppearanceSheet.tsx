'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';

import { THEME_PREFERENCES, useThemeStore, type ThemePreference } from '@/shared/theme';
import { Icon, type IconName } from '@/shared/ui';

const ICONS: Record<ThemePreference, IconName> = {
  system: 'contrast',
  light: 'sun',
  dark: 'moon',
};

/**
 * Appearance picker, opened as a sheet from the profile screen.
 *
 * Three choices, not a two-way switch: "Auto" has to stay reachable, because a
 * user who never opens this screen is on it, and a user who tried dark and
 * wants the phone to decide again has nowhere else to go back to.
 *
 * The whole sheet is registered with `ssr: false` — the preference lives in
 * localStorage, so rendering the selected row on the server would be a guess
 * that the client then contradicts (a hydration mismatch).
 */
export function AppearanceSheet() {
  const t = useTranslations('profile');
  const theme = useThemeStore((s) => s.theme);
  const resolved = useThemeStore((s) => s.resolved);
  const setTheme = useThemeStore((s) => s.setTheme);

  return (
    <div className="lang-list">
      <p className="lang-hint">{t('appearance.hint')}</p>

      {THEME_PREFERENCES.map((preference) => {
        const active = preference === theme;

        return (
          <button
            key={preference}
            type="button"
            className={clsx('lang-row', active && 'is-active')}
            aria-current={active}
            onClick={() => setTheme(preference)}
          >
            <span className="appr-icon">
              <Icon name={ICONS[preference]} size={19} />
            </span>
            <span className="lang-row-body">
              <span className="lang-row-name">{t(`appearance.${preference}`)}</span>
              {preference === 'system' ? (
                <span className="lang-row-code">
                  {t(`appearance.following.${resolved}`)}
                </span>
              ) : null}
            </span>
            {active ? <Icon name="check" size={18} className="lang-row-check" /> : null}
          </button>
        );
      })}
    </div>
  );
}
