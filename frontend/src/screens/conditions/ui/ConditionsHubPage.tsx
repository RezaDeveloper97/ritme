'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  CONDITION_LOOK,
  ConditionNudgeCard,
  type ConditionProgram,
  useConditionCatalog,
  useConditionNudges,
  useConditions,
} from '@/entities/condition';
import { PELVIC_PATH } from '@/entities/pelvic';
import { Link, useDirection, useRouter } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import {
  EmptyState,
  Icon,
  IconCircle,
  InfoNote,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { notADiagnosis, programCards, type ProgramCard } from '../model/cards';
import { ProgramSheet } from './ProgramSheet';

/**
 * «برنامه‌های مراقبت» (`/programs`, CB-COND-02, nbl_Cond_Hub): the condition programs with what each one logs and
 * whether she is in it, this cycle's heavy-pain / heavy-flow suggestion (GET /messages/nudges) as a soft card, and the
 * «not a diagnosis» note. A card opens a sheet to start / leave the program or open its diary; PCOS opens the cycle
 * analysis. Reached from the Services «برنامه‌های مراقبتی» tiles; a services hub, so the bottom nav shows (خدمات).
 */
export function ConditionsHubPage() {
  const t = useTranslations('conditions.hub');
  const locale = useLocale();
  const router = useRouter();
  const overview = useConditions();
  const programsCopy = useConditionCatalog('condition_programs', locale);
  const alerts = useConditionCatalog('condition_alerts', locale);
  const nudges = useConditionNudges();
  const [open, setOpen] = useState<ConditionProgram | null>(null);

  const cards = overview.data ? programCards(overview.data, programsCopy.data) : [];
  const openCard = cards.find((c) => c.code === open) ?? null;
  const note = notADiagnosis(alerts.data);

  let body;
  if (overview.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="cnd-list">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} shape="card" className="cnd-skel" />
        ))}
      </SkeletonGroup>
    );
  } else if (overview.isError) {
    body = (
      <EmptyState
        icon="warning"
        title={t('loadError')}
        action={
          <SecondaryButton icon="refresh" block={false} loading={overview.isFetching} onClick={() => void overview.refetch()}>
            {t('retry')}
          </SecondaryButton>
        }
      />
    );
  } else if (!cards.length) {
    body = <EmptyState icon="grid" title={t('empty')} />;
  } else {
    body = (
      <ul className="cnd-list" aria-label={t('listLabel')}>
        {cards.map((card) => (
          <li key={card.code}>
            <ProgramCardButton card={card} onOpen={() => setOpen(card.code)} />
          </li>
        ))}
      </ul>
    );
  }

  return (
    <div className="view cnd-page">
      <SkyLayer />
      <div className="scroll cnd-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/services')} backLabel={t('back')} />
        <div className="cnd-body">
          <div className="cnd-intro">
            <h2 className="cnd-heading">{t('heading')}</h2>
            <p className="cnd-lead">{t('intro')}</p>
          </div>
          {nudges.data?.map((nudge) => <ConditionNudgeCard key={nudge.key} nudge={nudge} />)}
          {body}
          <PelvicCard />
          <InfoNote className="cnd-note">
            {note ? (note.body ? t('noteJoin', { title: note.title, body: note.body }) : note.title) : t('note')}
          </InfoNote>
        </div>
      </div>
      {openCard ? <ProgramSheet key={openCard.code} card={openCard} onClose={() => setOpen(null)} /> : null}
      <BottomNav />
    </div>
  );
}

function ProgramCardButton({ card, onOpen }: { card: ProgramCard; onOpen: () => void }) {
  const t = useTranslations('conditions');
  const rtl = useDirection() === 'rtl';
  const look = CONDITION_LOOK[card.code];
  const title = card.title ?? t(`programs.${card.code}.title`);
  const sub = card.sub ?? t(`programs.${card.code}.sub`);
  const logs = card.logs ?? t(`programs.${card.code}.logs`);
  return (
    <button
      type="button"
      className={clsx('nb-card cnd-card', card.enrolled && 'is-active')}
      aria-haspopup="dialog"
      aria-label={t('hub.openLabel', { title, state: t(card.enrolled ? 'hub.stateActive' : 'hub.stateInactive') })}
      onClick={onOpen}
      {...pilot(`programs.card:${card.code}`)}
    >
      <span className="cnd-card-head">
        <IconCircle icon={look.icon} tone={look.tone} size="lg" />
        <span className="cnd-card-titles">
          <b className="cnd-card-title">{title}</b>
          <span className="cnd-card-sub">{sub}</span>
        </span>
        {card.enrolled ? (
          <StatusPill tone="data" className="cnd-card-pill">
            {t('hub.active')}
          </StatusPill>
        ) : (
          <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} strokeWidth={2} className="cnd-card-chev" />
        )}
      </span>
      <span className="cnd-card-logs">{t('hub.logs', { logs })}</span>
    </button>
  );
}

/** CB-PELV-02: the pelvic floor program (its own API, not an enrolment) — a card straight to its plan. */
function PelvicCard() {
  const t = useTranslations('conditions');
  const rtl = useDirection() === 'rtl';
  const title = t('pelvicCard.title');
  return (
    <Link href={PELVIC_PATH} className="nb-card cnd-card" aria-label={t('pelvicCard.openLabel', { title })} {...pilot('programs.pelvic-card')}>
      <span className="cnd-card-head">
        <IconCircle icon="target" tone="data" size="lg" />
        <span className="cnd-card-titles">
          <b className="cnd-card-title">{title}</b>
          <span className="cnd-card-sub">{t('pelvicCard.sub')}</span>
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} strokeWidth={2} className="cnd-card-chev" />
      </span>
      <span className="cnd-card-logs">{t('hub.logs', { logs: t('pelvicCard.logs') })}</span>
    </Link>
  );
}
