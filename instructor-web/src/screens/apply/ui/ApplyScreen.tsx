'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import type { Instructor, InstructorAccess } from '@/entities/instructor';
import { ApplyForm } from '@/features/apply-instructor';
import { toPersianDigits } from '@/shared/lib';
import { Icon } from '@/shared/ui';
import { StateFrame } from '@/widgets/state-frame';

const STEPS = ['step1', 'step2', 'step3'] as const;

/** /apply — «instructor_required»: never applied, or access revoked (may re-apply). */
export function ApplyScreen({ instructor, access }: { instructor: Instructor | null; access: InstructorAccess }) {
  const t = useTranslations('apply');
  const router = useRouter();
  const revoked = access === 'revoked';
  return (
    <StateFrame>
      <section className="state-hero">
        <span className={revoked ? 'state-hero__icon state-hero__icon--warm' : 'state-hero__icon'}>
          <Icon name={revoked ? 'lock' : 'sparkle'} size={26} />
        </span>
        <h1 className="state-hero__title">{revoked ? t('revokedTitle') : t('title')}</h1>
        <p className="state-hero__lead">{revoked ? t('revokedLead') : t('lead')}</p>
      </section>
      {revoked ? null : (
        <ol className="steps">
          {STEPS.map((key, i) => (
            <li key={key} className="steps__item">
              <span className="steps__num">{toPersianDigits(i + 1)}</span>
              <span>{t(`steps.${key}`)}</span>
            </li>
          ))}
        </ol>
      )}
      <section className="card">
        <h2 className="card__title">{t('formTitle')}</h2>
        <ApplyForm
          initial={instructor}
          submitLabel={revoked ? t('reapply') : t('submit')}
          onDone={() => router.replace('/pending')}
        />
      </section>
    </StateFrame>
  );
}
