'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  type SideEffectCode,
  type TreatmentItem,
  type TreatmentItemInput,
  type TreatmentKind,
  type TreatmentScreen,
  type TreatmentTip,
  useSaveSideEffects,
  useTreatment,
  useTreatmentIntake,
} from '@/entities/menopause';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatMonthLabel, formatNumber, fromApiDate, toParts } from '@/shared/lib/date';
import {
  Card,
  ChipGroup,
  EmptyState,
  Icon,
  IconCircle,
  InfoNote,
  PillChip,
  ScreenHeader,
  SecondaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  WeekDots,
} from '@/shared/ui';

import {
  dotsSummary,
  earliestStart,
  groupWeekDots,
  inLocaleOrder,
  itemLook,
  lifestyleSuggestions,
  lifestyleTipOf,
  tipBody,
  todayAmount,
  weekDates,
} from '../model/treatment';
import { ItemSheet, MinutesSheet } from './ItemSheet';

/** What the add / edit sheet is showing. */
type SheetState =
  | { mode: 'add'; kind: TreatmentKind; preset?: Partial<TreatmentItemInput> }
  | { mode: 'edit'; item: TreatmentItem }
  | null;

/** «ماه سال» of an API date in the locale's calendar. */
function monthYear(date: string, locale: Locale): string {
  const p = toParts(fromApiDate(date), locale);
  return formatMonthLabel(p.year, p.month, locale);
}

/**
 * «درمان و مراقبت» (`/menopause/treatment`, CB-MENO-10, nbl_Meno_Treatment):
 * HRT with today's intake toggle, the week's dots and the doctor review,
 * side-effect chips for today with the spotting note, supplements, lifestyle
 * goals with this week's progress, items she stopped, and the «only with your
 * doctor» note. Every item is added / edited / stopped in a sheet. The app
 * never prescribes: names and doses are hers. Back header, no bottom nav.
 */
export function MenopauseTreatmentPage() {
  const t = useTranslations('menopause.treatment');
  const router = useRouter();
  const query = useTreatment();
  const [sheet, setSheet] = useState<SheetState>(null);
  const [minutesFor, setMinutesFor] = useState<TreatmentItem | null>(null);

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="mtr-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="pill"
        title={t('loadError')}
        action={
          <SecondaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </SecondaryButton>
        }
      />
    );
  } else {
    body = (
      <Content
        screen={query.data}
        onAdd={(kind, preset) => setSheet({ mode: 'add', kind, preset })}
        onEdit={(item) => setSheet({ mode: 'edit', item })}
        onMinutes={setMinutesFor}
      />
    );
  }

  return (
    <div className="view mtr-page">
      <SkyLayer />
      <div className="scroll mtr-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/home')} backLabel={t('back')} />
        {body}
      </div>
      {sheet ? (
        <ItemSheet
          key={sheet.mode === 'edit' ? `e${sheet.item.id}` : `a${sheet.kind}`}
          kind={sheet.mode === 'edit' ? sheet.item.kind : sheet.kind}
          item={sheet.mode === 'edit' ? sheet.item : null}
          preset={sheet.mode === 'add' ? sheet.preset : undefined}
          today={query.data?.date ?? null}
          onClose={() => setSheet(null)}
        />
      ) : null}
      {minutesFor && query.data ? (
        <MinutesSheet item={minutesFor} today={query.data.date} onClose={() => setMinutesFor(null)} />
      ) : null}
    </div>
  );
}

function Content({
  screen,
  onAdd,
  onEdit,
  onMinutes,
}: {
  screen: TreatmentScreen;
  onAdd: (kind: TreatmentKind, preset?: Partial<TreatmentItemInput>) => void;
  onEdit: (item: TreatmentItem) => void;
  onMinutes: (item: TreatmentItem) => void;
}) {
  const t = useTranslations('menopause.treatment');
  const { hrt, supplement, lifestyle } = screen.items;
  const hasHrt = hrt.length > 0;
  return (
    <div className="mtr-body">
      <HrtSection screen={screen} onAdd={() => onAdd('hrt')} onEdit={onEdit} />
      {screen.review && hasHrt ? <ReviewCard screen={screen} /> : null}
      {hasHrt ? <SideEffects screen={screen} /> : null}

      <section className="mtr-sec" aria-labelledby="mtr-supplement">
        <SectionTitle id="mtr-supplement" title={t('kinds.supplement')} actionLabel={t('add')} onAction={() => onAdd('supplement')} />
        {supplement.length ? (
          <Card className="mtr-list" padding="none">
            {supplement.map((item, i) => (
              <MedicineRow key={item.id} item={item} index={i} today={screen.date} onEdit={onEdit} />
            ))}
          </Card>
        ) : (
          <p className="mtr-empty">{t('empty.supplement')}</p>
        )}
      </section>

      <LifestyleSection screen={screen} items={lifestyle} onAdd={onAdd} onEdit={onEdit} onMinutes={onMinutes} />

      {screen.stopped.length ? <StoppedSection items={screen.stopped} onEdit={onEdit} /> : null}

      <InfoNote className="mtr-note">{tipBody(screen.tips, 'doctor_only') ?? t('doctorOnly')}</InfoNote>
    </div>
  );
}

function HrtSection({ screen, onAdd, onEdit }: { screen: TreatmentScreen; onAdd: () => void; onEdit: (item: TreatmentItem) => void }) {
  const t = useTranslations('menopause.treatment');
  const locale = useLocale() as Locale;
  const items = screen.items.hrt;
  const start = earliestStart(items);
  const dates = weekDates(screen.week.from, items);
  const dots = groupWeekDots(items, dates, screen.date);
  const summary = dotsSummary(dots);
  const todayIndex = inLocaleOrder(dates, locale).indexOf(screen.date);
  return (
    <section className="mtr-sec" aria-labelledby="mtr-hrt">
      <div className="mtr-sect-head">
        <SectionTitle id="mtr-hrt" title={t('kinds.hrt')} actionLabel={t('add')} onAction={onAdd} />
        {start ? <p className="mtr-sect-sub">{t('startedOn', { date: monthYear(start, locale) })}</p> : null}
      </div>
      {items.length ? (
        <Card className="mtr-list" padding="none">
          {items.map((item, i) => (
            <MedicineRow key={item.id} item={item} index={i} today={screen.date} onEdit={onEdit} />
          ))}
          <div className="mtr-week">
            <div className="mtr-week-head">
              <h3 className="mtr-week-title">{t('thisWeek')}</h3>
              <span className="mtr-week-sum">
                {t('weekDays', { taken: formatNumber(summary.done, locale), days: formatNumber(summary.days, locale) })}
              </span>
            </div>
            <WeekDots
              days={inLocaleOrder(dots, locale)}
              locale={locale}
              label={t('thisWeek')}
              stateLabels={{ done: t('dot.done'), missed: t('dot.missed'), future: t('dot.future') }}
              todayIndex={todayIndex >= 0 ? todayIndex : undefined}
            />
          </div>
        </Card>
      ) : (
        <p className="mtr-empty">{t('empty.hrt')}</p>
      )}
    </section>
  );
}

function MedicineRow({
  item,
  index,
  today,
  onEdit,
}: {
  item: TreatmentItem;
  index: number;
  today: string;
  onEdit: (item: TreatmentItem) => void;
}) {
  const t = useTranslations('menopause.treatment');
  const intake = useTreatmentIntake();
  const look = itemLook(item.kind, index);
  const sub = [item.schedule ? t(`schedule.${item.schedule}`) : null, item.dose ?? t('asPrescribed')].filter(Boolean).join(' · ');
  const taken = item.takenToday;
  return (
    <div className="mtr-row">
      <button type="button" className="mtr-row-main" onClick={() => onEdit(item)} aria-label={t('editA11y', { name: item.name })}>
        <IconCircle icon={look.icon} tone={look.tone} size="md" />
        <span className="mtr-row-text">
          <b className="mtr-row-title">{item.name}</b>
          <span className="mtr-row-sub">{sub}</span>
        </span>
      </button>
      <button
        type="button"
        className={clsx('mtr-take', taken ? 'is-taken' : 'is-open')}
        aria-pressed={taken}
        aria-label={t(taken ? 'untakeA11y' : 'takeA11y', { name: item.name })}
        disabled={intake.isPending}
        onClick={() => intake.mutate({ id: item.id, date: today, taken: !taken })}
      >
        {taken ? t('taken') : t('log')}
      </button>
      {intake.isError ? (
        <p className="mtr-row-error" role="alert">
          {t('error')}
        </p>
      ) : null}
    </div>
  );
}

function ReviewCard({ screen }: { screen: TreatmentScreen }) {
  const t = useTranslations('menopause.treatment');
  const locale = useLocale() as Locale;
  const review = screen.review;
  if (!review) return null;
  return (
    <Card className="mtr-review" padding="sm">
      <IconCircle icon="clock" tone="bloom" size="md" />
      <span className="mtr-row-text">
        <b className="mtr-row-title">
          {t('review.title', { date: monthYear(review.on, locale) })}
          {review.suggested ? <span className="mtr-review-tag">{t('review.suggested')}</span> : null}
        </b>
        <span className="mtr-review-body">{tipBody(screen.tips, 'hrt_review') ?? t('review.body')}</span>
      </span>
    </Card>
  );
}

function SideEffects({ screen }: { screen: TreatmentScreen }) {
  const t = useTranslations('menopause.treatment.sideEffects');
  const save = useSaveSideEffects();
  // Optimistic set while the PUT runs, so taps in a row don't drop each other.
  const [pending, setPending] = useState<SideEffectCode[] | null>(null);
  const selected = pending ?? screen.sideEffects.today;
  const toggle = (code: SideEffectCode, on: boolean) => {
    const next = on ? [...selected.filter((c) => c !== code), code] : selected.filter((c) => c !== code);
    setPending(next);
    save.mutate({ date: screen.date, codes: next }, { onSettled: () => setPending(null) });
  };
  return (
    <section className="mtr-sec" aria-labelledby="mtr-side">
      <SectionTitle id="mtr-side" title={t('title')} />
      <Card className="mtr-side">
        <ChipGroup label={t('label')}>
          {screen.sideEffects.codes.map((code) => (
            <PillChip key={code} mode="multi" tone="bloom" pressed={selected.includes(code)} onPressedChange={(on) => toggle(code, on)}>
              {t(`codes.${code}`)}
            </PillChip>
          ))}
        </ChipGroup>
        <p className="mtr-side-note">{tipBody(screen.tips, 'hrt_spotting') ?? t('spottingNote')}</p>
        {save.isError ? (
          <p className="mtr-row-error" role="alert">
            {t('error')}
          </p>
        ) : null}
      </Card>
    </section>
  );
}

function LifestyleSection({
  screen,
  items,
  onAdd,
  onEdit,
  onMinutes,
}: {
  screen: TreatmentScreen;
  items: TreatmentItem[];
  onAdd: (kind: TreatmentKind, preset?: Partial<TreatmentItemInput>) => void;
  onEdit: (item: TreatmentItem) => void;
  onMinutes: (item: TreatmentItem) => void;
}) {
  const t = useTranslations('menopause.treatment');
  const suggestions = lifestyleSuggestions(items, screen.tips);
  return (
    <section className="mtr-sec" aria-labelledby="mtr-lifestyle">
      <SectionTitle id="mtr-lifestyle" title={t('kinds.lifestyle')} actionLabel={t('add')} onAction={() => onAdd('lifestyle')} />
      {items.length ? (
        <Card className="mtr-list" padding="none">
          {items.map((item, i) => (
            <LifestyleRow key={item.id} item={item} index={i} screen={screen} onEdit={onEdit} onMinutes={onMinutes} />
          ))}
        </Card>
      ) : (
        <p className="mtr-empty">{t('empty.lifestyle')}</p>
      )}
      {suggestions.length ? (
        <div className="mtr-suggest">
          <h3 className="mtr-suggest-title">{t('lifestyle.suggestions')}</h3>
          <Card className="mtr-list" padding="none">
            {suggestions.map((tip, i) => (
              <SuggestionRow
                key={tip.code}
                tip={tip}
                index={items.length + i}
                onAdd={() =>
                  onAdd('lifestyle', { name: tip.title ?? '', weeklyGoal: tip.weeklyGoal, goalUnit: tip.goalUnit })
                }
              />
            ))}
          </Card>
        </div>
      ) : null}
    </section>
  );
}

function useGoalText() {
  const t = useTranslations('menopause.treatment.lifestyle');
  const locale = useLocale() as Locale;
  return (target: number, unit: 'sessions' | 'minutes') => t(unit, { count: formatNumber(target, locale) });
}

function LifestyleRow({
  item,
  index,
  screen,
  onEdit,
  onMinutes,
}: {
  item: TreatmentItem;
  index: number;
  screen: TreatmentScreen;
  onEdit: (item: TreatmentItem) => void;
  onMinutes: (item: TreatmentItem) => void;
}) {
  const t = useTranslations('menopause.treatment');
  const locale = useLocale() as Locale;
  const goalText = useGoalText();
  const intake = useTreatmentIntake();
  const look = itemLook('lifestyle', index);
  const goal = item.goal;
  const tip = lifestyleTipOf(item, screen.tips);
  const target = goal ? goalText(goal.target, goal.unit) : null;
  const sub = [target, tip?.body && tip.body !== target ? tip.body : null].filter(Boolean).join(' · ');
  const doneToday = todayAmount(item, screen.date) > 0 || item.takenToday;
  const log = () => {
    if (goal?.unit === 'minutes') onMinutes(item);
    else intake.mutate({ id: item.id, date: screen.date, taken: !doneToday });
  };
  return (
    <div className="mtr-row">
      <button type="button" className="mtr-row-main" onClick={() => onEdit(item)} aria-label={t('editA11y', { name: item.name })}>
        <IconCircle icon={look.icon} tone={look.tone} size="md" />
        <span className="mtr-row-text">
          <b className="mtr-row-title">{item.name}</b>
          {sub ? <span className="mtr-row-sub">{sub}</span> : null}
        </span>
      </button>
      {goal ? (
        <button
          type="button"
          className={clsx('mtr-goal', goal.done && 'is-done', doneToday && 'is-today')}
          aria-pressed={goal.unit === 'sessions' ? doneToday : undefined}
          aria-label={t('lifestyle.logA11y', {
            name: item.name,
            progress: t('lifestyle.progress', {
              amount: formatNumber(goal.amount, locale),
              target: formatNumber(goal.target, locale),
            }),
          })}
          disabled={intake.isPending}
          onClick={log}
        >
          {doneToday ? <Icon name="check" size={14} strokeWidth={2.6} /> : null}
          {t('lifestyle.progress', { amount: formatNumber(goal.amount, locale), target: formatNumber(goal.target, locale) })}
        </button>
      ) : null}
      {intake.isError ? (
        <p className="mtr-row-error" role="alert">
          {t('error')}
        </p>
      ) : null}
    </div>
  );
}

function SuggestionRow({ tip, index, onAdd }: { tip: TreatmentTip; index: number; onAdd: () => void }) {
  const t = useTranslations('menopause.treatment');
  const goalText = useGoalText();
  const look = itemLook('lifestyle', index);
  const target = tip.weeklyGoal && tip.goalUnit ? goalText(tip.weeklyGoal, tip.goalUnit) : null;
  const sub = [target, tip.body && tip.body !== target ? tip.body : null].filter(Boolean).join(' · ');
  return (
    <div className="mtr-row">
      <span className="mtr-row-main">
        <IconCircle icon={look.icon} tone={look.tone} size="md" />
        <span className="mtr-row-text">
          <b className="mtr-row-title">{tip.title}</b>
          {sub ? <span className="mtr-row-sub">{sub}</span> : null}
        </span>
      </span>
      <button type="button" className="mtr-take is-open" onClick={onAdd} aria-label={t('lifestyle.addA11y', { name: tip.title ?? '' })}>
        <Icon name="plus" size={14} strokeWidth={2.6} />
        {t('add')}
      </button>
    </div>
  );
}

function StoppedSection({ items, onEdit }: { items: TreatmentItem[]; onEdit: (item: TreatmentItem) => void }) {
  const t = useTranslations('menopause.treatment');
  const locale = useLocale() as Locale;
  return (
    <section className="mtr-sec" aria-labelledby="mtr-stopped">
      <SectionTitle id="mtr-stopped" title={t('stopped.title')} />
      <Card className="mtr-list" padding="none">
        {items.map((item) => (
          <div key={item.id} className="mtr-row is-stopped">
            <button type="button" className="mtr-row-main" onClick={() => onEdit(item)} aria-label={t('editA11y', { name: item.name })}>
              <IconCircle icon={item.kind === 'lifestyle' ? 'run' : 'pill'} tone="neutral" size="md" />
              <span className="mtr-row-text">
                <b className="mtr-row-title">{item.name}</b>
                <span className="mtr-row-sub">
                  {t(`kinds.${item.kind}`)}
                  {item.stoppedOn ? ` · ${t('stopped.on', { date: formatDayMonth(fromApiDate(item.stoppedOn), locale) })}` : null}
                </span>
              </span>
            </button>
          </div>
        ))}
      </Card>
    </section>
  );
}
