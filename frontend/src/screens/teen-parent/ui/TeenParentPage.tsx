'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useId, useState } from 'react';

import { useRevokeCompanion } from '@/entities/companion';
import {
  maskParentView,
  NO_TEEN_GRANTS,
  ParentViewCard,
  primaryParentLink,
  TEEN_SHARE_KEYS,
  type TeenGrants,
  type TeenParentInvite,
  type TeenParentLink,
  type TeenShareKey,
  type TeenToday,
  useForgetParentLink,
  useInviteParent,
  useRenewParentInvite,
  useSaveParentNote,
  useTeenToday,
  useUpdateParentGrants,
  withShare,
} from '@/entities/teen';
import { useUserProfile } from '@/entities/user';
import { InviteCodeCard, normalizeMobile } from '@/features/invite-companion';
import { getApiErrorMessage, getApiErrorStatus, getApiSaveErrorMessage } from '@/shared/api';
import { useRouter } from '@/shared/i18n';
import { AppSheet } from '@/shared/sheet';
import {
  EmptyState,
  Icon,
  IconCircle,
  InfoNote,
  ListGroup,
  ListRow,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  Switch,
} from '@/shared/ui';

const NOTE_MAX = 280;

/** Message keys of the three switches, in the board's order. */
const SHARE_COPY = {
  teenPeriodWeek: { title: 'share.periodWeek.title', desc: 'share.periodWeek.desc' },
  teenKit: { title: 'share.kit.title', desc: null },
  teenNotes: { title: 'share.notes.title', desc: 'share.notes.desc' },
} as const satisfies Record<TeenShareKey, { title: string; desc: string | null }>;

/** A 422 / 429 carries a calm, specific server message (e.g. `teen_parent_only`); anything else the fallback. */
function apiText(error: unknown, fallback: string): string {
  const status = getApiErrorStatus(error);
  if (status === 422 || status === 429) return getApiErrorMessage(error) ?? fallback;
  return getApiSaveErrorMessage(error, fallback);
}

/**
 * «همراهی مادر» (`/teen/parent`, CB-TEEN-03, nbl_Teen_Parent): the teen
 * chooses what her mother sees — three view-only switches, all off until she
 * turns one on — with a live preview of the exact card her mother gets
 * (`parent_preview` masked by the switches), the note she may share, the
 * invite by phone (bloom's companions with `type: parent`), and revoke at any
 * time. A flow: back header, no bottom nav.
 */
export function TeenParentPage() {
  const t = useTranslations('teen.parent');
  const router = useRouter();
  const query = useTeenToday();
  const needsOnboarding = query.data?.needsOnboarding ?? false;

  useEffect(() => {
    if (needsOnboarding) router.replace('/teen/onboarding');
  }, [needsOnboarding, router]);

  let body;
  if (query.isPending || needsOnboarding) {
    body = (
      <SkeletonGroup label={t('loading')} className="tnp-skel">
        <Skeleton shape="line" width="medium" />
        <Skeleton shape="block" className="tnp-skel-card" />
        <Skeleton shape="block" className="tnp-skel-card is-short" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="mother"
        title={t('loadError')}
        action={
          <PrimaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (!query.data.isTeenMode) {
    body = <EmptyState icon="mother" title={t('notTeen')} />;
  } else {
    body = <Sharing data={query.data} />;
  }

  return (
    <div className="view tnp-page">
      <SkyLayer />
      <div className="scroll tnp-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/home')} backLabel={t('back')} />
        {body}
      </div>
    </div>
  );
}

function Sharing({ data }: { data: TeenToday }) {
  const t = useTranslations('teen.parent');
  const profile = useUserProfile();
  const link = primaryParentLink(data.parentLinks);
  // Before an invite exists the switches are a local draft — all off (most private).
  const [draft, setDraft] = useState<TeenGrants>(NO_TEEN_GRANTS);
  const [inviteOpen, setInviteOpen] = useState(false);
  const [shownInvite, setShownInvite] = useState<TeenParentInvite | null>(null);
  const update = useUpdateParentGrants();
  const grants = link?.grants ?? draft;
  const name = profile.data?.name?.trim().split(/\s+/)[0] || t('view.you');
  const preview = maskParentView(data.parentPreview, grants);

  const toggle = (key: TeenShareKey, on: boolean) => {
    const next = withShare(grants, key, on);
    if (link) update.mutate({ id: link.id, grants: next });
    else setDraft(next);
  };

  return (
    <>
      <div className="tnp-body">
        <div className="tnp-intro">
          <h2 className="tnp-title">{t('heading')}</h2>
          <p className="tnp-lead">{t('lead')}</p>
        </div>

        {link ? <LinkStatus link={link} onCode={setShownInvite} /> : null}

        <ListGroup className="tnp-share">
          {TEEN_SHARE_KEYS.map((key) => {
            const copy = SHARE_COPY[key];
            const rowId = `tnp-${key}`;
            return (
              <ListRow
                key={key}
                id={rowId}
                title={t(copy.title)}
                description={copy.desc ? t(copy.desc) : undefined}
                trailing={
                  <Switch
                    checked={grants[key] === 'view'}
                    onCheckedChange={(on) => toggle(key, on)}
                    labelledBy={`${rowId}-title`}
                  />
                }
              />
            );
          })}
          {grants.teenNotes === 'view' ? <NoteEditor saved={data.profile?.parentNote ?? ''} /> : null}
        </ListGroup>
        {update.isError ? (
          <p className="tnp-error" role="alert">
            {apiText(update.error, t('share.saveError'))}
          </p>
        ) : null}

        <section className="tnp-sec" aria-labelledby="tnp-preview">
          <SectionTitle id="tnp-preview" title={t('preview.title')} />
          <ParentViewCard variant="preview" name={name} view={preview} />
        </section>

        <InfoNote icon="shield" className="tnp-safe">
          {t('readOnly')}
        </InfoNote>

        {link ? <RevokeLink link={link} /> : null}
      </div>

      {link ? null : (
        <div className="tnp-footer">
          <PrimaryButton icon="send" onClick={() => setInviteOpen(true)}>
            {t('invite.cta')}
          </PrimaryButton>
        </div>
      )}

      <InviteSheet
        open={inviteOpen}
        grants={draft}
        onClose={() => setInviteOpen(false)}
        onCreated={(invite) => {
          setInviteOpen(false);
          setShownInvite(invite);
        }}
      />
      <CodeSheet invite={shownInvite} onClose={() => setShownInvite(null)} />
    </>
  );
}

/** «یادداشت برای مادرت» — shown while «علائم و یادداشت‌ها» is on; saved with PUT /teen/parent-note. */
function NoteEditor({ saved }: { saved: string }) {
  const t = useTranslations('teen.parent.note');
  const id = useId();
  const countId = useId();
  const save = useSaveParentNote();
  const [value, setValue] = useState(saved);
  const dirty = value.trim() !== saved.trim();

  return (
    <div className="tnp-note">
      <label htmlFor={id} className="tnp-note-label">
        {t('label')}
      </label>
      <textarea
        id={id}
        className="tnp-note-input"
        value={value}
        maxLength={NOTE_MAX}
        rows={3}
        placeholder={t('placeholder')}
        aria-describedby={countId}
        onChange={(e) => {
          setValue(e.target.value);
          if (save.isSuccess || save.isError) save.reset();
        }}
      />
      <div className="tnp-note-foot">
        <span id={countId} className="tnp-note-count">
          {t('count', { count: value.length, max: NOTE_MAX })}
        </span>
        <SecondaryButton
          block={false}
          className="tnp-note-save"
          loading={save.isPending}
          disabled={!dirty}
          onClick={() => save.mutate(value)}
        >
          {t('save')}
        </SecondaryButton>
      </div>
      {save.isError ? (
        <p className="tnp-error" role="alert">
          {apiText(save.error, t('error'))}
        </p>
      ) : save.isSuccess && !dirty ? (
        <p className="tnp-ok" role="status">
          <Icon name="checkCircle" size={16} />
          {t('saved')}
        </p>
      ) : null}
    </div>
  );
}

/** Who the link is with, and — for a pending invite — a fresh code (POST …/renew, shown once). */
function LinkStatus({ link, onCode }: { link: TeenParentLink; onCode: (invite: TeenParentInvite) => void }) {
  const t = useTranslations('teen.parent.status');
  const renew = useRenewParentInvite();
  const active = link.status === 'active';
  const name = link.displayName?.trim();

  return (
    <section className="nb-card tnp-status" aria-live="polite">
      <IconCircle icon="mother" tone={active ? 'data' : 'brand'} size="lg" />
      <span className="tnp-status-text">
        <b className="tnp-status-title">
          {active ? (name ? t('active', { name }) : t('activeNoName')) : t('invited')}
        </b>
        <span className="tnp-status-sub">{active ? t('activeSub') : t('invitedSub')}</span>
        {renew.isError ? (
          <span className="tnp-error" role="alert">
            {apiText(renew.error, t('renewError'))}
          </span>
        ) : null}
      </span>
      {active ? (
        <StatusPill tone="data" icon="check">
          {t('linked')}
        </StatusPill>
      ) : (
        <SecondaryButton
          block={false}
          variant="text"
          className="tnp-status-btn"
          loading={renew.isPending}
          onClick={() => renew.mutate(link.id, { onSuccess: ({ invite }) => onCode(invite) })}
        >
          {t('renew')}
        </SecondaryButton>
      )}
    </section>
  );
}

/** «فرستادن دعوت»: her mother's number (required — the code is SMSed and bound to it) and an optional name. */
function InviteSheet({
  open,
  grants,
  onClose,
  onCreated,
}: {
  open: boolean;
  grants: TeenGrants;
  onClose: () => void;
  onCreated: (invite: TeenParentInvite) => void;
}) {
  const t = useTranslations('teen.parent.invite');
  const nameId = useId();
  const phoneId = useId();
  const errId = useId();
  const invite = useInviteParent();
  const [name, setName] = useState('');
  const [phone, setPhone] = useState('');
  const [touched, setTouched] = useState(false);
  const mobile = normalizeMobile(phone);
  const phoneError = !phone.trim() ? t('phoneRequired') : mobile === null ? t('phoneInvalid') : null;
  const showError = touched && phoneError !== null;

  const send = () => {
    setTouched(true);
    if (!mobile || invite.isPending) return;
    invite.mutate(
      { phone: mobile, displayName: name, grants },
      {
        onSuccess: ({ invite: created }) => {
          setName('');
          setPhone('');
          setTouched(false);
          onCreated(created);
        },
      },
    );
  };

  return (
    <AppSheet
      open={open}
      onClose={() => (invite.isPending ? undefined : onClose())}
      size="half"
      title={t('sheetTitle')}
      footer={
        <PrimaryButton icon="send" loading={invite.isPending} onClick={send}>
          {t('send')}
        </PrimaryButton>
      }
    >
      <div className="tnp-sheet">
        <p className="tnp-sheet-lead">{t('lead')}</p>
        <div className="cmp-field">
          <label htmlFor={phoneId} className="cmp-field-label">
            {t('phone')}
          </label>
          <input
            id={phoneId}
            className="cmp-input is-phone"
            type="tel"
            inputMode="tel"
            dir="ltr"
            autoComplete="off"
            maxLength={20}
            value={phone}
            placeholder={t('phonePlaceholder')}
            aria-invalid={showError || undefined}
            aria-describedby={showError ? errId : undefined}
            onBlur={() => setTouched(true)}
            onChange={(e) => setPhone(e.target.value)}
          />
          {showError ? (
            <p id={errId} className="tnp-error" role="alert">
              {phoneError}
            </p>
          ) : null}
        </div>
        <div className="cmp-field">
          <label htmlFor={nameId} className="cmp-field-label">
            {t('name')}
          </label>
          <input
            id={nameId}
            className="cmp-input"
            value={name}
            maxLength={100}
            autoComplete="off"
            placeholder={t('namePlaceholder')}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        {invite.isError ? (
          <p className="tnp-error" role="alert">
            {apiText(invite.error, t('error'))}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}

/** The one-time code after an invite or a renew: bloom's code card + whether the SMS went out. */
function CodeSheet({ invite, onClose }: { invite: TeenParentInvite | null; onClose: () => void }) {
  const t = useTranslations('teen.parent.invite');
  // Keep the last code painted while the sheet plays its exit.
  const [shown, setShown] = useState<TeenParentInvite | null>(invite);
  if (invite && invite !== shown) setShown(invite);

  return (
    <AppSheet
      open={invite !== null}
      onClose={onClose}
      size="half"
      title={t('codeTitle')}
      footer={<PrimaryButton onClick={onClose}>{t('done')}</PrimaryButton>}
    >
      <div className="tnp-sheet">
        {shown ? (
          <>
            <p className="tnp-ok" role="status">
              <Icon name="checkCircle" size={16} />
              {shown.smsSent && shown.phone
                ? t.rich('smsSent', { phone: shown.phone, ltr: (chunks) => <bdi dir="ltr">{chunks}</bdi> })
                : t('smsNotSent')}
            </p>
            <InviteCodeCard invite={shown} title={t('codeCardTitle')} lead={t('codeLead')} />
          </>
        ) : null}
      </div>
    </AppSheet>
  );
}

/** «قطع همراهی» / «لغو دعوت» behind a calm confirmation — bloom's DELETE /companions/{id}. */
function RevokeLink({ link }: { link: TeenParentLink }) {
  const t = useTranslations('teen.parent.revoke');
  const revoke = useRevokeCompanion(link.id);
  const forget = useForgetParentLink();
  const [open, setOpen] = useState(false);
  const active = link.status === 'active';

  return (
    <>
      <button type="button" className="tnp-revoke" onClick={() => setOpen(true)}>
        {t(active ? 'cta' : 'cancelInvite')}
      </button>
      <AppSheet
        open={open}
        onClose={() => (revoke.isPending ? undefined : setOpen(false))}
        size="half"
        title={t(active ? 'title' : 'cancelTitle')}
        footer={
          <div className="cmp-sheet-btns">
            <SecondaryButton onClick={() => setOpen(false)} disabled={revoke.isPending}>
              {t('cancel')}
            </SecondaryButton>
            <SecondaryButton
              variant="text"
              className="cmp-danger-text"
              loading={revoke.isPending}
              onClick={() =>
                revoke.mutate(undefined, {
                  onSuccess: () => {
                    setOpen(false);
                    forget(link.id);
                  },
                })
              }
            >
              {t(active ? 'confirm' : 'cancelConfirm')}
            </SecondaryButton>
          </div>
        }
      >
        <p className="cmp-sheet-body">{t(active ? 'body' : 'cancelBody')}</p>
        {revoke.isError ? (
          <p className="tnp-error" role="alert">
            {apiText(revoke.error, t('error'))}
          </p>
        ) : null}
      </AppSheet>
    </>
  );
}
