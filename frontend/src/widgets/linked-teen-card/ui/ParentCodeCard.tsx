'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import {
  acceptErrorKind,
  CompanionCodeField,
  isCompleteCompanionCode,
  useAcceptCompanion,
} from '@/entities/companion';
import { teenKeys } from '@/entities/teen';
import { PrimaryButton } from '@/shared/ui';

/**
 * «کد همراهی از فرزندت داری؟» — where a mother (any non-companion account)
 * types the 6-character code her teen sent from `/teen/parent`
 * (`POST /companions/accept`, bloom's accept). The invite is bound to her
 * number, so only she can use it. After linking, her teen's read-only card
 * appears on her Today ({@link LinkedTeenCards}).
 */
export function ParentCodeCard() {
  const t = useTranslations('teen.parent.code');
  const queryClient = useQueryClient();
  const hintId = useId();
  const titleId = useId();
  const accept = useAcceptCompanion();
  const [code, setCode] = useState('');
  const [linked, setLinked] = useState<string | null | undefined>(undefined);
  const complete = isCompleteCompanionCode(code);

  const connect = () => {
    if (!complete || accept.isPending) return;
    accept.mutate(code, {
      onSuccess: (link) => {
        setCode('');
        setLinked(link.owner.name);
        void queryClient.invalidateQueries({ queryKey: teenKeys.linked() });
      },
    });
  };

  return (
    <section className="nb-card ltc-code" aria-labelledby={titleId}>
      <div className="ltc-code-text">
        <h3 id={titleId} className="ltc-code-title">
          {t('title')}
        </h3>
        <p id={hintId} className="ltc-code-lead">
          {t('lead')}
        </p>
      </div>
      <CompanionCodeField
        value={code}
        onChange={(next) => {
          setCode(next);
          setLinked(undefined);
          if (accept.isError) accept.reset();
        }}
        onSubmit={connect}
        label={t('label')}
        describedBy={hintId}
        invalid={accept.isError}
        disabled={accept.isPending}
      />
      {accept.isError ? (
        <p className="ltc-code-msg is-error" role="alert">
          {t(`errors.${acceptErrorKind(accept.error)}`)}
        </p>
      ) : linked !== undefined ? (
        <p className="ltc-code-msg is-success" role="status">
          {linked ? t('success', { name: linked }) : t('successNoName')}
        </p>
      ) : null}
      <PrimaryButton loading={accept.isPending} disabled={!complete} onClick={connect}>
        {t('connect')}
      </PrimaryButton>
    </section>
  );
}
