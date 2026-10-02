'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useReducer } from 'react';

import {
  type EpdsKind,
  type EpdsQuestionnaire,
  type EpdsResult,
  useEpdsQuestions,
  usePostpartum,
  useSubmitEpds,
} from '@/entities/postpartum';
import { getApiSaveErrorMessage } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate } from '@/shared/lib/date';
import {
  Card,
  EmptyState,
  IconCircle,
  PrimaryButton,
  ProgressSteps,
  RadioCardGroup,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { SafetyScreen } from './SafetyScreen';
import {
  answersPayload,
  type Answers,
  flowReducer,
  initialFlow,
  pageComplete,
  pagesOf,
} from '../model/flow';

type T = ReturnType<typeof useTranslations<'postpartum'>>;

interface Props {
  /** `?kind=` of the link (an alert / follow-up action); null → the check that is due, else the weekly one. */
  kind: EpdsKind | null;
}

/**
 * `/postpartum/mood` — the EPDS check (nbl_v15_MoodCheck, B-N5-01): questions
 * from the API (weekly = 3 items, full = 10), non-diagnostic copy, a result
 * screen, and the urgent safety screen (call buttons first). The answers live
 * in this component's state and the one request only. A form: no bottom nav.
 */
export function PostpartumMoodPage({ kind: kindParam }: Props) {
  const t = useTranslations('postpartum');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const overview = usePostpartum();
  const kind: EpdsKind | null = kindParam ?? (overview.isPending ? null : (overview.data?.checkin?.due ?? 'short'));
  const questions = useEpdsQuestions(kind ?? 'short', locale);
  const weeks = overview.data?.status?.weeks;
  const count = questions.data?.items.length;

  const subtitle =
    count == null
      ? undefined
      : weeks != null
        ? t('mood.subtitle', { week: formatNumber(Math.max(1, weeks), locale), count: formatNumber(count, locale) })
        : t('mood.subtitleNoWeek', { count: formatNumber(count, locale) });

  let body;
  if (!kind || questions.isPending) {
    body = (
      <SkeletonGroup label={t('common.loading')} className="pp-form">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (questions.isError) {
    body = (
      <div className="pp-form">
        <Card>
          <EmptyState
            icon="warning"
            title={t('common.loadError')}
            action={
              <PrimaryButton icon="refresh" loading={questions.isFetching} onClick={() => void questions.refetch()}>
                {t('common.retry')}
              </PrimaryButton>
            }
          />
        </Card>
      </div>
    );
  } else {
    body = <MoodFlow key={kind} questionnaire={questions.data} t={t} />;
  }

  return (
    <div className="view pp-screen pp-form-page">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('mood.title')}
          subtitle={subtitle}
          onBack={() => router.push('/postpartum')}
          backLabel={t('common.back')}
        />
        {body}
      </div>
    </div>
  );
}

function MoodFlow({ questionnaire, t }: { questionnaire: EpdsQuestionnaire; t: T }) {
  const locale = useLocale() as Locale;
  const router = useRouter();
  const [state, dispatch] = useReducer(flowReducer, initialFlow);
  const { state: submitState, submit } = useSubmitEpds();
  const pages = pagesOf(questionnaire.items);
  const items = questionnaire.items;

  const send = async (answers: Answers) => {
    const payload = answersPayload(items, answers);
    if (!payload) return;
    const result = await submit(questionnaire.kind, payload);
    if (result) dispatch({ type: 'success', result });
    else dispatch({ type: 'failure', kind: questionnaire.kind, items });
  };

  if (state.step === 'safety') {
    return (
      <SafetyScreen
        safety={state.safety}
        saved={state.saved}
        retrying={submitState.status === 'pending'}
        onRetry={state.answers ? () => void send(state.answers as Answers) : undefined}
        t={t}
      />
    );
  }
  if (state.step === 'result') return <ResultScreen result={state.result} t={t} />;

  const page = pages[state.page] ?? [];
  const last = state.page >= pages.length - 1;
  const complete = pageComplete(page, state.answers);
  const submitting = state.step === 'submitting';
  const failed = submitState.status === 'error' && state.step === 'questions';

  const onContinue = () => {
    if (!complete || submitting) return;
    if (!last) {
      dispatch({ type: 'next', pageCount: pages.length });
      return;
    }
    dispatch({ type: 'submit' });
    void send(state.answers);
  };

  return (
    <div className="pp-form">
      {questionnaire.disclaimer && (
        <div className="pp-disclaimer" role="note">
          <IconCircle icon="heartLine" tone="bloom" size="lg" />
          <p>{questionnaire.disclaimer}</p>
        </div>
      )}
      {page.map((item, i) => (
        <Card as="section" key={item.code} className="pp-q" aria-labelledby={`pp-q-${item.code}`}>
          <div className="pp-q-head">
            <span className="pp-q-num" aria-hidden>
              {formatNumber(state.page * pages[0].length + i + 1, locale)}
            </span>
            <h2 id={`pp-q-${item.code}`} className="pp-q-text">
              {state.page === 0 && i === 0 && questionnaire.intro ? `${questionnaire.intro} ${item.text}` : item.text}
            </h2>
          </div>
          <RadioCardGroup
            className="pp-q-options"
            label={item.text}
            value={state.answers[item.code] !== undefined ? String(state.answers[item.code]) : null}
            onChange={(v) => dispatch({ type: 'answer', code: item.code, score: Number(v) })}
            options={item.options.map((o) => ({ value: String(o.score), title: o.label }))}
          />
        </Card>
      ))}
      {pages.length > 1 && (
        <ProgressSteps
          total={pages.length}
          current={state.page + 1}
          label={t('mood.page', { page: formatNumber(state.page + 1, locale), pages: formatNumber(pages.length, locale) })}
          className="pp-q-steps"
        />
      )}
      {failed && (
        <p className="pp-error" role="alert">
          {getApiSaveErrorMessage(submitState.error, t('mood.saveError'))}
        </p>
      )}
      <PrimaryButton disabled={!complete} loading={submitting} onClick={onContinue}>
        {last ? t('mood.submit') : t('mood.continue')}
      </PrimaryButton>
      {state.page > 0 && !submitting ? (
        <SecondaryButton variant="text" onClick={() => dispatch({ type: 'prev' })}>
          {t('mood.previous')}
        </SecondaryButton>
      ) : (
        <SecondaryButton variant="text" onClick={() => router.push('/postpartum')}>
          {t('mood.notNow')}
        </SecondaryButton>
      )}
    </div>
  );
}

function ResultScreen({ result, t }: { result: EpdsResult; t: T }) {
  const locale = useLocale() as Locale;
  const { check, safety } = result;
  const nextDue = result.checkin?.nextDueOn;
  const openCheck = safety?.actions.find((a) => a.type === 'open_check');
  return (
    <div className="pp-form">
      <Card as="section" className="pp-result" aria-labelledby="pp-result-title">
        <h2 id="pp-result-title" className="pp-field-title">
          {t('mood.result.title')}
        </h2>
        {check.bandLabel && <p className="pp-result-band">{check.bandLabel}</p>}
        <p className="pp-result-score">
          {t('mood.result.score', { total: formatNumber(check.total, locale), max: formatNumber(check.max, locale) })}
        </p>
        <p className="pp-result-note">{t('mood.result.note')}</p>
        {nextDue && (
          <p className="pp-result-next">{t('mood.result.nextDue', { date: formatDayMonth(fromApiDate(nextDue), locale) })}</p>
        )}
      </Card>
      {safety && (safety.title || safety.body) && (
        <Card as="section" className="pp-alert is-stacked">
          <div className="pp-alert-text">
            {safety.title && <p className="pp-alert-title">{safety.title}</p>}
            {safety.body && <p className="pp-alert-body">{safety.body}</p>}
          </div>
          {openCheck?.type === 'open_check' && (
            <Link href={`/postpartum/mood?kind=${openCheck.kind}`} className="nb-btn is-primary is-block">
              {openCheck.label ?? t('home.moodCheck.start')}
            </Link>
          )}
        </Card>
      )}
      <p className="pp-result-talk">{t('mood.result.talk')}</p>
      <Link href="/postpartum" className="nb-btn is-outline is-block">
        {t('mood.result.home')}
      </Link>
    </div>
  );
}
