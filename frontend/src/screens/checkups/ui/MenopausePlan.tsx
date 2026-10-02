'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { type CheckupItem, checkupIcon, formatCheckupMonth, useCheckups } from '@/entities/checkup';
import { type Locale, Link } from '@/shared/i18n';
import { Icon, Skeleton, SkeletonGroup } from '@/shared/ui';

import { useMenoCheckupGroups, useMenoCheckupsIntro } from '../api/meno';
import { MENO_CHIP_TONE, groupForMenopause, menoCadence, menoChip, menoWhen } from '../model/meno';

/**
 * The checkups body in menopause mode (CB-MENO-09, nbl_Meno_Checkups): the
 * `checkups_intro` note, then her plan (already audience-filtered by the API)
 * in the `meno_checkup_groups` sections with the board's chips. Rows open the
 * usual checkup detail. «افزودن نتیجه آزمایش» waits for the lab upload screen
 * (bloom B-N6-07) — not rendered until that route exists.
 */
export function MenopausePlan() {
  const t = useTranslations('checkups');
  const locale = useLocale();
  const plan = useCheckups('all');
  const groups = useMenoCheckupGroups(locale);
  const intro = useMenoCheckupsIntro(locale);

  // Groups are content: if they fail the rows still show, in one section.
  const sections = plan.data ? groupForMenopause(plan.data.items, groups.data ?? []) : [];
  const loading = plan.isPending || (groups.isPending && groups.fetchStatus !== 'idle');

  return (
    <>
      {intro.data ? (
        <p className="nb-card mck-intro">
          <Icon name="shield" size={18} className="mck-intro-icon" />
          <span>{intro.data}</span>
        </p>
      ) : null}

      {loading ? (
        <SkeletonGroup label={t('loading')} className="rmd-form-skel">
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      ) : plan.isError ? (
        <div className="rmd-state" role="alert">
          <p>{t('loadError')}</p>
          <button type="button" className="rmd-retry" onClick={() => void plan.refetch()}>
            {t('retry')}
          </button>
        </div>
      ) : (
        sections.map((section) => (
          <section key={section.code} className="mck-sec" aria-labelledby={`mck-sec-${section.code}`}>
            <h2 id={`mck-sec-${section.code}`} className="mck-sec-title">
              {section.title ?? t('meno.more')}
            </h2>
            <ul className="nb-card mck-list">
              {section.items.map((item) => (
                <MenoRow key={item.id} item={item} />
              ))}
            </ul>
          </section>
        ))
      )}
    </>
  );
}

function MenoRow({ item }: { item: CheckupItem }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const chip = menoChip(item);
  const when = menoWhen(item);
  const whenText =
    when.kind === 'never'
      ? t('meno.never')
      : t(`meno.${when.kind}`, { date: formatCheckupMonth(when.date, locale) });
  const meta = [menoCadence(item), whenText].filter(Boolean).join(t('meno.separator'));

  return (
    <li>
      <Link href={`/checkups/${item.id}`} className={clsx(`ck-tone-${item.tone}`, 'mck-row')}>
        <span className="mck-row-icon" aria-hidden>
          <Icon name={checkupIcon(item.icon, { category: item.category })} size={20} />
        </span>
        <span className="mck-row-text">
          <span className="mck-row-title">{item.title}</span>
          <span className="mck-row-meta">{meta}</span>
        </span>
        <span className={clsx('mck-chip', `nb-tone-${MENO_CHIP_TONE[chip]}`)}>{t(`meno.chips.${chip}`)}</span>
      </Link>
    </li>
  );
}
