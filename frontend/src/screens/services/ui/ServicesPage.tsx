'use client';

import { Fragment } from 'react';
import { useLocale, useTranslations } from 'next-intl';

import { type ServiceSection, type ServicesHub, useServicesHub } from '@/entities/service-hub';
import { type Locale } from '@/shared/i18n';
import { openSheet } from '@/shared/sheet';
import {
  EmptyState,
  HeaderButton,
  HubHeader,
  PrimaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import {
  BookingCard,
  CareGrid,
  CheckupsLink,
  LearningCard,
  MotherChildCard,
  ProgramsRow,
  SearchLink,
  ShopCard,
  SosCard,
} from './sections';

/**
 * «خدمات» tab (B-N7-01, nbl_/nbd_v17_Main). Every section, its order and its copy come from GET /services (admin
 * catalog); a destination without a screen yet is shown «به‌زودی» and never linked. The upcoming-booking card
 * renders only when the API returns a booking. The emergency card stays on screen while loading and on error.
 */
export function ServicesPage() {
  const t = useTranslations('services');
  const hub = useServicesHub();

  return (
    <div className="view svc-page">
      <SkyLayer />
      <div className="scroll svc-scroll">
        <HubHeader
          date={t('subtitle')}
          greeting={t('title')}
          actions={
            <HeaderButton label={t('notifications')} icon="bell" variant="soft" onClick={() => openSheet('notifications')} />
          }
        />
        {hub.isPending ? (
          <>
            <SkeletonGroup label={t('loading')} className="svc-skel">
              <Skeleton shape="block" className="svc-skel-search" />
              <div className="svc-grid">
                {Array.from({ length: 6 }, (_, i) => (
                  <Skeleton key={i} shape="card" className="svc-skel-tile" />
                ))}
              </div>
              <Skeleton shape="card" className="svc-skel-row" />
            </SkeletonGroup>
            <SosCard section={null} />
          </>
        ) : hub.isError ? (
          <>
            <div className="svc-sec">
              <EmptyState
                icon="warning"
                title={t('loadError')}
                action={<PrimaryButton onClick={() => void hub.refetch()}>{t('retry')}</PrimaryButton>}
              />
            </div>
            <SosCard section={null} />
          </>
        ) : (
          <HubSections hub={hub.data} />
        )}
      </div>
      <BottomNav />
    </div>
  );
}

/** Sections that carry content (search and the emergency card alone count as an empty hub). */
function hasContent(section: ServiceSection, hub: ServicesHub): boolean {
  switch (section.code) {
    case 'booking':
      return hub.upcomingBooking !== null;
    case 'care':
      return hub.care.length > 0;
    case 'programs':
      return hub.programs.length > 0;
    case 'search':
    case 'emergency':
      return false;
    default:
      return true;
  }
}

function HubSections({ hub }: { hub: ServicesHub }) {
  const t = useTranslations('services');
  const locale = useLocale() as Locale;
  const empty = !hub.sections.some((s) => hasContent(s, hub));

  return (
    <>
      {hub.sections.map((section) => {
        switch (section.code) {
          case 'search':
            return <SearchLink key={section.code} section={section} />;
          case 'booking':
            return hub.upcomingBooking ? (
              <BookingCard key={section.code} booking={hub.upcomingBooking} locale={locale} />
            ) : null;
          case 'care':
            return hub.care.length ? <CareGrid key={section.code} section={section} tiles={hub.care} /> : null;
          case 'checkups':
            return <CheckupsLink key={section.code} section={section} />;
          case 'programs':
            return hub.programs.length ? (
              <ProgramsRow key={section.code} section={section} tiles={hub.programs} />
            ) : null;
          case 'mother_child':
            return <MotherChildCard key={section.code} section={section} />;
          case 'learning':
            return <LearningCard key={section.code} section={section} />;
          case 'shop':
            return <ShopCard key={section.code} section={section} />;
          case 'emergency':
            return (
              <Fragment key={section.code}>
                {empty ? (
                  <div className="svc-sec">
                    <EmptyState icon="grid" title={t('empty.title')} body={t('empty.body')} />
                  </div>
                ) : null}
                <SosCard section={section} />
              </Fragment>
            );
          default:
            return null;
        }
      })}
    </>
  );
}
