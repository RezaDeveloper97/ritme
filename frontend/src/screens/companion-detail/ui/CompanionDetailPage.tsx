'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  type CompanionAuditEntry,
  type CompanionGrants,
  type CompanionInvite,
  companionName,
  hoursUntil,
  type OwnerCompanion,
  PersonBubble,
  sameGrants,
  useCompanion,
  useCompanionAudit,
  useRenewCompanionInvite,
  useRevokeCompanion,
  useUpdateCompanionChildren,
  useUpdateCompanionGrants,
} from '@/entities/companion';
import { AccessEditor, ChildrenPicker, InviteCodeCard, sameChildIds } from '@/features/invite-companion';
import { getApiErrorMessage, getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { AppSheet } from '@/shared/sheet';
import {
  EmptyState,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

const ACTIVITY_LIMIT = 5;

/** A 422/429/503 carries a localized API message; anything else gets the generic copy. */
function apiText(error: unknown, fallback: string): string {
  const status = getApiErrorStatus(error);
  if (status === 422 || status === 429 || status === 503) return getApiErrorMessage(error) ?? fallback;
  return fallback;
}

/**
 * One companion (`/companions/[id]`, B-N4-04): who they are, the per-section
 * access editor (PUT /companions/{id}/grants), a fresh code for a pending
 * invite (POST …/renew, shown once), the shared children of a spouse (PUT
 * …/children, B-N4-10b), recent activity from the audit trail and
 * «حذف همدم» behind a confirmation (DELETE /companions/{id}).
 */
export function CompanionDetailPage({ id }: { id: number }) {
  const t = useTranslations('companions');
  const router = useRouter();
  const query = useCompanion(id);
  const companion = query.data;
  const toList = () => router.push('/companions');
  // The query is gated on the localStorage token, so the server always renders the skeleton and the
  // first client pass must match it (no hydration mismatch) — loading is decided only after mount.
  const mounted = useMounted();
  const validId = Number.isFinite(id) && id > 0;

  let body;
  if (companion) {
    body = <DetailBody companion={companion} onRemoved={toList} />;
  } else if (!mounted || (validId && query.isPending)) {
    body = (
      <SkeletonGroup label={t('detail.loading')} className="cmp-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError && getApiErrorStatus(query.error) !== 404) {
    body = (
      <EmptyState
        icon="users"
        title={t('list.errorTitle')}
        body={t('list.errorBody')}
        action={
          <PrimaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('list.retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = (
      <EmptyState
        icon="users"
        title={t('detail.notFoundTitle')}
        body={t('detail.notFoundBody')}
        action={
          <PrimaryButton block={false} onClick={toList}>
            {t('detail.toList')}
          </PrimaryButton>
        }
      />
    );
  }

  const title = companion ? (companionName(companion) ?? t('unnamed')) : t('detail.title');
  return (
    <div className="view cmp-page">
      <SkyLayer />
      <div className="scroll cmp-scroll">
        <ScreenHeader title={title} onBack={toList} backLabel={t('back')} />
        <div className="cmp-body">{body}</div>
      </div>
    </div>
  );
}

function DetailBody({ companion, onRemoved }: { companion: OwnerCompanion; onRemoved: () => void }) {
  const t = useTranslations('companions');
  const locale = useLocale() as Locale;
  const name = companionName(companion) ?? t('unnamed');
  const when = companion.status === 'active' ? companion.acceptedAt : companion.invitedAt;

  return (
    <>
      <section className="nb-card cmp-hero">
        <PersonBubble name={companionName(companion)} tone="companion" size="lg" />
        <div className="cmp-hero-text">
          <b className="cmp-hero-name">{name}</b>
          <span className="cmp-hero-sub">
            {t('statusLine', { type: t(`types.${companion.type}`), status: t(`status.${companion.status}`) })}
          </span>
          {when ? (
            <span className="cmp-hero-date">
              {t(companion.status === 'active' ? 'detail.since' : 'detail.invitedOn', {
                date: formatLongDate(new Date(when), locale),
              })}
            </span>
          ) : null}
        </div>
      </section>

      {companion.status === 'invited' ? <PendingInvite companion={companion} /> : null}

      <GrantsSection companion={companion} />

      {companion.type === 'spouse' ? <ChildrenSection companion={companion} /> : null}

      <Activity companionId={companion.id} />

      <RemoveCompanion companion={companion} name={name} onRemoved={onRemoved} />
    </>
  );
}

function PendingInvite({ companion }: { companion: OwnerCompanion }) {
  const t = useTranslations('companions');
  const locale = useLocale() as Locale;
  const renew = useRenewCompanionInvite(companion.id);
  // The renewed code exists only in this state — never cached, stored or logged.
  const [fresh, setFresh] = useState<CompanionInvite | null>(null);
  const hours = companion.invite ? hoursUntil(companion.invite.expiresAt) : 0;

  return (
    <section className="cmp-sec" aria-labelledby="cmp-pending-title">
      <SectionTitle id="cmp-pending-title" title={t('detail.pendingTitle')} />
      {fresh ? (
        <InviteCodeCard invite={fresh} lead={t('flow.invite.codeLead')} />
      ) : (
        <div className="nb-card cmp-pending">
          {companion.invite?.phone ? (
            <p className="cmp-pending-line">
              {t.rich('detail.sentTo', {
                phone: companion.invite.phone,
                ltr: (chunks) => <bdi dir="ltr">{chunks}</bdi>,
              })}
            </p>
          ) : null}
          <p className={hours > 0 ? 'cmp-pending-line' : 'cmp-pending-line is-expired'}>
            {hours > 0 ? t('detail.expiresIn', { hours: formatNumber(hours, locale) }) : t('detail.expired')}
          </p>
          <SecondaryButton
            icon="refresh"
            loading={renew.isPending}
            onClick={() => renew.mutate(undefined, { onSuccess: (r) => setFresh(r.invite) })}
          >
            {t('detail.renew')}
          </SecondaryButton>
          <p className="cmp-pending-note">{t('detail.renewNote')}</p>
          {renew.isError ? (
            <p className="cmp-error" role="alert">
              {apiText(renew.error, t('detail.renewError'))}
            </p>
          ) : null}
        </div>
      )}
    </section>
  );
}

function GrantsSection({ companion }: { companion: OwnerCompanion }) {
  const t = useTranslations('companions');
  const save = useUpdateCompanionGrants(companion.id);
  // null = untouched: the editor shows the server's grants.
  const [draft, setDraft] = useState<CompanionGrants | null>(null);
  const [saved, setSaved] = useState(false);
  const value = draft ?? companion.grants;
  const dirty = draft !== null && !sameGrants(draft, companion.grants);

  const onSave = () => {
    if (!draft || save.isPending) return;
    save.mutate(draft, {
      onSuccess: () => {
        setDraft(null);
        setSaved(true);
      },
    });
  };

  return (
    <section className="cmp-sec" aria-labelledby="cmp-access-title">
      <SectionTitle id="cmp-access-title" title={t('detail.accessTitle')} />
      <p className="cmp-lead">{t('detail.accessLead')}</p>
      <AccessEditor
        value={value}
        disabled={save.isPending}
        onChange={(grants) => {
          setSaved(false);
          setDraft(grants);
        }}
      />
      {dirty ? (
        <PrimaryButton loading={save.isPending} onClick={onSave}>
          {t('detail.save')}
        </PrimaryButton>
      ) : null}
      {saved && !dirty ? (
        <p className="cmp-status" role="status">
          {t('detail.saved')}
        </p>
      ) : null}
      {save.isError ? (
        <p className="cmp-error" role="alert">
          {apiText(save.error, t('detail.saveError'))}
        </p>
      ) : null}
    </section>
  );
}

/** Spouse only: which of her children are shared (PUT /companions/{id}/children; empty = none). */
function ChildrenSection({ companion }: { companion: OwnerCompanion }) {
  const t = useTranslations('companions');
  const save = useUpdateCompanionChildren(companion.id);
  const shared = companion.family?.sharedChildIds ?? [];
  // null = untouched: the picker shows the server's list.
  const [draft, setDraft] = useState<number[] | null>(null);
  const [saved, setSaved] = useState(false);
  const value = draft ?? shared;
  const dirty = draft !== null && !sameChildIds(draft, shared);

  const onSave = () => {
    if (!draft || save.isPending) return;
    save.mutate(draft, {
      onSuccess: () => {
        setDraft(null);
        setSaved(true);
      },
    });
  };

  return (
    <section className="cmp-sec" aria-labelledby="cmp-children-title">
      <SectionTitle id="cmp-children-title" title={t('detail.childrenTitle')} />
      <p className="cmp-lead">{t('detail.childrenLead')}</p>
      <ChildrenPicker
        value={value}
        disabled={save.isPending}
        onChange={(ids) => {
          setSaved(false);
          setDraft(ids);
        }}
      />
      {dirty ? (
        <PrimaryButton loading={save.isPending} onClick={onSave}>
          {t('detail.childrenSave')}
        </PrimaryButton>
      ) : null}
      {saved && !dirty ? (
        <p className="cmp-status" role="status">
          {t('detail.childrenSaved')}
        </p>
      ) : null}
      {save.isError ? (
        <p className="cmp-error" role="alert">
          {apiText(save.error, t('detail.childrenSaveError'))}
        </p>
      ) : null}
    </section>
  );
}

function Activity({ companionId }: { companionId: number }) {
  const t = useTranslations('companions');
  const locale = useLocale() as Locale;
  const audit = useCompanionAudit();
  if (audit.isError) return null;
  const rows: CompanionAuditEntry[] = (audit.data ?? [])
    .filter((e) => e.companionId === companionId)
    .slice(0, ACTIVITY_LIMIT);

  return (
    <section className="cmp-sec" aria-labelledby="cmp-activity-title">
      <SectionTitle id="cmp-activity-title" title={t('detail.activity')} />
      {audit.isPending ? (
        <SkeletonGroup label={t('detail.loading')}>
          <Skeleton />
          <Skeleton width="medium" />
        </SkeletonGroup>
      ) : rows.length === 0 ? (
        <p className="cmp-lead">{t('detail.activityEmpty')}</p>
      ) : (
        <ul className="nb-card cmp-activity">
          {rows.map((e) => {
            const action = t(`detail.actions.${e.action}`);
            return (
              <li key={e.id} className="cmp-activity-row">
                <span className="cmp-activity-what">
                  {e.section ? t('detail.activityLine', { action, section: t(`sectionsShort.${e.section}`) }) : action}
                  {e.actor?.isMe ? <span className="cmp-activity-by"> · {t('detail.byYou')}</span> : null}
                </span>
                <span className="cmp-activity-when">{formatLongDate(new Date(e.at), locale)}</span>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

function RemoveCompanion({ companion, name, onRemoved }: { companion: OwnerCompanion; name: string; onRemoved: () => void }) {
  const t = useTranslations('companions');
  const revoke = useRevokeCompanion(companion.id);
  const [open, setOpen] = useState(false);

  return (
    <>
      <button type="button" className="cmp-remove" onClick={() => setOpen(true)}>
        {t('detail.remove')}
      </button>
      <AppSheet
        open={open}
        onClose={() => (revoke.isPending ? undefined : setOpen(false))}
        size="half"
        title={t('detail.removeTitle', { name })}
        footer={
          <div className="cmp-sheet-btns">
            <SecondaryButton onClick={() => setOpen(false)} disabled={revoke.isPending}>
              {t('detail.cancel')}
            </SecondaryButton>
            <SecondaryButton
              variant="text"
              className="cmp-danger-text"
              loading={revoke.isPending}
              onClick={() => revoke.mutate(undefined, { onSuccess: onRemoved })}
            >
              {t('detail.removeConfirm')}
            </SecondaryButton>
          </div>
        }
      >
        <p className="cmp-sheet-body">{t('detail.removeBody')}</p>
        {revoke.isError ? (
          <p className="cmp-error" role="alert">
            {apiText(revoke.error, t('detail.removeError'))}
          </p>
        ) : null}
      </AppSheet>
    </>
  );
}
