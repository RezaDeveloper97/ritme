'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { useMedications } from '@/entities/care-reminder';
import { useToggleMedicationActive } from '@/features/manage-medication';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon, Skeleton, SkeletonGroup, Switch } from '@/shared/ui';

import { type MedicationRowState, medicationRowState, sectionStatus, sortMedications } from '../model/view';
import { SectionHead } from './SectionHead';

type T = ReturnType<typeof useTranslations<'care'>>;

function rowMeta(row: MedicationRowState, t: T, locale: Locale): string {
  const schedule =
    row.schedule.kind === 'everyDay'
      ? t('medications.everyDay')
      : row.schedule.kind === 'everyOtherDay'
        ? t('medications.everyOtherDay')
        : t('medications.daysPerWeekCount', {
            count: row.schedule.count,
            days: formatNumber(row.schedule.count, locale),
          });
  const time = row.slots
    .map((s) =>
      t('slotWithPeriod', { time: formatNumber(s.clock, locale), period: t(`slotPeriod.${s.period}`) }),
    )
    .join(t('listSeparator'));
  return t('medications.rowMeta', {
    schedule,
    time,
    amount: formatNumber(row.amount, locale),
    form: t(`medicationForm.forms.${row.form}`),
  });
}

/**
 * «داروها و مکمل‌ها» — every medication with its schedule and the active
 * switch (`manage-medication`). The row body opens the edit form (T-M3-07);
 * the switch is a sibling, not nested in the link, so each stays one control.
 */
export function MedicationSection() {
  const t = useTranslations('care');
  const locale = useLocale() as Locale;
  const query = useMedications();
  const toggle = useToggleMedicationActive();
  const status = sectionStatus(query);

  return (
    <section className="rmd-sec" aria-labelledby="rmd-meds-title">
      <SectionHead
        id="rmd-meds-title"
        title={t('medications.title')}
        addHref="/reminders/medication/new"
        addText={t('add')}
        addLabel={t('medications.add')}
      />

      <div className="rmd-list">
        {status === 'loading' && (
          <SkeletonGroup label={t('loading')} className="rmd-state">
            <Skeleton shape="block" className="rmd-row-skel" />
            <Skeleton shape="block" className="rmd-row-skel" />
          </SkeletonGroup>
        )}
        {status === 'error' && (
          <div className="rmd-state">
            <p className="rmd-empty">{t('loadError')}</p>
            <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
              {t('retry')}
            </button>
          </div>
        )}
        {status === 'empty' && <p className="rmd-empty rmd-state">{t('medications.empty')}</p>}
        {status === 'ready' &&
          sortMedications(query.data ?? []).map(medicationRowState).map((row) => {
            const name = row.dose
              ? t('medications.nameWithDose', { title: row.title, dose: row.dose })
              : row.title;
            // While the switch's write is in flight, show where it is going.
            const pending = toggle.isPending && toggle.variables?.id === row.id;
            const on = pending && toggle.variables ? toggle.variables.isActive : row.isActive;
            return (
              <div key={row.id} className="rmd-row">
                <Link href={`/reminders/medication/${row.id}`} className="rmd-row-link">
                  <span className={clsx('rmd-tile', `is-${row.tone}`)} aria-hidden>
                    <Icon name={row.icon} size={20} strokeWidth={1.8} />
                  </span>
                  <span className="rmd-row-body">
                    <span className="rmd-row-title">{name}</span>
                    <span className="rmd-row-meta">{rowMeta(row, t, locale)}</span>
                  </span>
                </Link>
                <Switch
                  compact
                  className="rmd-hit"
                  checked={on}
                  label={t('medications.toggle', { title: row.title })}
                  disabled={pending}
                  onCheckedChange={(next) => toggle.mutate({ id: row.id, isActive: next })}
                />
              </div>
            );
          })}
      </div>
    </section>
  );
}
