'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { instructorKeys, type Instructor } from '@/entities/instructor';
import { ApplyForm } from '@/features/apply-instructor';
import { Button, Icon } from '@/shared/ui';
import { StateFrame } from '@/widgets/state-frame';

const dateFormat = new Intl.DateTimeFormat('fa-IR-u-ca-persian', {
  year: 'numeric',
  month: 'long',
  day: 'numeric',
  timeZone: 'Asia/Tehran',
});

/** /pending — «instructor_pending»: the application waits for an admin (B-N8-08). */
export function PendingScreen({ instructor }: { instructor: Instructor }) {
  const t = useTranslations('pending');
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [checking, setChecking] = useState(false);

  const recheck = async () => {
    setChecking(true);
    // The gate moves on by itself once /me says approved.
    await queryClient.refetchQueries({ queryKey: instructorKeys.me() });
    setChecking(false);
  };

  const submitted = instructor.created_at ? dateFormat.format(new Date(instructor.created_at)) : null;

  return (
    <StateFrame>
      <section className="state-hero">
        <span className="state-hero__icon state-hero__icon--data">
          <Icon name="hourglass" size={26} />
        </span>
        <span className="chip chip--warm">{t('badge')}</span>
        <h1 className="state-hero__title">{t('title')}</h1>
        <p className="state-hero__lead">{t('lead')}</p>
      </section>

      {editing ? (
        <section className="card">
          <h2 className="card__title">{t('editTitle')}</h2>
          <ApplyForm
            initial={instructor}
            submitLabel={t('save')}
            onDone={() => setEditing(false)}
            onCancel={() => setEditing(false)}
          />
        </section>
      ) : (
        <section className="card">
          <h2 className="card__title">{t('summaryTitle')}</h2>
          <dl className="summary">
            <div className="summary__row">
              <dt>{t('name')}</dt>
              <dd>{instructor.display_name}</dd>
            </div>
            <div className="summary__row">
              <dt>{t('titleLabel')}</dt>
              <dd>{instructor.title || t('empty')}</dd>
            </div>
            {instructor.bio ? (
              <div className="summary__row summary__row--block">
                <dt>{t('bio')}</dt>
                <dd>{instructor.bio}</dd>
              </div>
            ) : null}
            {submitted ? (
              <div className="summary__row">
                <dt>{t('submitted')}</dt>
                <dd>{submitted}</dd>
              </div>
            ) : null}
          </dl>
          <div className="card__actions">
            <Button block onClick={recheck} loading={checking}>
              <Icon name="refresh" size={20} />
              {t('recheck')}
            </Button>
            <Button variant="ghost" block onClick={() => setEditing(true)}>
              <Icon name="edit" size={20} />
              {t('edit')}
            </Button>
          </div>
        </section>
      )}
      <p className="state-note">
        <Icon name="shield" size={18} />
        <span>{t('note')}</span>
      </p>
    </StateFrame>
  );
}
