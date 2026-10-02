'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  type MenopauseScoreQuestion,
  useMenopauseScoreQuestions,
  useMenopauseScores,
  useSaveMenopauseScore,
} from '@/entities/menopause';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatMonthLabel, formatNumber, todayParts } from '@/shared/lib/date';
import {
  Card,
  EmptyState,
  InfoNote,
  NumericScale,
  PrimaryButton,
  ProgressSteps,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { answersFor, groupQuestions, initialAnswers, isComplete, isKnownDomain } from '../model/score';

/**
 * Monthly questionnaire (`/menopause/score/questionnaire`, CB-MENO-08): the
 * 11 `meno_score_items` questions rated 0–4, grouped by domain, prefilled with
 * this month's answers when she already filled it (a refill replaces it).
 * → `POST /menopause/scores`, then back to the score. A form: no bottom nav.
 */
export function MenopauseQuestionnairePage() {
  const t = useTranslations('menopause.score.questionnaire');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const questions = useMenopauseScoreQuestions();
  const history = useMenopauseScores(6);
  const now = todayParts(locale);

  let body;
  if (questions.isPending || history.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="msq-skel">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} shape="card" />
        ))}
      </SkeletonGroup>
    );
  } else if (questions.isError || !questions.data.length) {
    body = (
      <EmptyState
        icon="note"
        title={t('loadError')}
        action={
          <PrimaryButton icon="refresh" loading={questions.isFetching} onClick={() => void questions.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <QuestionnaireForm questions={questions.data} initial={initialAnswers(history.data)} />;
  }

  return (
    <div className="view msq-page">
      <SkyLayer />
      <div className="scroll msq-scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={formatMonthLabel(now.year, now.month, locale)}
          onBack={() => router.push('/menopause/score')}
          backLabel={t('back')}
        />
        {body}
      </div>
    </div>
  );
}

function QuestionnaireForm({
  questions,
  initial,
}: {
  questions: MenopauseScoreQuestion[];
  initial: Record<string, number>;
}) {
  const t = useTranslations('menopause.score.questionnaire');
  const tDomain = useTranslations('menopause.score.domains');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const save = useSaveMenopauseScore();
  const [answers, setAnswers] = useState<Record<string, number>>(() => answersFor(questions, initial));

  const done = questions.filter((q) => answers[q.code] !== undefined).length;
  const complete = isComplete(questions, answers);
  const progress = t('progress', { done: formatNumber(done, locale), total: formatNumber(questions.length, locale) });

  const submit = () => {
    if (!complete || save.isPending) return;
    save.mutate({ answers: answersFor(questions, answers) }, { onSuccess: () => router.replace('/menopause/score') });
  };

  return (
    <>
      <div className="msq-intro">
        <p className="msq-lead">{t('lead')}</p>
        <ProgressSteps total={questions.length} current={done} label={progress} className="msq-steps" />
      </div>

      {groupQuestions(questions).map((group) => (
        <section key={group.domain} className="msq-group" aria-labelledby={`msq-${group.domain}`}>
          <h2 id={`msq-${group.domain}`} className="msq-group-title">
            {isKnownDomain(group.domain) ? tDomain(group.domain) : group.domain}
          </h2>
          <Card className="msq-card">
            {group.questions.map((q) => (
              <div key={q.code} className="msq-q">
                <NumericScale
                  min={0}
                  max={q.max}
                  value={answers[q.code] ?? null}
                  onChange={(v) => setAnswers((a) => ({ ...a, [q.code]: v }))}
                  locale={locale}
                  label={
                    <span className="msq-q-label">
                      <b className="msq-q-title">{q.title}</b>
                      {q.body ? <span className="msq-q-body">{q.body}</span> : null}
                    </span>
                  }
                  minLabel={t('scaleMin')}
                  maxLabel={t('scaleMax')}
                />
              </div>
            ))}
          </Card>
        </section>
      ))}

      <InfoNote className="msq-note">{t('note')}</InfoNote>

      <div className="msq-footer">
        {save.isError ? (
          <p className="msq-error" role="alert">
            {getApiSaveErrorMessage(save.error, t('saveError'))}
          </p>
        ) : null}
        <p className="msq-progress" aria-live="polite">
          {progress}
        </p>
        <PrimaryButton loading={save.isPending} disabled={!complete} onClick={submit}>
          {t('submit')}
        </PrimaryButton>
      </div>
    </>
  );
}
