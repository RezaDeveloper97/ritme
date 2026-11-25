'use client';

import { useTranslations } from 'next-intl';

import { useServicesHubHref } from '@/entities/app-module';
import { useLabRequests, useLabServices } from '@/entities/partner-lab';
import { Link, useDirection, useRouter } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { Card, EmptyState, Icon, PrimaryButton, ScreenHeader, SectionTitle, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import { StatusBadge } from './StatusBadge';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/hlb.css';

/**
 * `/services/home-lab` — «آزمایش و چکاپ در منزل»: the services that can be requested and my requests with their
 * status. Built from existing components (no dedicated artboard, docs/night-bloom/extras.md §3).
 */
export function HomeLabPage() {
  const t = useTranslations('homeLab');
  const router = useRouter();
  // «من» while an admin switched the services hub off.
  const servicesHref = useServicesHubHref();
  const dir = useDirection();
  const services = useLabServices();
  const requests = useLabRequests();

  const retry = () => {
    void services.refetch();
    void requests.refetch();
  };

  let body;
  if (services.isPending || requests.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="hlb-stack">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (services.isError || requests.isError) {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={t('loadError')}
          action={
            <PrimaryButton onClick={retry} loading={services.isFetching || requests.isFetching}>
              {t('retry')}
            </PrimaryButton>
          }
        />
      </Card>
    );
  } else {
    body = (
      <>
        <section className="hlb-stack">
          <SectionTitle title={t('list.servicesTitle')} />
          {services.data.services.length === 0 ? (
            <Card>
              <EmptyState icon="flask" title={t('list.noServices.title')} body={t('list.noServices.body')} />
            </Card>
          ) : (
            <ul className="hlb-list">
              {services.data.services.map((s) => (
                <li key={s.id}>
                  <Link href={`/services/home-lab/new?service=${s.id}`} className="hlb-row" {...pilot(`home-lab.service:${s.id}`)}>
                    <span className="hlb-row-art" aria-hidden>
                      <Icon name={s.kind === 'checkup' ? 'stetho' : 'flask'} size={22} strokeWidth={1.6} />
                    </span>
                    <span className="hlb-row-main">
                      <span className="hlb-row-title">{s.title}</span>
                      <span className="hlb-row-sub">{t('list.partner', { name: s.partner_name })}</span>
                    </span>
                    <Icon name={dir === 'rtl' ? 'chevronLeft' : 'chevronRight'} size={18} className="hlb-row-go" />
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </section>
        <section className="hlb-stack">
          <SectionTitle title={t('list.requestsTitle')} />
          {requests.data.requests.length === 0 ? (
            <Card>
              <EmptyState icon="calendar" title={t('list.noRequests.title')} body={t('list.noRequests.body')} />
            </Card>
          ) : (
            <ul className="hlb-list">
              {requests.data.requests.map((r) => (
                <li key={r.id}>
                  <Link href={`/services/home-lab/${r.id}`} className="hlb-row" {...pilot(`home-lab.request:${r.id}`)}>
                    <span className="hlb-row-main">
                      <span className="hlb-row-title">{r.service.title}</span>
                      <span className="hlb-row-sub">{r.reference}</span>
                    </span>
                    <StatusBadge status={r.status} />
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </section>
      </>
    );
  }

  return (
    <div className="view hlb-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('title')} subtitle={t('tag')} onBack={() => router.push(servicesHref)} backLabel={t('back')} />
        <div className="hlb-body">{body}</div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
