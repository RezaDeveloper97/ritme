'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { useCourses } from '@/entities/course';
import {
  GrantRow,
  GrantStatusPill,
  RevokeButton,
  deleteGroup,
  phoneLabel,
  progressText,
  studentKeys,
  useGroup,
} from '@/entities/student';
import { GroupForm } from '@/features/group-form';
import { useErrorMessage } from '@/shared/i18n';
import { toPersianDigits } from '@/shared/lib';
import { Button, Icon, QueryError, Skeleton } from '@/shared/ui';

/** /groups/{id} — nbl_Ins_Group: courses of the group, members with progress, add member, rename / delete. */
export function GroupScreen({ id }: { id: number }) {
  const t = useTranslations('group');
  const errorMessage = useErrorMessage();
  const router = useRouter();
  const qc = useQueryClient();
  const { data, isPending, isError, error, refetch } = useGroup(id);
  const courses = useCourses();
  const [editing, setEditing] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const remove = useMutation({
    mutationFn: () => deleteGroup(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: studentKeys.all });
      router.push('/groups');
    },
  });

  if (isPending) {
    return (
      <div className="page" aria-busy="true">
        <Skeleton className="skeleton--line" />
        <Skeleton className="skeleton--card" />
        <Skeleton className="skeleton--card" />
      </div>
    );
  }
  if (isError) {
    return (
      <div className="page">
        <QueryError error={error} onRetry={() => void refetch()} />
      </div>
    );
  }

  const { group, members, pending_signups: pending } = data;
  const options = (courses.data?.courses ?? [])
    .filter((c) => c.kind === 'course')
    .map((c) => ({ id: c.id, title: c.title, published: c.status === 'published' }));

  return (
    <div className="page">
      <header className="page-head">
        <Link href="/groups" className="icon-btn" aria-label={t('back')}>
          <Icon name="back" />
        </Link>
        <div className="page-head__text">
          <h1 className="page-head__title">{group.name}</h1>
          <p className="page-head__sub">{t('membersCount', { count: toPersianDigits(members.length) })}</p>
        </div>
      </header>

      {editing ? (
        <GroupForm
          groupId={group.id}
          initialName={group.name}
          initialCourseIds={group.courses.map((c) => c.id)}
          courses={options}
          onCancel={() => setEditing(false)}
          onSaved={() => setEditing(false)}
        />
      ) : (
        <section className="panel-card">
          <div className="panel-card__head">
            <h2 className="panel-card__title">{t('courses')}</h2>
            <button type="button" className="pill-btn" onClick={() => setEditing(true)}>
              <Icon name="edit" size={16} />
              {t('edit')}
            </button>
          </div>
          {group.courses.length === 0 ? (
            <p className="panel-card__empty">{t('noCourses')}</p>
          ) : (
            <div className="chips">
              {group.courses.map((c) => (
                <span key={c.id} className="access-chip access-chip--bloom">
                  {c.title}
                </span>
              ))}
            </div>
          )}
        </section>
      )}

      <section className="panel-card">
        <div className="panel-card__head">
          <h2 className="panel-card__title">{t('members')}</h2>
          <Link href={`/students/add?scope=group&target=${group.id}`} className="panel-card__link">
            <Icon name="plus" size={16} />
            {t('addMember')}
          </Link>
        </div>
        {members.length === 0 ? (
          <p className="panel-card__empty">{t('noMembers')}</p>
        ) : (
          <ul className="row-list">
            {members.map((m) => (
              <GrantRow
                key={m.id}
                grant={m}
                meta={[m.student_name ? phoneLabel(m.phone) : null, progressText(m)].filter(Boolean).join(' · ') || t('noProgress')}
                trailing={
                  <>
                    <GrantStatusPill grant={m} />
                    <RevokeButton grantId={m.id} />
                  </>
                }
              />
            ))}
          </ul>
        )}
        {pending > 0 ? (
          <p className="note-card">{t('pendingNote', { count: toPersianDigits(pending) })}</p>
        ) : null}
      </section>

      <section className="panel-card">
        {confirmDelete ? (
          <div className="form-card__actions">
            <span className="field__label">{t('deleteAsk')}</span>
            <Button loading={remove.isPending} onClick={() => remove.mutate()}>
              {t('deleteYes')}
            </Button>
            <Button variant="text" onClick={() => setConfirmDelete(false)}>
              {t('cancel')}
            </Button>
            {remove.isError ? <p className="form-error">{errorMessage(remove.error)}</p> : null}
          </div>
        ) : (
          <button type="button" className="pill-btn pill-btn--danger" onClick={() => setConfirmDelete(true)}>
            <Icon name="trash" size={16} />
            {t('delete')}
          </button>
        )}
      </section>
    </div>
  );
}
