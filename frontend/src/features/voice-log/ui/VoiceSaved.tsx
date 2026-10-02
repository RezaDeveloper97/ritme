'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { Locale } from '@/shared/i18n';
import { formatDecimal } from '@/shared/lib/date';
import { Icon, type IconName, ListGroup, ListRow, PrimaryButton, Switch, type Tone } from '@/shared/ui';

import { isNightlyReminder, useDailyLogReminder, useSetNightlyReminder } from '../api/reminder';
import { isDiaryTarget, type SavedRow, savedRowText } from '../model/review';
import { diaryLook } from './look';

interface VoiceSavedProps {
  rows: readonly SavedRow[];
  iconOf: (category: string) => IconName;
  toneOf: (category: string) => Tone;
  onDone: () => void;
}

/**
 * nbl_Voice_Saved: check disc, «N مورد ثبت شد», the saved items (server chip labels split into title ·
 * caption), «دفعه بعد سریع‌تر» — the nightly 21:00 `daily_log` reminder switch — and «برگشت به خانه».
 */
export function VoiceSaved({ rows, iconOf, toneOf, onDone }: VoiceSavedProps) {
  const t = useTranslations('voiceLog.saved');
  const locale = useLocale() as Locale;
  const reminder = useDailyLogReminder();
  const setReminder = useSetNightlyReminder();
  const on = isNightlyReminder(reminder.data);
  return (
    <div className="vlog vlog-saved">
      <div className="vlog-saved-head">
        <span className="vlog-saved-check" aria-hidden>
          <Icon name="check" size={40} strokeWidth={2.2} />
        </span>
        <h3 className="vlog-saved-title">{t('title', { count: rows.length })}</h3>
        <p className="vlog-saved-body">{t('body')}</p>
      </div>

      {rows.length ? (
        <ListGroup className="vlog-saved-list">
          {rows.map((row) => {
            const { title, sub } = savedRowText({ ...row, label: formatDecimal(row.label, locale) });
            const look = isDiaryTarget(row.category) ? diaryLook(row.category) : { icon: iconOf(row.category), tone: toneOf(row.category) };
            return <ListRow key={row.key} icon={look.icon} iconTone={look.tone} title={title} description={sub ?? undefined} />;
          })}
        </ListGroup>
      ) : null}

      <section className="nb-card vlog-remind" aria-labelledby="vlog-remind-title">
        <div className="vlog-remind-text">
          <h4 id="vlog-remind-title" className="vlog-remind-title">
            {t('reminderTitle')}
          </h4>
          <p id="vlog-remind-body" className="vlog-remind-body">
            {t('reminderBody')}
          </p>
          {setReminder.isError ? (
            <p className="vlog-error" role="alert">
              {t('reminderError')}
            </p>
          ) : null}
        </div>
        <Switch
          checked={on}
          onCheckedChange={(next) => setReminder.mutate(next)}
          labelledBy="vlog-remind-title"
          describedBy="vlog-remind-body"
          disabled={reminder.isPending || reminder.isError}
        />
      </section>

      <div className="vlog-cta">
        <PrimaryButton onClick={onDone}>{t('done')}</PrimaryButton>
      </div>
    </div>
  );
}
