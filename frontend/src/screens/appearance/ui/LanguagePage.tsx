'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useSwitchLocale } from '@/features/switch-locale';
import { useRouter, type Locale } from '@/shared/i18n';
import { calendarSystem, formatLongDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  InfoNote,
  ListGroup,
  ListRow,
  RadioCardGroup,
  ScreenHeader,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

/**
 * Language & calendar (B-N1-10) at `/profile/language` — no artboard; built
 * from the Me_* settings vocabulary. The language list is whatever the backend
 * ships (CLAUDE.md §6: admins add languages without a release). The calendar
 * follows the language (fa → Jalali, otherwise Gregorian, `calendarSystem`);
 * a separate calendar choice is an open question (bloom/QUESTIONS.md).
 */
export function LanguagePage() {
  const t = useTranslations('me');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const mounted = useMounted();
  const { locale, languages, switchLocale, isPending } = useSwitchLocale();
  const calendar = calendarSystem(loc);

  return (
    <div className="view app-page">
      <SkyLayer />
      <div className="scroll app-scroll">
        <ScreenHeader title={t('language.title')} onBack={() => router.push('/profile')} backLabel={t('language.back')} />

        <section className="me-sec" aria-labelledby="lang-list">
          <SectionTitle id="lang-list" title={t('language.languages')} />
          {!mounted || languages.length === 0 ? (
            <SkeletonGroup label={t('loading')} className="lngp-skel">
              <Skeleton shape="card" />
              <Skeleton shape="card" />
            </SkeletonGroup>
          ) : (
            <RadioCardGroup
              label={t('language.languages')}
              value={locale}
              onChange={(code) => (isPending ? undefined : switchLocale(code))}
              options={languages.map((language) => ({
                value: language.code,
                title: <bdi dir={language.direction}>{language.name}</bdi>,
                description: language.englishName,
                icon: 'globe' as const,
              }))}
            />
          )}
          <p className="lngp-hint">{t('language.hint')}</p>
        </section>

        <section className="me-sec" aria-labelledby="lang-cal">
          <SectionTitle id="lang-cal" title={t('language.calendar')} />
          <ListGroup>
            <ListRow
              icon="calendar"
              iconTone="data"
              title={t(`calendars.${calendar}`)}
              description={mounted ? t('language.today', { date: formatLongDate(new Date(), loc) }) : undefined}
            />
          </ListGroup>
          <InfoNote>{t('language.calendarNote')}</InfoNote>
        </section>
      </div>
    </div>
  );
}
