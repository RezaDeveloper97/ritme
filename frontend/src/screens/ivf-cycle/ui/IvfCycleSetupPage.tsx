'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { ivfValidationMessage, useIvfHome, useIvfProtocols, useStartIvfCycle } from '@/entities/ivf';
import { lifeStageKeys } from '@/entities/user';
import { getApiSaveErrorMessage } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { Card, EmptyState, InfoNote, PrimaryButton, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import {
  emptySetup,
  SETUP_STAGES,
  type SetupDraft,
  setSetupStage,
  setupProblems,
  setupToInput,
} from '../model/cycle';
import { ChoiceRow, DateSheet, FieldButton, FieldError } from './Fields';
import { protocolCodes, useProtocolLabel } from './protocols';

function Shell({ children }: { children: React.ReactNode }) {
  const t = useTranslations('ivf.cycle');
  const router = useRouter();
  return (
    <div className="view ivfm-form-page ivfc-page">
      <SkyLayer />
      <div className="scroll is-form">
        <ScreenHeader title={t('setup.title')} onBack={() => router.push('/ivf')} backLabel={t('back')} />
        {children}
      </div>
    </div>
  );
}

/**
 * «شروع سیکل درمان» — `/ivf/cycle/new` (CB-IVF-06b): protocol (catalog
 * `ivf_protocols`, optional), where she is now (preparation / stimulation),
 * cycle start and stimulation start → `POST /ivf/cycles` → the IVF home. A
 * form: no bottom nav. With a cycle already open it points at the editor.
 */
export function IvfCycleSetupPage() {
  const t = useTranslations('ivf.cycle');
  const mounted = useMounted();
  const home = useIvfHome();

  if (!mounted || home.isPending) {
    return (
      <Shell>
        <SkeletonGroup label={t('loading')} className="ivfm-form">
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (home.data?.cycle) {
    return (
      <Shell>
        <div className="ivfm-form">
          <Card>
            <EmptyState
              icon="drop"
              title={t('setup.open')}
              action={
                <Link href="/ivf/cycle" className="nb-btn is-outline is-block">
                  {t('setup.openCta')}
                </Link>
              }
            />
          </Card>
        </div>
      </Shell>
    );
  }

  return (
    <Shell>
      <SetupForm />
    </Shell>
  );
}

function SetupForm() {
  const t = useTranslations('ivf.cycle');
  const ti = useTranslations('ivf');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const queryClient = useQueryClient();
  const protocols = useIvfProtocols(locale);
  const protocolLabel = useProtocolLabel(protocols.data);
  const start = useStartIvfCycle();
  const todayIso = toApiDate(today());
  const [draft, setDraft] = useState<SetupDraft>(() => emptySetup(todayIso));
  const [picker, setPicker] = useState<'startedOn' | 'stimStartedOn' | null>(null);

  const problems = setupProblems(draft, todayIso);
  const blocked = Object.keys(problems).length > 0;
  const problemText = (field: 'startedOn' | 'stimStartedOn') =>
    problems[field] ? t(`editor.errors.${problems[field]}`) : null;
  const dateValue = (value: string | null) => (value ? formatDayMonth(fromApiDate(value), locale) : t('choose'));
  const selected = protocols.data?.find((p) => p.code === draft.protocol);

  const submit = () => {
    if (blocked) return;
    start.mutate(setupToInput(draft), {
      onSuccess: () => {
        // Starting a cycle switches «IVF/IUI» on server-side — the nav/home read it.
        void queryClient.invalidateQueries({ queryKey: lifeStageKeys.all });
        router.replace('/ivf');
      },
    });
  };

  return (
    <div className="ivfm-form">
      <p className="ivfm-block-hint ivfc-intro">{t('setup.intro')}</p>

      <Card className="ivfm-card">
        <ChoiceRow
          label={t('setup.protocol')}
          hint={t('setup.protocolHint')}
          options={protocolCodes(protocols.data)}
          value={draft.protocol}
          labelOf={protocolLabel}
          allowNone
          onChange={(protocol) => setDraft((d) => ({ ...d, protocol }))}
        />
        {selected?.body ? <p className="ivfm-block-hint">{selected.body}</p> : null}
      </Card>

      <Card className="ivfm-card">
        <ChoiceRow
          label={t('setup.where')}
          options={SETUP_STAGES}
          value={draft.stage}
          labelOf={(stage) => ti(`stages.${stage}.title`)}
          onChange={(stage) => stage && setDraft((d) => setSetupStage(d, stage))}
        />
        <div className="ivfm-two">
          <FieldButton
            label={t('setup.startedOn')}
            value={dateValue(draft.startedOn)}
            invalid={!!problemText('startedOn')}
            onClick={() => setPicker('startedOn')}
          />
          {draft.stage === 'stim' ? (
            <FieldButton
              label={t('setup.stimStartedOn')}
              value={dateValue(draft.stimStartedOn)}
              invalid={!!problemText('stimStartedOn')}
              onClick={() => setPicker('stimStartedOn')}
            />
          ) : null}
        </div>
        <FieldError text={problemText('startedOn') ?? problemText('stimStartedOn')} />
      </Card>

      <InfoNote>{t('setup.later')}</InfoNote>

      <div className="ivfm-footer">
        {start.isError ? (
          <p className="ivfm-error" role="alert">
            {ivfValidationMessage(start.error) ?? getApiSaveErrorMessage(start.error, t('setup.error'))}
          </p>
        ) : null}
        <PrimaryButton loading={start.isPending} disabled={blocked} onClick={submit}>
          {t('setup.submit')}
        </PrimaryButton>
      </div>

      {picker ? (
        <DateSheet
          title={t(`setup.${picker}`)}
          value={draft[picker]}
          onClose={() => setPicker(null)}
          onPick={(value) => {
            const field = picker;
            setDraft((d) => ({ ...d, [field]: value }));
            setPicker(null);
          }}
        />
      ) : null}
    </div>
  );
}
