'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  type FamilyAccess,
  type RecordSharing,
  type ShareAccessEntry,
  type ShareCode,
  useRecordSharing,
  useRevokeShareCode,
} from '@/entities/health-record';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, formatTimeOfDay } from '@/shared/lib/date';
import {
  Card,
  EmptyState,
  IconCircle,
  InfoNote,
  ListGroup,
  ListRow,
  PrimaryButton,
  ScreenHeader,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';

import { ShareCodeSheet } from './ShareCodeSheet';

/**
 * `/record/share` «چه کسی پرونده‌ات را می‌بیند؟» (CB-REC-05, nbl_Rec_Share): the 24h doctor code + QR (create in a
 * sheet, revoke live ones), bloom's 7-day report link (built on /record/export — not duplicated), the insurer rows
 * (informational only — DECISIONS #12: nothing is ever sent to an insurer), the companions' record scope (meds &
 * appointments only) and the access history (doctor-code / share-link opens only). Back header, no bottom nav.
 */
export function RecordSharePage() {
  const t = useTranslations('record.share');
  const router = useRouter();
  const query = useRecordSharing();
  const [sheetOpen, setSheetOpen] = useState(false);

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="rsh-skel">
        <Skeleton shape="line" width="short" />
        <Skeleton shape="card" className="rsh-skel-card" />
        <Skeleton shape="line" width="short" />
        <Skeleton shape="card" className="rsh-skel-card" />
      </SkeletonGroup>
    );
  } else if (query.isError || !query.data) {
    body = (
      <EmptyState
        icon="warning"
        title={t('loadError')}
        action={
          <PrimaryButton icon="refresh" block={false} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <ShareContent data={query.data} onCreate={() => setSheetOpen(true)} />;
  }

  return (
    <div className="view rsh-page">
      <SkyLayer />
      <div className="scroll rsh-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/record')} backLabel={t('back')} />
        {body}
      </div>
      <ShareCodeSheet open={sheetOpen} onClose={() => setSheetOpen(false)} maxActive={query.data?.doctor.maxActive ?? 5} />
    </div>
  );
}

function ShareContent({ data, onCreate }: { data: RecordSharing; onCreate: () => void }) {
  const t = useTranslations('record.share');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const revoke = useRevokeShareCode();
  const { doctor } = data;
  const active = doctor.items.filter((c) => c.status === 'active');
  const full = doctor.activeCount >= doctor.maxActive;

  return (
    <>
      <SectionTitle title={t('doctor')} />
      <Card className="rsh-doctor">
        <div className="rsh-doctor-head">
          <IconCircle icon="qr" tone="brand" size="lg" />
          <div className="rsh-doctor-text">
            <p className="rsh-doctor-title">{t('codeTitle')}</p>
            <p className="rsh-doctor-body">{t('codeBody')}</p>
          </div>
        </div>
        <PrimaryButton onClick={onCreate} disabled={!doctor.available || full}>
          {t('create')}
        </PrimaryButton>
        {!doctor.available ? <p className="rsh-hint">{t('unavailable')}</p> : null}
        {doctor.available && full ? <p className="rsh-hint">{t('limit', { max: formatNumber(doctor.maxActive, loc) })}</p> : null}
        {active.length ? (
          <ul className="rsh-codes">
            {active.map((c) => (
              <ActiveCodeRow
                key={c.id}
                code={c}
                busy={revoke.isPending && revoke.variables === c.id}
                onRevoke={() => revoke.mutate(c.id)}
              />
            ))}
          </ul>
        ) : null}
        {revoke.isError ? (
          <p className="rsh-error" role="alert">
            {t('revokeError')}
          </p>
        ) : null}
      </Card>
      <ListGroup className="rsh-list">
        <ListRow icon="share" title={t('linkTitle')} description={t('linkBody')} onClick={() => router.push('/record/export')} />
      </ListGroup>

      <SectionTitle title={t('insurer')} />
      <ListGroup className="rsh-list">
        <ListRow title={t('insClaims')} description={t('insClaimsBody')} trailing={<StatusPill tone="neutral">{t('byYou')}</StatusPill>} />
        <ListRow
          title={t('insQuestionnaire')}
          description={t('insQuestionnaireBody')}
          trailing={<StatusPill tone="neutral">{t('byYou')}</StatusPill>}
        />
        <ListRow title={t('insWhole')} description={t('insWholeBody')} trailing={<StatusPill tone="success">{t('never')}</StatusPill>} />
      </ListGroup>
      <InfoNote icon="shield">{t('insNote')}</InfoNote>

      <SectionTitle title={t('family')} />
      <ListGroup className="rsh-list">
        {data.family.length ? (
          data.family.map((f) => (
            <FamilyRow key={f.companionId} family={f} onOpen={() => router.push(`/companions/${f.companionId}`)} />
          ))
        ) : (
          <ListRow icon="users" title={t('familyEmpty')} onClick={() => router.push('/companions')} />
        )}
      </ListGroup>

      <SectionTitle title={t('history')} />
      <ListGroup className="rsh-list">
        {data.accessLog.length ? (
          data.accessLog.map((e) => <AccessRow key={e.id} entry={e} />)
        ) : (
          <ListRow icon="history" iconTone="neutral" title={t('historyEmpty')} />
        )}
      </ListGroup>
      <InfoNote icon="lock">{t('historyNote')}</InfoNote>
    </>
  );
}

function ActiveCodeRow({ code, busy, onRevoke }: { code: ShareCode; busy: boolean; onRevoke: () => void }) {
  const t = useTranslations('record.share');
  const loc = useLocale() as Locale;
  const expires = new Date(code.expiresAt);
  return (
    <li className="rsh-code">
      <div className="rsh-code-text">
        <p className="rsh-code-title">{t('activeCode', { time: formatTimeOfDay(expires, loc) })}</p>
        <p className="rsh-code-meta">{t('codeViews', { count: code.views, n: formatNumber(code.views, loc) })}</p>
      </div>
      <button type="button" className="rsh-revoke" onClick={onRevoke} disabled={busy}>
        {t('revoke')}
      </button>
    </li>
  );
}

function FamilyRow({ family, onOpen }: { family: FamilyAccess; onOpen: () => void }) {
  const t = useTranslations('record.share');
  const parts = [
    family.meds !== 'none' ? t('familyMeds') : null,
    family.appointments !== 'none' ? t('familyAppointments') : null,
  ].filter((p): p is string => p !== null);
  let description: string;
  if (family.status === 'invited') description = t('familyInvited');
  else if (parts.length) description = t('familyScope', { scope: parts.join(t('familyAnd')) });
  else description = t('familyNone');
  return (
    <ListRow
      icon="user"
      title={family.name || t(`familyType.${family.type}`)}
      description={description}
      onClick={onOpen}
    />
  );
}

function AccessRow({ entry }: { entry: ShareAccessEntry }) {
  const t = useTranslations('record.share');
  const loc = useLocale() as Locale;
  const at = new Date(entry.viewedAt);
  const when = t('historyAt', { date: formatDayMonth(at, loc), time: formatTimeOfDay(at, loc) });
  let title: string;
  if (entry.kind === 'report') title = t('historyReport');
  else title = entry.via === 'code' ? t('historyCode') : t('historyQr');
  let meta = `${when} · ${t(`device.${entry.device}`)}`;
  if (entry.linkStatus === 'expired') meta = t('historyExpired', { at: meta });
  else if (entry.linkStatus === 'revoked') meta = t('historyRevoked', { at: meta });
  return (
    <ListRow
      icon={entry.kind === 'report' ? 'fileDoc' : 'qr'}
      iconTone="neutral"
      title={title}
      description={meta}
    />
  );
}
