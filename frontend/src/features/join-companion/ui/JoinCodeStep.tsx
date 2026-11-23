'use client';

import { useTranslations } from 'next-intl';
import { useId } from 'react';

import { CompanionCodeField } from '@/entities/companion';
import { Icon } from '@/shared/ui';

import type { JoinCompanion } from '../model/use-join-companion';
import '@/shared/ui/styles/onb.css';
import '@/shared/ui/styles/cmh.css';

interface JoinCodeStepProps {
  join: JoinCompanion;
  autoFocus?: boolean;
  /** The onboarding screen titles itself through its frame. */
  hideIntro?: boolean;
}

/** compv2 `join`: link illustration, «کدت را وارد کن», the 6-character field and the 24-hour validity line. */
export function JoinCodeStep({ join, autoFocus, hideIntro }: JoinCodeStepProps) {
  const t = useTranslations('companionHome');
  const hintId = useId();
  const busy = join.checking || join.accepting;
  return (
    <div className="cmh2-join">
      {hideIntro ? null : (
        <>
          <span className="cmh2-illus" aria-hidden>
            <Icon name="users" size={34} />
          </span>
          <div className="cmh2-join-intro">
            <h2 className="cmh2-title">{t('join.title')}</h2>
            <p className="cmh2-sub">{t('join.sub')}</p>
          </div>
        </>
      )}
      <div className="cmh2-field">
        <span className="cmh2-field-label" aria-hidden>
          {t('join.fieldLabel')}
        </span>
        <CompanionCodeField
          value={join.code}
          onChange={join.changeCode}
          onSubmit={join.submitCode}
          label={t('code.label')}
          describedBy={hintId}
          invalid={join.error != null}
          disabled={busy}
          autoFocus={autoFocus}
          pilotId="companion.code"
        />
      </div>
      {join.error ? (
        <p className="onb2-error is-center" role="alert">
          {t(`code.errors.${join.error}`)}
        </p>
      ) : (
        <p id={hintId} className="cmh-code-hint">
          {t('code.hint')}
        </p>
      )}
      <p className="cmh2-note">
        <Icon name="clock" size={16} className="cmh2-note-ic" />
        <span>{t('join.validity')}</span>
      </p>
    </div>
  );
}
