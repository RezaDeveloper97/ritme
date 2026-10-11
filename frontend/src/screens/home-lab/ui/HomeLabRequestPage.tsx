'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  type LabStatus,
  useCancelLabRequest,
  useImportLabResult,
  useLabRequest,
} from '@/entities/partner-lab';
import { getApiErrorMessage, getApiErrorStatus } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, fromApiDate } from '@/shared/lib/date';
import {
  Card,
  Checkbox,
  EmptyState,
  InfoNote,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StepTimeline,
  type TimelineStep,
} from '@/shared/ui';

import { StatusBadge } from './StatusBadge';

const FLOW: readonly LabStatus[] = ['requested', 'accepted', 'sampled', 'result_ready'];

/**
 * `/services/home-lab/[id]` — status of one request: the four-step timeline, what she sent (until the 30-day purge),
 * cancel while it is requested / accepted, and — when the result is ready and she ticked the consent — the
 * re-confirmed copy into her lab results.
 */
export function HomeLabRequestPage({ id }: { id: number }) {
  const t = useTranslations('homeLab');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const q = useLabRequest(id);
  const cancel = useCancelLabRequest(id);
  const importer = useImportLabResult(id);
  const [consent, setConsent] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const header = (
    <ScreenHeader title={t('detail.title')} subtitle={t('tag')} onBack={() => router.push('/services/home-lab')} backLabel={t('back')} />
  );
  const shell = (children: React.ReactNode) => (
    <div className="view hlb-view">
      <SkyLayer />
      <div className="scroll">
        {header}
        <div className="hlb-body">{children}</div>
        <div className="page-tail" />
      </div>
    </div>
  );

  if (q.isPending) {
    return shell(
      <SkeletonGroup label={t('loading')} className="hlb-stack">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>,
    );
  }
  if (q.isError) {
    const missing = getApiErrorStatus(q.error) === 404;
    return shell(
      <Card>
        <EmptyState
          icon="warning"
          title={missing ? t('detail.notFound') : t('loadError')}
          action={
            missing ? (
              <PrimaryButton onClick={() => router.push('/services/home-lab')}>{t('back')}</PrimaryButton>
            ) : (
              <PrimaryButton onClick={() => void q.refetch()} loading={q.isFetching}>
                {t('retry')}
              </PrimaryButton>
            )
          }
        />
      </Card>,
    );
  }

  const r = q.data;
  const rank = FLOW.indexOf(r.status);
  const steps: TimelineStep[] = FLOW.map((s, i) => ({
    id: s,
    title: t(`status.${s}`),
    state: rank < 0 ? 'todo' : i < rank ? 'done' : i === rank ? 'current' : 'todo',
  }));
  const day = formatLongDate(fromApiDate(r.preferred_date), locale);

  const onCancel = () => {
    setError(null);
    cancel.mutate(undefined, { onError: () => setError(t('detail.cancelError')) });
  };
  const onImport = () => {
    setError(null);
    importer.mutate(consent, { onError: (e) => setError(getApiErrorMessage(e) ?? t('result.importError')) });
  };

  return shell(
    <>
      <Card className="hlb-svc">
        <span className="hlb-row-main">
          <span className="hlb-row-title">{r.service.title}</span>
          <span className="hlb-row-sub">
            {t('detail.reference')}: {r.reference}
          </span>
          <span className="hlb-row-sub">{t('detail.when', { day, window: t(`form.windows.${r.preferred_window}`) })}</span>
        </span>
        <StatusBadge status={r.status} />
      </Card>

      {rank >= 0 ? (
        <Card>
          <StepTimeline
            steps={steps}
            label={t('detail.steps')}
            stateLabels={{ done: t('detail.stateDone'), current: t('detail.stateCurrent'), todo: t('detail.stateTodo') }}
            locale={locale}
          />
        </Card>
      ) : (
        <InfoNote icon="info">{r.status === 'rejected' ? t('detail.rejectedNote') : t('detail.cancelledNote')}</InfoNote>
      )}

      {r.status === 'result_ready' && (
        <Card as="section" className="hlb-stack">
          <SectionTitle title={t('result.title')} level={3} />
          {r.imported ? (
            <>
              <p className="hlb-note">{t('result.imported')}</p>
              {r.lab_id ? (
                <Link href={`/labs/${r.lab_id}`} className="hlb-link">
                  {t('result.open')}
                </Link>
              ) : null}
            </>
          ) : r.can_import ? (
            <>
              <p className="hlb-note">{t('result.body')}</p>
              <Checkbox checked={consent} onCheckedChange={setConsent} label={t('result.consent', { partner: r.service.partner_name })} />
              <PrimaryButton onClick={onImport} disabled={!consent} loading={importer.isPending}>
                {importer.isPending ? t('result.importing') : t('result.import')}
              </PrimaryButton>
            </>
          ) : (
            <p className="hlb-note">{t('result.noConsent')}</p>
          )}
        </Card>
      )}

      <Card as="section" className="hlb-stack">
        <SectionTitle title={t('detail.yourDetails')} level={3} />
        {r.details ? (
          <dl className="hlb-dl">
            <dt>{t('form.fullName')}</dt>
            <dd>{r.details.full_name}</dd>
            <dt>{t('form.phone')}</dt>
            <dd dir="ltr">{r.details.phone}</dd>
            <dt>{t('form.city')}</dt>
            <dd>{r.details.city}</dd>
            <dt>{t('form.address')}</dt>
            <dd>{r.details.address}</dd>
            {r.details.note ? (
              <>
                <dt>{t('form.note')}</dt>
                <dd>{r.details.note}</dd>
              </>
            ) : null}
          </dl>
        ) : (
          <p className="hlb-note">{t('detail.purged')}</p>
        )}
      </Card>

      {error && (
        <p role="alert" className="hlb-err">
          {error}
        </p>
      )}
      {r.can_cancel && (
        <SecondaryButton onClick={onCancel} loading={cancel.isPending}>
          {cancel.isPending ? t('detail.cancelling') : t('detail.cancel')}
        </SecondaryButton>
      )}
      {(r.status === 'rejected' || r.status === 'cancelled') && (
        <PrimaryButton onClick={() => router.push('/services/home-lab')}>{t('detail.newRequest')}</PrimaryButton>
      )}
    </>,
  );
}
