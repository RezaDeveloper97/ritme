'use client';

import { useTranslations } from 'next-intl';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { useCourses } from '@/entities/course';
import { useGroups } from '@/entities/student';
import { GroupForm } from '@/features/group-form';
import { toPersianDigits } from '@/shared/lib';
import { Button, Icon, QueryError, Skeleton } from '@/shared/ui';

/** /groups — nbl_Ins_Groups: every group with member count, courses and pending note; create. */
export function GroupsScreen() {
  const t = useTranslations('groups');
  const router = useRouter();
  const { data, isPending, isError, error, refetch } = useGroups();
  const courses = useCourses();
  const [creating, setCreating] = useState(false);

  const groups = data?.groups ?? [];
  const members = groups.reduce((n, g) => n + g.members.active + g.members.pending, 0);
  const options = (courses.data?.courses ?? [])
    .filter((c) => c.kind === 'course')
    .map((c) => ({ id: c.id, title: c.title, published: c.status === 'published' }));

  return (
    <div className="page">
      <header className="page-head">
        <div className="page-head__text">
          <h1 className="page-head__title">{t('title')}</h1>
          {data ? (
            <p className="page-head__sub">
              {t('summary', { groups: toPersianDigits(groups.length), members: toPersianDigits(members) })}
            </p>
          ) : null}
        </div>
        <Link href="/students/add" className="icon-btn" aria-label={t('addStudents')} title={t('addStudents')}>
          <Icon name="userPlus" />
        </Link>
      </header>

      {isPending ? (
        <Skeleton className="skeleton--card" />
      ) : isError ? (
        <QueryError error={error} onRetry={() => void refetch()} />
      ) : (
        <section className="panel-card">
          {groups.length === 0 ? (
            <p className="panel-card__empty">{t('empty')}</p>
          ) : (
            <ul className="row-list">
              {groups.map((g) => {
                const total = g.members.active + g.members.pending;
                const meta = [t('membersCount', { count: toPersianDigits(total) })];
                if (g.members.pending > 0) meta.push(t('pendingCount', { count: toPersianDigits(g.members.pending) }));
                return (
                  <li key={g.id}>
                    <Link href={`/groups/${g.id}`} className="row">
                      <span className="kind-tile kind-tile--sm kind-tile--draft" aria-hidden="true">
                        <Icon name="groups" size={22} />
                      </span>
                      <span className="row__body">
                        <span className="row__title">{g.name}</span>
                        <span className="row__meta">{meta.join(' · ')}</span>
                        {g.courses.length > 0 ? (
                          <span className="chips chips--tight">
                            {g.courses.map((c) => (
                              <span key={c.id} className="access-chip access-chip--brand">
                                {c.title}
                              </span>
                            ))}
                          </span>
                        ) : null}
                      </span>
                    </Link>
                  </li>
                );
              })}
            </ul>
          )}
        </section>
      )}

      {creating ? (
        <GroupForm
          courses={options}
          onCancel={() => setCreating(false)}
          onSaved={(d) => router.push(`/groups/${d.group.id}`)}
        />
      ) : (
        <Button variant="ghost" onClick={() => setCreating(true)}>
          <Icon name="plus" size={18} />
          {t('newGroup')}
        </Button>
      )}
    </div>
  );
}
