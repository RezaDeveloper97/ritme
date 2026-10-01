'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { APP_VERSION } from '@/shared/config';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber, formatYear, todayParts } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  EmptyState,
  ListGroup,
  ListRow,
  PrimaryButton,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type IconName,
  type Tone,
} from '@/shared/ui';

import { useInfoPage } from '../api/info-page';
import { type InfoPageSection, splitAbout } from '../model/info-page';

/** Icons for admin rows, in order (the content is data; the icons are decoration). */
const ROW_ICONS: ReadonlyArray<readonly [IconName, Tone]> = [
  ['stetho', 'data'],
  ['bookOpen', 'brand'],
  ['shield', 'warm'],
  ['grid', 'neutral'],
];

/**
 * About Ritme (B-N1-12, `nbl_Me_About` / `nbd_Me_About`) at `/profile/about`.
 * Everything but the chrome is admin content (`info_sections`, group `about`):
 * the intro, the rows (tap → the box in a sheet), the link chips (boxes with a
 * web link) and the disclaimer box. «قوانین» and «حریم خصوصی» open /profile/legal.
 */
export function AboutPage() {
  const t = useTranslations('me.about');
  const appName = useTranslations('common')('appName');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const query = useInfoPage('about', loc);
  const [open, setOpen] = useState<InfoPageSection | null>(null);
  const parts = query.data ? splitAbout(query.data) : null;

  return (
    <div className="view abt-page">
      <SkyLayer />
      <div className="scroll abt-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/profile')} backLabel={t('back')} />

        <div className="abt-logo" aria-hidden>
          <span className="abt-logo-word">{appName}</span>
        </div>

        {query.isPending ? (
          <SkeletonGroup label={t('loading')} className="abt-skel">
            <Skeleton />
            <Skeleton />
            <Skeleton width="medium" />
          </SkeletonGroup>
        ) : query.isError ? (
          <EmptyState
            icon="info"
            title={t('error')}
            action={
              <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
                {t('retry')}
              </PrimaryButton>
            }
          />
        ) : parts?.intro ? (
          <p className="abt-intro">{parts.intro.body}</p>
        ) : null}

        <ListGroup className="abt-list">
          {(parts?.rows ?? []).map((row, i) => {
            const [icon, tone] = ROW_ICONS[i % ROW_ICONS.length] ?? ['info', 'brand'];
            return <ListRow key={row.id} icon={icon} iconTone={tone} title={row.heading} onClick={() => setOpen(row)} />;
          })}
          <ListRow
            icon="note"
            iconTone="neutral"
            title={t('terms')}
            onClick={() => router.push('/profile/legal?tab=terms')}
          />
          <ListRow
            icon="lock"
            iconTone="neutral"
            title={t('privacy')}
            onClick={() => router.push('/profile/legal?tab=privacy')}
          />
        </ListGroup>

        {parts && parts.links.length > 0 ? (
          <nav className="abt-links" aria-label={appName}>
            {parts.links.map((l) => (
              <a
                key={l.id}
                className="abt-link"
                href={l.linkUrl ?? undefined}
                aria-label={l.heading}
                {...(l.linkUrl?.startsWith('http') ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
              >
                {l.linkLabel}
              </a>
            ))}
          </nav>
        ) : null}

        <p className="abt-version">
          {t('version', { version: formatNumber(APP_VERSION, loc), year: formatYear(todayParts(loc).year, loc) })}
        </p>
        {parts?.disclaimer ? <p className="abt-disclaimer">{parts.disclaimer.body}</p> : null}
      </div>

      <AppSheet open={open !== null} onClose={() => setOpen(null)} size="full" title={open?.heading}>
        {open ? <p className="abt-sheet-body">{open.body}</p> : null}
      </AppSheet>
    </div>
  );
}
