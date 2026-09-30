'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState, type CSSProperties, type KeyboardEvent } from 'react';

import { useDirection, useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  systemReducesMotion,
  TEXT_SCALES,
  useDisplayStore,
  useThemeStore,
  watchSystemMotion,
  type ResolvedTheme,
} from '@/shared/theme';
import { Checkbox, ListGroup, ListRow, ScreenHeader, SkyLayer, Switch } from '@/shared/ui';

const THEMES: readonly ResolvedTheme[] = ['dark', 'light'];

/** Miniature of a screen in the given theme (theme-stable colours, globals.css `.app-pv`). */
function ThemePreview({ theme }: { theme: ResolvedTheme }) {
  return (
    <span className={clsx('app-pv', `is-${theme}`)} aria-hidden>
      <span className="app-pv-card" />
      <span className="app-pv-line" />
      <span className="app-pv-line is-short" />
      <span className="app-pv-cta" />
    </span>
  );
}

/** The OS reduced-motion setting, live (false until mounted). */
function useSystemReducedMotion(): boolean {
  const [reduced, setReduced] = useState(false);
  useEffect(() => {
    setReduced(systemReducesMotion());
    return watchSystemMotion(() => setReduced(systemReducesMotion()));
  }, []);
  return reduced;
}

/**
 * Appearance (B-N1-10, `nbl_Me_Appearance` / `nbd_Me_Appearance`) at
 * `/profile/appearance`: theme dark/light + follow the phone, text size, and
 * reduced motion that overrides the OS. The artboard's «لرزش هنگام ثبت»
 * (haptics) row is native-only and hidden on web.
 *
 * Every value lives in localStorage, so the controls render their defaults
 * until mounted (same markup as the server) and then settle.
 */
export function AppearancePage() {
  const t = useTranslations('me');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const mounted = useMounted();
  const preference = useThemeStore((s) => s.preference);
  const resolved = useThemeStore((s) => s.theme);
  const setPreference = useThemeStore((s) => s.setPreference);
  const textScale = useDisplayStore((s) => s.textScale);
  const setTextScale = useDisplayStore((s) => s.setTextScale);
  const motion = useDisplayStore((s) => s.motion);
  const setMotion = useDisplayStore((s) => s.setMotion);
  const systemReduced = useSystemReducedMotion();

  const followSystem = mounted && preference === 'system';
  const checkedTheme: ResolvedTheme | null = mounted ? resolved : null;
  const rtl = useDirection() === 'rtl';
  const themeRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const tabStop = Math.max(0, THEMES.indexOf(checkedTheme ?? 'light'));
  // APG radiogroup: arrows move selection and focus together (two options → toggle).
  const onThemeKey = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    const keys = ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Home', 'End'];
    if (!keys.includes(event.key)) return;
    event.preventDefault();
    const forward = event.key === 'ArrowDown' || event.key === (rtl ? 'ArrowLeft' : 'ArrowRight');
    const next =
      event.key === 'Home' ? 0 : event.key === 'End' ? THEMES.length - 1
      : (index + (forward ? 1 : -1) + THEMES.length) % THEMES.length;
    setPreference(THEMES[next]);
    themeRefs.current[next]?.focus();
  };

  const scaleIndex = mounted ? textScale : 2;
  const percent = Math.round(TEXT_SCALES[scaleIndex] * 100);
  const fill = (scaleIndex / (TEXT_SCALES.length - 1)) * 100;
  const reduceOn = mounted && (motion === 'reduce' || systemReduced);

  return (
    <div className="view app-page">
      <SkyLayer />
      <div className="scroll app-scroll">
        <ScreenHeader title={t('appearance.title')} onBack={() => router.push('/profile')} backLabel={t('appearance.back')} />

        <section className="nb-card app-card" aria-labelledby="app-theme">
          <h2 id="app-theme" className="app-card-title">
            {t('appearance.theme')}
          </h2>
          <div role="radiogroup" aria-labelledby="app-theme" className="app-themes">
            {THEMES.map((theme, index) => (
              <button
                key={theme}
                ref={(el) => {
                  themeRefs.current[index] = el;
                }}
                type="button"
                role="radio"
                aria-checked={checkedTheme === theme}
                tabIndex={index === tabStop ? 0 : -1}
                className="app-theme"
                onClick={() => setPreference(theme)}
                onKeyDown={(event) => onThemeKey(event, index)}
              >
                <ThemePreview theme={theme} />
                <span className="app-theme-label">
                  <span className="app-radio" aria-hidden />
                  {t(`theme.${theme}`)}
                </span>
              </button>
            ))}
          </div>
          <Checkbox
            checked={followSystem}
            onCheckedChange={(next) => setPreference(next ? 'system' : resolved)}
            label={t('appearance.followSystem')}
          />
        </section>

        <section className="nb-card app-card" aria-labelledby="app-size">
          <h2 id="app-size" className="app-card-title">
            {t('appearance.textSize')}
          </h2>
          <div className="app-size">
            <span className="app-size-a is-small" aria-hidden>
              {t('appearance.sample')}
            </span>
            <input
              type="range"
              className="app-range"
              min={0}
              max={TEXT_SCALES.length - 1}
              step={1}
              value={scaleIndex}
              onChange={(event) => setTextScale(Number(event.target.value))}
              aria-labelledby="app-size"
              aria-valuetext={t('appearance.textSizeValue', { percent: formatNumber(percent, loc) })}
              style={{ '--fill': `${fill}%` } as CSSProperties}
            />
            <span className="app-size-a is-large" aria-hidden>
              {t('appearance.sample')}
            </span>
          </div>
        </section>

        <ListGroup className="app-rows">
          <ListRow
            id="app-motion"
            title={t('appearance.reduceMotion')}
            description={
              mounted && systemReduced && motion !== 'reduce'
                ? t('appearance.reduceMotionSystem')
                : t('appearance.reduceMotionSub')
            }
            trailing={
              <Switch
                checked={reduceOn}
                labelledBy="app-motion-title"
                // The OS already reduces motion: «off» could not undo it, so the
                // switch is locked on and the caption says why.
                disabled={mounted && systemReduced && motion !== 'reduce'}
                onCheckedChange={(next) => setMotion(next ? 'reduce' : 'system')}
              />
            }
          />
        </ListGroup>
      </div>
    </div>
  );
}
