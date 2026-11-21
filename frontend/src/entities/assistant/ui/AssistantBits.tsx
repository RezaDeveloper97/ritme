'use client';

import { useLocale, useTranslations } from 'next-intl';

import { getApiErrorCode, getApiErrorMessage } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, EmptyState, Icon, InfoNote, PrimaryButton, Skeleton, SkeletonGroup, StatusPill, UrgentCard } from '@/shared/ui';

import { useAssistantConsent, useGrantConsent } from '../api/queries';
import type { AssistantMessage, TriageLevel } from '../api/schema';
import { EMERGENCY_PHONE, TRIAGE_TONE } from '../model/format';
import '@/shared/ui/styles/as.css';

/** Triage pill («غیر اورژانسی» …) — a suggestion of the assistant, never a diagnosis. */
export function TriagePill({ level }: { level: TriageLevel }) {
  const t = useTranslations('assistant');
  return (
    <StatusPill tone={TRIAGE_TONE[level]} icon={level === 'emergency' ? 'warning' : 'info'}>
      {t(`triage.${level}`)}
    </StatusPill>
  );
}

const KNOWN = new Set(['ai_unavailable', 'ai_failed', 'ai_budget_exhausted', 'ai_user_budget_exhausted', 'ai_busy', 'too_many_requests', 'summary_empty', 'summary_failed', 'share_consent_required', 'booking_closed', 'booking_not_found', 'assistant_photo_not_found', 'storage_unavailable']);

/** Localized text for a failed assistant call: known `error_code` → our copy, else the server's message, else generic. */
export function useAssistantError() {
  const t = useTranslations('assistant');
  return (error: unknown, code?: string): string => {
    const c = code ?? getApiErrorCode(error);
    if (c && KNOWN.has(c)) return t(`errors.${c}` as 'errors.generic');
    return (error ? getApiErrorMessage(error) : undefined) ?? t('errors.generic');
  };
}

export function LoadError({ onRetry, busy }: { onRetry: () => void; busy?: boolean }) {
  const t = useTranslations('assistant');
  return (
    <Card>
      <EmptyState icon="warning" title={t('common.loadError')} action={<PrimaryButton icon="refresh" loading={busy} onClick={onRetry}>{t('common.retry')}</PrimaryButton>} />
    </Card>
  );
}

export function BodySkeleton({ rows = 3 }: { rows?: number }) {
  const t = useTranslations('assistant');
  return (
    <SkeletonGroup label={t('common.loading')} className="as-body">
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} shape="block" />
      ))}
    </SkeletonGroup>
  );
}

/** The 115 card: prominent, with a tel: link. `card` comes from the SSE stream or a stored message. */
export function EmergencyCard({ card }: { card?: AssistantMessage['card'] }) {
  const t = useTranslations('assistant');
  const locale = useLocale() as Locale;
  const phone = card?.phone || EMERGENCY_PHONE;
  return (
    <UrgentCard
      title={card?.title || t('emergency.title')}
      action={
        <a className="as-call" href={`tel:${phone}`}>
          <Icon name="phone" size={20} />
          <span>{card?.call_label || t('emergency.call')}</span>
          <strong>{formatNumber(phone, locale)}</strong>
        </a>
      }
    >
      {card?.body || t('emergency.body')}
    </UrgentCard>
  );
}

/** Standing «not a doctor» note under hub / chat / summary. */
export function Disclaimer() {
  const t = useTranslations('assistant');
  return <InfoNote icon="shield">{t('common.disclaimer')}</InfoNote>;
}

/** The consent text in force with an accept button; renders nothing once granted. */
export function ConsentCard({ code, onGranted }: { code: 'ai_assistant' | 'assistant_profile'; onGranted?: () => void }) {
  const t = useTranslations('assistant');
  const query = useAssistantConsent(code);
  const grant = useGrantConsent(code);
  if (query.isPending) return <Skeleton shape="card" />;
  const c = query.data;
  if (query.isError || !c) return <LoadError onRetry={() => void query.refetch()} busy={query.isFetching} />;
  if (!c.needs_consent && c.granted) return null;
  return (
    <Card className="as-consent">
      <h2 className="as-h2">{c.title}</h2>
      <p className="as-text">{c.body}</p>
      {c.points.length ? (
        <ul className="as-points">
          {c.points.map((p) => (
            <li key={p}>{p}</li>
          ))}
        </ul>
      ) : null}
      {grant.isError ? <p className="as-error" role="alert">{t('errors.generic')}</p> : null}
      <PrimaryButton block loading={grant.isPending} onClick={() => grant.mutate(c.version, { onSuccess: () => onGranted?.() })}>
        {t('consent.accept')}
      </PrimaryButton>
    </Card>
  );
}
