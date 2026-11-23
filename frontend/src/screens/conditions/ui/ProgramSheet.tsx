'use client';

import { useLocale, useTranslations } from 'next-intl';

import { CONDITION_LOOK, CONDITION_PATH, useEnrolCondition, useLeaveCondition } from '@/entities/condition';
import { getApiSaveErrorMessage } from '@/shared/api';
import { Link, type Locale } from '@/shared/i18n';
import { formatLongDate, fromApiDate } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import { AppSheet } from '@/shared/sheet';
import { IconCircle, PrimaryButton, SecondaryButton } from '@/shared/ui';

import type { ProgramCard } from '../model/cards';
import '@/shared/ui/styles/cnd.css';

/**
 * One program: what it logs and whether she is in it. Not in it → «شروع برنامه» (POST /conditions/enrolments); in
 * it → open the diary (PCOS: the cycle analysis) or leave (DELETE …, her entries stay). The sheet stays open after a
 * change so the next step — opening the diary — is one tap away.
 */
export function ProgramSheet({ card, onClose }: { card: ProgramCard; onClose: () => void }) {
  const t = useTranslations('conditions');
  const locale = useLocale() as Locale;
  const enrol = useEnrolCondition();
  const leave = useLeaveCondition();
  const look = CONDITION_LOOK[card.code];
  const title = card.title ?? t(`programs.${card.code}.title`);
  const sub = card.sub ?? t(`programs.${card.code}.sub`);
  const logs = card.logs ?? t(`programs.${card.code}.logs`);
  const busy = enrol.isPending || leave.isPending;
  const failed = enrol.isError ? enrol.error : leave.isError ? leave.error : null;

  const run = (write: typeof enrol) => {
    enrol.reset();
    leave.reset();
    write.mutate(card.code);
  };

  return (
    <AppSheet open size="half" onClose={onClose} title={title}>
      <div className="cnd-sheet">
        <div className="cnd-sheet-head">
          <IconCircle icon={look.icon} tone={look.tone} size="lg" />
          <div className="cnd-sheet-titles">
            <p className="cnd-sheet-sub">{sub}</p>
            <p className={card.enrolled ? 'cnd-sheet-state' : 'cnd-sheet-state is-off'}>
              {card.enrolled
                ? card.enrolledOn
                  ? t('sheet.activeSince', { date: formatLongDate(fromApiDate(card.enrolledOn), locale) })
                  : t('hub.active')
                : t('sheet.inactive')}
            </p>
          </div>
        </div>
        <p className="cnd-sheet-logs">{t('hub.logs', { logs })}</p>
        {card.code === 'pcos' ? <p className="cnd-sheet-hint">{t('sheet.pcosHint')}</p> : null}
        <p className="cnd-sheet-hint">{t(card.enrolled ? 'sheet.leaveHint' : 'sheet.joinHint')}</p>
        {failed ? (
          <p className="cnd-sheet-error" role="alert">
            {getApiSaveErrorMessage(failed, t('sheet.saveError'))}
          </p>
        ) : null}
        <div className="cnd-sheet-actions">
          {card.enrolled ? (
            <>
              <Link href={CONDITION_PATH[card.code]} className="nb-btn is-primary is-block" {...pilot(`programs.open:${card.code}`)}>
                {t(`sheet.open.${card.code}`)}
              </Link>
              <SecondaryButton variant="text" loading={leave.isPending} disabled={busy} onClick={() => run(leave)}>
                {t('sheet.leave')}
              </SecondaryButton>
            </>
          ) : (
            <PrimaryButton loading={enrol.isPending} disabled={busy} onClick={() => run(enrol)} {...pilot(`programs.join:${card.code}`)}>
              {t('sheet.join')}
            </PrimaryButton>
          )}
        </div>
      </div>
    </AppSheet>
  );
}
