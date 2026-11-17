'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  companionName,
  FamilyStrip,
  familySpouse,
  hoursUntil,
  type OwnerCompanion,
  PersonBubble,
  useCompanions,
} from '@/entities/companion';
import { useUserProfile } from '@/entities/user';
import { DisconnectSheet, useChildNames } from '@/features/invite-companion';
import { pilot } from '@/shared/pilot';
import { type Locale, useDirection, useRouter } from '@/shared/i18n';
import { diffInDays, formatLongDate, formatNumber, today } from '@/shared/lib/date';
import {
  EmptyState,
  Icon,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

/** Up to this many days the row says «از N روز پیش»; older links show the date. */
const RELATIVE_DAYS = 30;

/**
 * «همراهان من» (`/companions`, companion-v2 `manage`): the invited and active
 * companions (since, status, goal), «افزودن همراه جدید», a per-row «قطع
 * اشتراک» behind the confirm sheet and, once cut, the calm `disconnected`
 * state inline on the same route. The family strip (a spouse) stays.
 */
export function CompanionsPage() {
  const t = useTranslations('companions');
  const router = useRouter();
  const query = useCompanions();
  const profile = useUserProfile();
  const companions = query.data ?? [];
  const spouse = familySpouse(companions);
  const kids = useChildNames(spouse?.family?.sharedChildIds ?? []);
  // The row whose «قطع اشتراک» sheet is open, then the name shown on the success state.
  const [cutting, setCutting] = useState<{ id: number; name: string } | null>(null);
  const [disconnected, setDisconnected] = useState<string | null>(null);

  if (disconnected !== null) {
    return (
      <div className="view cmp-page">
        <SkyLayer />
        <div className="scroll cmp-scroll is-form">
          <ScreenHeader
            title=""
            backIcon="close"
            backLabel={t('disconnect.close')}
            onBack={() => setDisconnected(null)}
          />
          <div className="cmp-body cmp2-center cmp2-success">
            <span className="cmp2-success-badge" aria-hidden>
              <Icon name="check" size={40} strokeWidth={2.4} />
            </span>
            <h2 className="cmp2-title" role="status">
              {t('disconnect.doneTitle')}
            </h2>
            <p className="cmp-lead">{t('disconnect.doneBody', { name: disconnected })}</p>
          </div>
          <div className="cmp-footer">
            <PrimaryButton onClick={() => setDisconnected(null)}>{t('disconnect.doneBack')}</PrimaryButton>
          </div>
        </div>
      </div>
    );
  }

  const add = () => router.push('/companions/new');

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('list.loading')} className="cmp-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
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
  } else if (companions.length === 0) {
    body = (
      <>
        <EmptyState icon="users" title={t('list.emptyTitle')} body={t('list.emptyBody')} className="cmp-empty" />
        <PrimaryButton icon="plus" onClick={add} {...pilot('companion.add')}>
          {t('list.add')}
        </PrimaryButton>
      </>
    );
  } else {
    const spouseName = spouse ? (companionName(spouse) ?? t('unnamed')) : '';
    let familyNote: string;
    if (spouse?.status === 'invited') familyNote = t('family.pending', { name: spouseName });
    else if (kids.length) {
      familyNote = t('family.childrenShared', {
        names: kids.map((k) => k.name).join(t('listSeparator')),
        count: kids.length,
      });
    } else familyNote = t('family.childrenNone');
    body = (
      <>
        <section className="cmp2-sec" aria-labelledby="cmp2-active-title">
          <h3 id="cmp2-active-title" className="cmp2-label">
            {t('list.active')}
          </h3>
          <ul className="cmp-list">
            {companions.map((c) => (
              <li key={c.id}>
                <CompanionRow
                  companion={c}
                  onOpen={() => router.push(`/companions/${c.id}`)}
                  onCut={() => setCutting({ id: c.id, name: companionName(c) ?? t('unnamed') })}
                />
              </li>
            ))}
          </ul>
        </section>
        <SecondaryButton icon="plus" onClick={add} {...pilot('companion.add')}>
          {t('list.add')}
        </SecondaryButton>
        <p className="cmp2-privacy">
          <Icon name="info" size={16} className="cmp2-privacy-icon" />
          <span>{t('list.note')}</span>
        </p>
        {spouse ? (
          <FamilyStrip selfName={profile.data?.name ?? null} companionName={spouseName} kids={kids} note={familyNote} />
        ) : null}
      </>
    );
  }

  return (
    <div className="view cmp-page">
      <SkyLayer />
      <div className="scroll cmp-scroll">
        <ScreenHeader title={t('list.title')} onBack={() => router.push('/profile')} backLabel={t('back')} />
        <div className="cmp-body">
          {body}
        </div>
      </div>
      <DisconnectSheet
        companionId={cutting?.id ?? null}
        name={cutting?.name ?? ''}
        onClose={() => setCutting(null)}
        onDisconnected={() => {
          setDisconnected(cutting?.name ?? t('unnamed'));
          setCutting(null);
        }}
      />
    </div>
  );
}

interface CompanionRowProps {
  companion: OwnerCompanion;
  onOpen: () => void;
  onCut: () => void;
}

/** One «همراهان فعال» row: initial, name, «از N روز پیش • فعال · <goal>», chevron to the detail, «قطع اشتراک». */
function CompanionRow({ companion, onOpen, onCut }: CompanionRowProps) {
  const t = useTranslations('companions');
  const locale = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const name = companionName(companion) ?? t('unnamed');
  const active = companion.status === 'active';

  let when: string | null = null;
  if (active && companion.acceptedAt) {
    const at = new Date(companion.acceptedAt);
    const days = Math.max(0, diffInDays(today(), at));
    when =
      days <= RELATIVE_DAYS
        ? t('list.since', { days, n: formatNumber(days, locale) })
        : t('list.sinceDate', { date: formatLongDate(at, locale) });
  } else if (!active && companion.invite) {
    const hours = hoursUntil(companion.invite.expiresAt);
    when = hours > 0 ? t('list.expiresIn', { hours: formatNumber(hours, locale) }) : t('list.expired');
  }
  const goals = companion.preferences.goals;

  return (
    <div className="nb-card cmp2-row">
      <button type="button" className="cmp2-row-main" onClick={onOpen} {...pilot(`companion.link:${companion.id}`)}>
        <PersonBubble name={companionName(companion)} tone="companion" />
        <span className="cmp2-row-text">
          <b className="cmp2-row-name">{name}</b>
          <span className="cmp2-row-meta">
            <span className={active ? 'cmp2-dot is-active' : 'cmp2-dot is-invited'} aria-hidden />
            {when ? t('list.meta', { status: t(`status.${companion.status}`), when }) : t(`status.${companion.status}`)}
          </span>
          {goals.length ? (
            <span className="cmp2-row-goals">
              {goals.map((goal) => (
                <span key={goal} className="cmp2-row-goal">
                  {t(`goals.${goal}.title`)}
                </span>
              ))}
            </span>
          ) : null}
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="cmp-card-chev" />
      </button>
      <button type="button" className="cmp2-row-cut" onClick={onCut}>
        {active ? t('list.cut') : t('list.cancelInvite')}
      </button>
    </div>
  );
}
