'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState, type ReactNode } from 'react';

import {
  formatLabValue,
  isBusy,
  type LabMarker,
  type MarkerInput,
  useAddLabMarker,
  useDeleteLabMarker,
  useLab,
  useLabCatalog,
  useUpdateLabMarker,
  useVerifyLab,
} from '@/entities/lab';
import { getApiErrorCode, getApiErrorMessage, getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  EmptyState,
  Icon,
  IconCircle,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { MarkerSheet } from './MarkerSheet';

type Editing = { kind: 'new' } | { kind: 'edit'; marker: LabMarker } | null;

function fieldErrorsOf(error: unknown): Record<string, string> {
  if (getApiErrorStatus(error) !== 422 || typeof error !== 'object' || error === null) return {};
  const data = (error as { response?: { data?: { errors?: Record<string, unknown> } } }).response?.data?.errors ?? {};
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(data)) {
    const first = Array.isArray(v) ? v.find((m) => typeof m === 'string') : undefined;
    if (typeof first === 'string') out[k] = first;
  }
  return out;
}

function Shell({ header, children, footer }: { header: ReactNode; children: ReactNode; footer?: ReactNode }) {
  return (
    <div className="view lab-screen">
      <SkyLayer />
      <div className="scroll lab-scroll">
        {header}
        <div className="lab-body">{children}</div>
        {footer}
      </div>
    </div>
  );
}

/**
 * `/labs/[id]/verify` (nbl_Lab_Verify): the values read from the sheet, the
 * low-confidence ones highlighted, each correctable / removable in a sheet,
 * missed ones addable, then «تأیید و ادامه» (`POST /labs/{id}/verify`, 202) →
 * processing (the explanation) → result. Editing a ready lab sends it back to
 * review on the server, so this screen also serves «ویرایش مقادیر».
 */
export function LabVerifyPage({ id }: { id: number }) {
  const t = useTranslations('labs');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const valid = Number.isFinite(id) && id > 0;
  const lab = useLab(valid ? id : null);
  const [editing, setEditing] = useState<Editing>(null);
  const catalog = useLabCatalog(editing?.kind === 'new');
  const add = useAddLabMarker(id);
  const update = useUpdateLabMarker(id);
  const remove = useDeleteLabMarker(id);
  const verify = useVerifyLab(id);
  const [verifyError, setVerifyError] = useState<string | null>(null);

  const status = lab.data?.status;
  useEffect(() => {
    if (status && (isBusy(status) || status === 'failed')) router.replace(`/labs/${id}/processing`);
  }, [status, id, router]);

  const header = (
    <ScreenHeader title={t('verify.title')} onBack={() => router.push('/labs')} backLabel={t('common.back')} />
  );

  if (!valid || getApiErrorStatus(lab.error) === 404) {
    return (
      <Shell header={header}>
        <EmptyState
          icon="flask"
          title={t('common.notFoundTitle')}
          body={t('common.notFoundBody')}
          action={<PrimaryButton onClick={() => router.push('/labs')}>{t('common.toList')}</PrimaryButton>}
        />
      </Shell>
    );
  }
  if (!mounted || lab.isPending) {
    return (
      <Shell header={header}>
        <SkeletonGroup label={t('common.loading')}>
          <Skeleton shape="line" width="medium" />
          <Skeleton shape="block" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }
  if (lab.isError) {
    return (
      <Shell header={header}>
        <Card className="lab-state" role="alert">
          <IconCircle icon="warning" tone="danger" size="lg" />
          <p className="lab-state-text">{t('common.loadError')}</p>
          <SecondaryButton icon="refresh" block={false} loading={lab.isFetching} onClick={() => void lab.refetch()}>
            {t('common.retry')}
          </SecondaryButton>
        </Card>
      </Shell>
    );
  }

  const data = lab.data;
  const saving = add.isPending || update.isPending || remove.isPending;
  const sheetError = add.error ?? update.error ?? remove.error;
  const closeSheet = () => {
    setEditing(null);
    add.reset();
    update.reset();
    remove.reset();
  };
  const onSave = (input: MarkerInput) => {
    if (editing?.kind === 'edit') update.mutate({ markerId: editing.marker.id, input }, { onSuccess: closeSheet });
    else add.mutate(input, { onSuccess: closeSheet });
  };
  const submit = () => {
    setVerifyError(null);
    if (data.source === 'manual') {
      router.push(`/labs/${id}`);
      return;
    }
    verify.mutate(undefined, {
      onSuccess: (l) => router.replace(l.status === 'ready' ? `/labs/${id}` : `/labs/${id}/processing`),
      onError: (e) => {
        const code = getApiErrorCode(e);
        setVerifyError(
          code === 'lab_interpret_limit' || code === 'lab_busy' || getApiErrorStatus(e) === 429
            ? (getApiErrorMessage(e) ?? t('verify.limit'))
            : (getApiErrorMessage(e) ?? t('verify.error')),
        );
      },
    });
  };

  return (
    <Shell
      header={header}
      footer={
        <div className="lab-footer">
          {verifyError ? (
            <p className="lab-error is-block" role="alert">
              {verifyError}
            </p>
          ) : null}
          <PrimaryButton loading={verify.isPending} disabled={data.markers.length === 0 || saving} onClick={submit}>
            {t('verify.submit')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="lab-verify-intro">
        <h2 className="lab-display">{t('verify.heading')}</h2>
        <p className="lab-lead">{t('verify.lead')}</p>
      </div>
      {data.lowConfidenceCount > 0 ? (
        <p className="lab-banner is-warm" role="status">
          <Icon name="warning" size={18} strokeWidth={2} />
          <span>{t('verify.lowConfidence', { count: data.lowConfidenceCount, n: formatNumber(data.lowConfidenceCount, locale) })}</span>
        </p>
      ) : null}
      <Card as="section" className="lab-verify-card" aria-label={t('verify.listLabel')}>
        {data.markers.length === 0 ? <p className="lab-empty-body">{t('verify.empty')}</p> : null}
        <ul className="lab-vrows">
          {data.markers.map((m) => (
            <li key={m.id} className={clsx('lab-vrow', m.lowConfidence && 'is-low')}>
              <span className="lab-vrow-name">{m.name}</span>
              <button
                type="button"
                className="lab-vrow-value"
                onClick={() => setEditing({ kind: 'edit', marker: m })}
                aria-label={t('verify.editLabel', { name: m.name, value: formatLabValue(m, locale), unit: m.unit ?? '' })}
              >
                <span className="lab-vrow-num">{formatLabValue(m, locale)}</span>
                {m.unit ? <span className="lab-vrow-unit">{m.unit}</span> : null}
              </button>
              <span className="lab-vrow-mark" aria-hidden>
                <Icon name={m.lowConfidence ? 'pencil' : 'check'} size={16} strokeWidth={2.2} />
              </span>
              {m.lowConfidence ? <span className="sr-only">{t('verify.lowOne')}</span> : null}
            </li>
          ))}
        </ul>
        <button type="button" className="lab-add" onClick={() => setEditing({ kind: 'new' })}>
          <Icon name="plus" size={18} />
          <span>{t('verify.add')}</span>
        </button>
      </Card>

      {editing ? (
        <MarkerSheet
          marker={editing.kind === 'edit' ? editing.marker : null}
          catalog={catalog.data ?? []}
          busy={saving}
          fieldErrors={fieldErrorsOf(sheetError)}
          error={sheetError && getApiErrorStatus(sheetError) !== 422 ? (getApiErrorMessage(sheetError) ?? t('verify.saveError')) : null}
          onClose={closeSheet}
          onSave={onSave}
          onDelete={
            editing.kind === 'edit' ? () => remove.mutate(editing.marker.id, { onSuccess: closeSheet }) : undefined
          }
        />
      ) : null}
    </Shell>
  );
}
