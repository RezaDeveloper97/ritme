'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCurrentAdmin } from '@/features/auth';
import { formatDate, formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import {
  Button,
  DataTable,
  ErrorState,
  LinkTabs,
  Pagination,
  Panel,
  SearchInput,
  Skeleton,
  toast,
  useNotifyError,
  type Column,
} from '@/shared/ui';

import { instructorsApi, useInstructorAction, type Instructor, type InstructorAction } from '../api/learning';
import { INSTRUCTOR_TABS } from '../lib/review';
import { InstructorBadge, NoteDialog } from './parts';

const tabHref = (status: string) => (status === 'all' ? '/learning/instructors' : `/learning/instructors?status=${status}`);

/** /learning/instructors?status=&q= — applications and instructors; approve / revoke (super admins, B-N8-08). */
export function LearningInstructorsScreen() {
  const t = useTranslations('learning.instructors');
  const locale = useLocale();
  const isSuper = useCurrentAdmin()?.role === 'super';
  const list = useListParams({ status: 'all' });
  const status = list.params.filters.status ?? 'all';
  const query = instructorsApi.useList(list.query);
  const counts = query.data?.counts;
  const n = (v: number) => formatNumber(v, locale);
  const [action, setAction] = useState<{ instructor: Instructor; action: InstructorAction } | null>(null);
  const [details, setDetails] = useState<number | null>(null);

  const columns: Column<Instructor>[] = [
    {
      key: 'name',
      header: t('name'),
      cell: (i) => (
        <span className="flex flex-col items-start">
          <span className="font-semibold" dir="auto">
            {i.display_name}
          </span>
          {i.title ? (
            <span className="text-xs text-ink-3" dir="auto">
              {i.title}
            </span>
          ) : null}
        </span>
      ),
    },
    {
      key: 'account',
      header: t('account'),
      cell: (i) => (
        <span className="flex flex-col items-start">
          <Link href={`/users/${i.user.id}`} className="text-brand no-underline hover:underline" dir="auto">
            {i.user.name || t('noName')}
          </Link>
          {i.user.mobile ? (
            <span dir="ltr" className="text-xs text-muted">
              {i.user.mobile}
            </span>
          ) : null}
        </span>
      ),
    },
    { key: 'status', header: t('status'), cell: (i) => <InstructorBadge status={i.status} /> },
    {
      key: 'courses',
      header: t('courses'),
      cell: (i) =>
        i.courses_count > 0 ? (
          <Link href={`/learning/courses?instructor_id=${i.id}`} className="text-brand no-underline hover:underline">
            {t('coursesCount', { published: n(i.published_courses), total: n(i.courses_count) })}
          </Link>
        ) : (
          <span className="text-muted">—</span>
        ),
      className: 'whitespace-nowrap',
    },
    { key: 'students', header: t('students'), cell: (i) => n(i.students_count), className: 'cell-num' },
    {
      key: 'date',
      header: t('appliedAt'),
      cell: (i) => formatDate(i.created_at, locale),
      className: 'cell-num whitespace-nowrap text-ink-3',
    },
    {
      key: 'actions',
      header: <span className="sr-only">{t('actions')}</span>,
      cell: (i) => (
        <div className="row-actions">
          <Button size="sm" variant="ghost" onClick={() => setDetails(i.id)}>
            {t('details')}
          </Button>
          {isSuper && i.status !== 'approved' ? (
            <Button size="sm" variant="primary" onClick={() => setAction({ instructor: i, action: 'approve' })}>
              {t('approve')}
            </Button>
          ) : null}
          {isSuper && i.status !== 'revoked' ? (
            <Button size="sm" variant="danger" onClick={() => setAction({ instructor: i, action: 'revoke' })}>
              {i.status === 'pending' ? t('reject') : t('revoke')}
            </Button>
          ) : null}
        </div>
      ),
      className: 'cell-actions',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <LinkTabs
        label={t('title')}
        items={INSTRUCTOR_TABS.map((tab) => ({
          key: tab,
          href: tabHref(tab),
          label: counts ? `${t(`tabs.${tab}`)} (${n(counts[tab] ?? 0)})` : t(`tabs.${tab}`),
          active: tab === status,
        }))}
      />
      <Panel title={t('title')} bodyClassName="">
        <div className="filter-bar">
          <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
        </div>
        <p className="panel-note">{isSuper ? t('intro') : t('introEditor')}</p>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(i) => i.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={list.params.q || status !== 'all' ? t('emptyFiltered') : t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
      <InstructorActionDialog current={action} onClose={() => setAction(null)} />
      <InstructorDetailsDialog id={details} onClose={() => setDetails(null)} />
    </div>
  );
}

function InstructorActionDialog({
  current,
  onClose,
}: {
  current: { instructor: Instructor; action: InstructorAction } | null;
  onClose: () => void;
}) {
  const t = useTranslations('learning.instructors');
  const notifyError = useNotifyError();
  const mutation = useInstructorAction();
  const action = current?.action ?? 'approve';
  const pending = current?.instructor.status === 'pending';
  const name = current?.instructor.display_name ?? '';
  return (
    <NoteDialog
      open={current !== null}
      title={action === 'approve' ? t('approveTitle', { name }) : pending ? t('rejectTitle', { name }) : t('revokeTitle', { name })}
      message={action === 'approve' ? t('approveMessage') : t('revokeMessage')}
      submitLabel={action === 'approve' ? t('approve') : pending ? t('reject') : t('revoke')}
      required={false}
      tone={action === 'approve' ? 'primary' : 'danger'}
      saving={mutation.isPending}
      onClose={onClose}
      onSubmit={(note) =>
        current &&
        mutation.mutate(
          { id: current.instructor.id, action, note },
          {
            onSuccess: () => {
              toast.success(action === 'approve' ? t('approvedToast') : t('revokedToast'));
              onClose();
            },
            onError: notifyError,
          },
        )
      }
    />
  );
}

/** Bio and moderation history of one instructor, read in a dialog. */
function InstructorDetailsDialog({ id, onClose }: { id: number | null; onClose: () => void }) {
  const t = useTranslations('learning.instructors');
  const th = useTranslations('learning.history');
  const locale = useLocale();
  const query = instructorsApi.useDetail(id);
  const data = query.data;

  return (
    <dialog
      className="dialog"
      aria-label={t('details')}
      ref={(el) => {
        if (!el) return;
        if (id !== null && !el.open) el.showModal();
        if (id === null && el.open) el.close();
      }}
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
    >
      {id !== null ? (
        <div className="flex flex-col gap-4 p-5">
          {query.error && !data ? (
            <ErrorState error={query.error} onRetry={() => query.refetch()} />
          ) : !data ? (
            <div className="flex flex-col gap-2" aria-busy="true">
              <Skeleton className="h-6 w-48" />
              <Skeleton className="h-16 w-full" />
            </div>
          ) : (
            <>
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="m-0 text-base font-bold" dir="auto">
                  {data.instructor.display_name}
                </h2>
                <InstructorBadge status={data.instructor.status} />
              </div>
              <p className="m-0 whitespace-pre-wrap break-words text-ink-3" dir="auto">
                {data.instructor.bio || t('noBio')}
              </p>
              {data.instructor.approved_by ? (
                <p className="m-0 text-xs text-muted">
                  {t('approvedBy', {
                    name: data.instructor.approved_by.name ?? '—',
                    date: formatDate(data.instructor.approved_at, locale),
                  })}
                </p>
              ) : null}
              <div>
                <h3 className="m-0 mb-2 text-sm font-bold">{th('title')}</h3>
                {data.history.length === 0 ? (
                  <p className="m-0 text-sm text-muted">{th('empty')}</p>
                ) : (
                  <ul className="m-0 flex list-none flex-col gap-2 p-0">
                    {data.history.map((h) => (
                      <li key={h.id} className="flex flex-col gap-0.5 text-sm">
                        <span>
                          <span className="font-semibold">
                            {th.has(`action.${h.action}` as 'action.instructor.approve')
                              ? th(`action.${h.action}` as 'action.instructor.approve')
                              : h.action}
                          </span>
                          {' · '}
                          <span className="text-ink-3">{h.admin?.name ?? '—'}</span>
                          {' · '}
                          <span className="tabular-nums text-muted">{formatDateTime(h.created_at, locale)}</span>
                        </span>
                        {h.note ? (
                          <span className="text-ink-3" dir="auto">
                            {h.note}
                          </span>
                        ) : null}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </>
          )}
          <div className="flex justify-end">
            <Button onClick={onClose} autoFocus>
              {t('close')}
            </Button>
          </div>
        </div>
      ) : null}
    </dialog>
  );
}
