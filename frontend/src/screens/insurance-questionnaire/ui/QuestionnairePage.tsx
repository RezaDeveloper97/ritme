'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRef, useState } from 'react';

import {
  evidenceLabel,
  questionCode,
  sectionCode,
  type InsuranceQuestionnaire,
  type QuestionAnswer,
  type QuestionnairePurpose,
  type QuestionnaireQuestion,
  useConfirmQuestionnaire,
  useQuestionnaire,
  useQuestionnaires,
  useStartQuestionnaire,
  useUpdateQuestionnaire,
} from '@/entities/insurance';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import {
  ChipGroup,
  EmptyState,
  Icon,
  InfoNote,
  PillChip,
  PrimaryButton,
  ProgressSteps,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';

import { useQuestionnairePdf } from '../lib/use-questionnaire-pdf';
import '@/shared/ui/styles/ins.css';
import './insq.css';

const MAX_EXPLANATION = 2000;

function Loading() {
  const t = useTranslations('insurance.questionnaire');
  return (
    <SkeletonGroup label={t('loading')} className="ins-skel">
      <Skeleton shape="card" />
      <Skeleton shape="card" />
    </SkeletonGroup>
  );
}

/** First visit (or after the last one was sent): choose what it is for and start a draft prefilled from the record. */
function StartCard() {
  const t = useTranslations('insurance.questionnaire');
  const start = useStartQuestionnaire();
  const [purpose, setPurpose] = useState<QuestionnairePurpose>('renewal');
  return (
    <section className="nb-card ins-card" aria-labelledby="insq-start-h">
      <h2 id="insq-start-h" className="ins-h">
        {t('start.title')}
      </h2>
      <p className="ins-body">{t('start.body')}</p>
      <ChipGroup label={t('start.purposeLabel')} pilot={pilot('ins-quest.purpose', 'choice')}>
        {(['issue', 'renewal'] as const).map((p) => (
          <PillChip key={p} pressed={purpose === p} onPressedChange={() => setPurpose(p)} {...pilotOption(p)}>
            {t(`start.purpose.${p}`)}
          </PillChip>
        ))}
      </ChipGroup>
      {start.isError ? (
        <p className="ins-problem" role="alert">
          {t('saveError')}
        </p>
      ) : null}
      <PrimaryButton icon="fileDoc" loading={start.isPending} onClick={() => start.mutate({ purpose })} {...pilot('ins-quest.begin')}>
        {start.isPending ? t('start.starting') : t('start.begin')}
      </PrimaryButton>
    </section>
  );
}

function AnswerRow({
  question,
  onAnswer,
}: {
  question: QuestionnaireQuestion;
  onAnswer: (value: QuestionAnswer) => void;
}) {
  const t = useTranslations('insurance.questionnaire');
  const qc = questionCode(question.code);
  const label = qc ? t(`q.${qc}`) : question.code;
  return (
    <div className="nb-card insq-row">
      <p className="insq-q" id={`insq-${question.code}`}>
        {label}
      </p>
      <div className="insq-answer">
        <ChipGroup label={label} layout="fill" pilot={pilot(`ins-quest.answer:${question.code}`, 'choice')}>
          {(['yes', 'no'] as const).map((v) => (
            <PillChip key={v} pressed={question.answer === v} onPressedChange={() => onAnswer(v)} {...pilotOption(v)}>
              {t(v)}
            </PillChip>
          ))}
        </ChipGroup>
        {question.fromRecord ? (
          <StatusPill tone="data" icon="fileDoc">
            {t('fromRecord')}
          </StatusPill>
        ) : null}
      </div>
    </div>
  );
}

/** The record items behind the «yes» answers of a section, one line each, with the link back to the record. */
function EvidenceList({ questions }: { questions: QuestionnaireQuestion[] }) {
  const t = useTranslations('insurance.questionnaire');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const lines = questions
    .filter((q) => q.answer === 'yes' && q.evidence.length > 0)
    .map((q) => {
      const items = q.evidence.map((raw) => {
        const e = evidenceLabel(raw);
        const text = e.code ? t(`evidence.${e.code}`) : e.text;
        return e.date ? `${text} ${formatLongDate(fromApiDate(e.date), locale)}` : text;
      });
      const qc = questionCode(q.code);
      return { code: q.code, label: qc ? t(`q.${qc}`) : q.code, text: items.join(' · ') };
    });
  if (lines.length === 0) return null;
  return (
    <>
      <ul className="insq-evidence" aria-label={t('evidence.title')}>
        {lines.map((l) => (
          <li key={l.code}>
            <span className="insq-evidence-k">{l.label}</span> {l.text}
          </li>
        ))}
      </ul>
      <SecondaryButton variant="text" icon="fileDoc" block={false} onClick={() => router.push('/record')}>
        {t('explanation.openRecord')}
      </SecondaryButton>
    </>
  );
}

function ExplanationCard({
  value,
  onChange,
  questions,
}: {
  value: string;
  onChange: (v: string) => void;
  questions: QuestionnaireQuestion[];
}) {
  const t = useTranslations('insurance.questionnaire.explanation');
  return (
    <section className="nb-card ins-card" aria-labelledby="insq-expl-h">
      <h2 id="insq-expl-h" className="ins-h">
        {t('title')}
      </h2>
      <EvidenceList questions={questions} />
      <textarea
        className="insq-text"
        aria-labelledby="insq-expl-h"
        rows={3}
        maxLength={MAX_EXPLANATION}
        placeholder={t('placeholder')}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        {...pilot('ins-quest.explanation', 'text')}
      />
      <p className="ins-hint">{t('hint')}</p>
    </section>
  );
}

function ReviewList({ q, answers, explanation, onEdit }: { q: InsuranceQuestionnaire; answers: Record<string, QuestionAnswer | null>; explanation: string; onEdit: (section: string) => void }) {
  const t = useTranslations('insurance.questionnaire');
  return (
    <>
      {q.sections
        .filter((s) => s !== 'review')
        .map((section) => (
          <section key={section} className="nb-card ins-card" aria-labelledby={`insq-rv-${section}`}>
            <div className="ins-sec-head">
              <h2 id={`insq-rv-${section}`} className="ins-h">
                {t(`sections.${sectionCode(section)}`)}
              </h2>
              <button type="button" className="ins-more insq-edit" onClick={() => onEdit(section)} {...pilot(`ins-quest.edit-section:${section}`)}>
                {t('review.edit')}
              </button>
            </div>
            <ul className="insq-sum">
              {q.questions
                .filter((x) => x.section === section)
                .map((x) => (
                  <li key={x.code} className="insq-sum-row">
                    <span>{questionCode(x.code) ? t(`q.${questionCode(x.code)!}`) : x.code}</span>
                    <span className="insq-sum-a">{answers[x.code] ? t(answers[x.code] as QuestionAnswer) : t('notAnswered')}</span>
                  </li>
                ))}
            </ul>
          </section>
        ))}
      {explanation.trim() ? (
        <section className="nb-card ins-card">
          <h2 className="ins-h">{t('explanation.title')}</h2>
          <p className="ins-body">{explanation.trim()}</p>
        </section>
      ) : null}
    </>
  );
}

/** Stepper over the questionnaire sections; the last step reviews and confirms. */
function Editor({ q, onLeave, onConfirmed }: { q: InsuranceQuestionnaire; onLeave: () => void; onConfirmed: () => void }) {
  const t = useTranslations('insurance.questionnaire');
  const tc = useTranslations('insurance');
  const locale = useLocale() as Locale;
  const update = useUpdateQuestionnaire(q.id);
  const confirm = useConfirmQuestionnaire(q.id);
  const [answers, setAnswers] = useState<Record<string, QuestionAnswer | null>>(() =>
    Object.fromEntries(q.questions.map((x) => [x.code, x.answer])),
  );
  const [explanation, setExplanation] = useState(q.explanation);
  const [step, setStep] = useState(0);
  const dirty = useRef(false);
  const [failed, setFailed] = useState(false);

  const sections = q.sections.length > 0 ? q.sections : ['review'];
  const section = sections[Math.min(step, sections.length - 1)]!;
  const isReview = section === 'review';
  const questions = q.questions.map((x) => ({ ...x, answer: answers[x.code] ?? null }));
  const inSection = questions.filter((x) => x.section === section);
  const unanswered = questions.filter((x) => x.answer === null).length;

  const persist = async (): Promise<boolean> => {
    if (!dirty.current) return true;
    try {
      await update.mutateAsync({ answers, explanation: explanation.trim() === '' ? null : explanation });
      dirty.current = false;
      setFailed(false);
      return true;
    } catch {
      setFailed(true);
      return false;
    }
  };
  const go = async (to: number) => {
    if (await persist()) setStep(to);
  };
  const doConfirm = async () => {
    if (!(await persist())) return;
    try {
      await confirm.mutateAsync();
      onConfirmed();
    } catch {
      setFailed(true);
    }
  };

  const answer = (code: string, value: QuestionAnswer) => {
    dirty.current = true;
    setAnswers((a) => ({ ...a, [code]: value }));
  };
  const showExplanation = !isReview && inSection.some((x) => x.answer === 'yes');

  return (
    <>
      <ScreenHeader
        title={t('title')}
        subtitle={t('subtitle')}
        onBack={() => void persist().then((ok) => ok && onLeave())}
        backLabel={tc('back')}
      />
      <div className="insq-stepper">
        <ProgressSteps
          total={sections.length}
          current={step + 1}
          label={t('stepLabel', { current: formatNumber(step + 1, locale), total: formatNumber(sections.length, locale) })}
        />
        <p className="insq-stepof">
          {t('stepOf', {
            current: formatNumber(step + 1, locale),
            total: formatNumber(sections.length, locale),
            name: t(`sections.${sectionCode(section)}`),
          })}
        </p>
      </div>

      {step === 0 ? (
        <div className="insq-banner">
          <Icon name="fileDoc" size={18} className="ins-doc-icon" />
          <div>
            <p className="insq-banner-t">{t('prefilled', { count: q.prefilledCount })}</p>
            <p className="ins-hint">{t('checkFirst')}</p>
          </div>
        </div>
      ) : null}

      {isReview ? (
        <ReviewList
          q={q}
          answers={answers}
          explanation={explanation}
          onEdit={(s) => void go(Math.max(0, sections.indexOf(s)))}
        />
      ) : (
        <>
          {inSection.map((x) => (
            <AnswerRow key={x.code} question={x} onAnswer={(v) => answer(x.code, v)} />
          ))}
          {showExplanation ? (
            <ExplanationCard
              value={explanation}
              onChange={(v) => {
                dirty.current = true;
                setExplanation(v);
              }}
              questions={inSection}
            />
          ) : null}
        </>
      )}

      <InfoNote icon="lock">{t('warning')}</InfoNote>

      {failed || confirm.isError ? (
        <p className="ins-problem" role="alert">
          {confirm.isError && unanswered > 0 ? t('incomplete', { count: unanswered }) : t('saveError')}
        </p>
      ) : null}

      <div className="insq-nav">
        {isReview ? (
          <PrimaryButton icon="check" loading={confirm.isPending || update.isPending} disabled={unanswered > 0} onClick={() => void doConfirm()} {...pilot('ins-quest.confirm')}>
            {confirm.isPending ? t('confirming') : t('confirm')}
          </PrimaryButton>
        ) : (
          <PrimaryButton loading={update.isPending} onClick={() => void go(step + 1)} {...pilot('ins-quest.next')}>
            {t('next')}
          </PrimaryButton>
        )}
        {isReview && unanswered > 0 ? <p className="ins-hint">{t('incomplete', { count: unanswered })}</p> : null}
        {step > 0 ? (
          <SecondaryButton variant="text" onClick={() => void go(step - 1)} {...pilot('ins-quest.prev')}>
            {t('prev')}
          </SecondaryButton>
        ) : null}
      </div>
    </>
  );
}

/** Confirmed: the answers are kept in her record; export a PDF, change them, or go record what the insurer said. */
function Confirmed({ q, onEdit }: { q: InsuranceQuestionnaire; onEdit: () => void }) {
  const t = useTranslations('insurance.questionnaire');
  const router = useRouter();
  const pdf = useQuestionnairePdf(q);
  return (
    <section className="nb-card ins-card insq-done" aria-labelledby="insq-done-h">
      <span className="ins-policy-ico" aria-hidden>
        <Icon name="check" size={22} />
      </span>
      <h2 id="insq-done-h" className="ins-h">
        {t('done.title')}
      </h2>
      <p className="ins-body">{t('done.body')}</p>
      <PrimaryButton icon="download" loading={pdf.state === 'working'} onClick={() => void pdf.run()}>
        {pdf.state === 'working' ? t('pdf.working') : t('pdf.export')}
      </PrimaryButton>
      {pdf.state === 'error' ? (
        <p className="ins-problem" role="alert">
          {t('pdf.error')}
        </p>
      ) : null}
      <SecondaryButton icon="history" onClick={() => router.push('/insurance/status')} {...pilot('ins-quest.status')}>
        {t('done.status')}
      </SecondaryButton>
      <SecondaryButton variant="text" icon="pencil" onClick={onEdit} {...pilot('ins-quest.edit')}>
        {t('done.edit')}
      </SecondaryButton>
    </section>
  );
}

function Active({ id }: { id: number }) {
  const t = useTranslations('insurance');
  const router = useRouter();
  const query = useQuestionnaire(id);
  const [editing, setEditing] = useState(false);
  const q = query.data;
  if (query.isPending) return <Loading />;
  if (query.isError || !q) {
    return (
      <EmptyState
        icon="shield"
        title={t('questionnaire.error')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  }
  if (q.status === 'confirmed' && !editing) {
    return (
      <>
        <ScreenHeader title={t('questionnaire.title')} subtitle={t('questionnaire.subtitle')} onBack={() => router.back('/insurance')} backLabel={t('back')} />
        <Confirmed q={q} onEdit={() => setEditing(true)} />
      </>
    );
  }
  return <Editor key={`${q.id}-${editing}`} q={q} onLeave={() => router.push('/insurance')} onConfirmed={() => setEditing(false)} />;
}

/**
 * Health questionnaire (CB-INS-05, nbl_Ins_Health) at `/insurance/questionnaire`: a section stepper of yes/no rows
 * prefilled from her own record (tagged «از پرونده سلامت»), an explanation for the «yes» answers, then review and
 * confirm. Nothing is sent to an insurer — the answers stay in her record and she can export a PDF on the device.
 * A form flow, so no bottom nav (IA_Nav).
 */
export function InsuranceQuestionnairePage() {
  const t = useTranslations('insurance');
  const router = useRouter();
  const list = useQuestionnaires();
  const active = list.data?.find((x) => x.status === 'draft' || x.status === 'confirmed') ?? null;

  let body;
  let header = true;
  if (list.isPending) {
    body = <Loading />;
  } else if (list.isError) {
    body = (
      <EmptyState
        icon="shield"
        title={t('questionnaire.error')}
        action={
          <PrimaryButton icon="refresh" loading={list.isFetching} onClick={() => void list.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (active) {
    header = false;
    body = <Active id={active.id} />;
  } else {
    body = <StartCard />;
  }

  return (
    <div className="view ins-page">
      <SkyLayer />
      <div className="scroll ins-scroll">
        {header ? (
          <ScreenHeader
            title={t('questionnaire.title')}
            subtitle={t('questionnaire.subtitle')}
            onBack={() => router.back('/insurance')}
            backLabel={t('back')}
          />
        ) : null}
        {body}
      </div>
    </div>
  );
}
