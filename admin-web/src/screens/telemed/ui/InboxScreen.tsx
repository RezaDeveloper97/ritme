'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState, type FormEvent } from 'react';

import { isApiError } from '@/shared/api';
import { cn, formatDateTime } from '@/shared/lib';
import {
  Badge,
  Button,
  ErrorState,
  PageHeader,
  Panel,
  Select,
  Skeleton,
  TextArea,
  TextInput,
  confirm,
  toast,
  useNotifyError,
} from '@/shared/ui';

import {
  useInboxBookings,
  usePrescription,
  useRxWrite,
  useSendMessage,
  useThread,
  useVideo,
  type InboxBooking,
} from '../api/telemed';
import { bookingTone, PRESCRIPTION_FORMS } from '../lib/format';

/** /telemed/inbox — the signed-in doctor's own visits: chat, prescription, video link (admin-api.md §19b). */
export function InboxScreen() {
  const t = useTranslations('telemed.inbox');
  const locale = useLocale();
  const query = useInboxBookings();
  const [selected, setSelected] = useState<number | null>(null);

  if (query.error && isApiError(query.error) && query.error.code === 'doctor_required') {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader title={t('title')} />
        <Panel>
          <p className="m-0 text-ink-3">{t('notADoctor')}</p>
        </Panel>
      </div>
    );
  }
  const items = query.data?.items ?? [];
  const current = items.find((b) => b.id === selected) ?? null;

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('title')} />
      {query.error && !query.data ? (
        <Panel>
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        </Panel>
      ) : (
        <div className="grid gap-4 lg:grid-cols-[minmax(0,320px)_1fr]">
          <Panel title={t('visits')} bodyClassName="p-2">
            {query.isPending ? (
              <div className="flex flex-col gap-2 p-2" aria-busy="true">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : items.length === 0 ? (
              <p className="m-0 p-3 text-sm text-muted">{t('empty')}</p>
            ) : (
              <ul className="m-0 flex list-none flex-col gap-1 p-0">
                {items.map((b) => (
                  <li key={b.id}>
                    <button
                      type="button"
                      onClick={() => setSelected(b.id)}
                      aria-current={b.id === selected ? 'true' : undefined}
                      className={cn(
                        'flex min-h-11 w-full flex-col items-start gap-1 rounded-lg border border-transparent bg-transparent p-3 text-start',
                        b.id === selected ? 'bg-brand-soft border-brand' : 'hover:bg-surface-2',
                      )}
                    >
                      <span className="flex w-full items-center justify-between gap-2">
                        <span dir="ltr" className="font-mono text-xs">
                          {b.reference}
                        </span>
                        <Badge tone={bookingTone(b.status)}>{t(`status.${b.status}` as 'status.confirmed')}</Badge>
                      </span>
                      <span className="text-xs text-ink-3">
                        {t(`mode.${b.mode}` as 'mode.video')} · {formatDateTime(b.starts_at, locale)}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </Panel>
          {current ? <VisitPanel key={current.id} booking={current} /> : <Panel><p className="m-0 text-muted">{t('pick')}</p></Panel>}
        </div>
      )}
    </div>
  );
}

function VisitPanel({ booking }: { booking: InboxBooking }) {
  return (
    <div className="flex flex-col gap-4">
      <ChatPanel booking={booking} />
      <PrescriptionPanel booking={booking} />
      {booking.mode === 'video' ? <VideoPanel id={booking.id} /> : null}
    </div>
  );
}

function ChatPanel({ booking }: { booking: InboxBooking }) {
  const t = useTranslations('telemed.inbox');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const thread = useThread(booking.id);
  const send = useSendMessage(booking.id);
  const [text, setText] = useState('');
  const open = thread.data?.chat.open ?? booking.chat_open;

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const body = text.trim();
    if (!body || send.isPending) return;
    send.mutate(body, { onSuccess: () => setText(''), onError: notifyError });
  };

  return (
    <Panel title={t('chat')}>
      <div className="flex flex-col gap-3">
        {thread.isPending ? (
          <Skeleton className="h-24 w-full" />
        ) : thread.error ? (
          <ErrorState error={thread.error} onRetry={() => thread.refetch()} />
        ) : (
          <ul className="m-0 flex max-h-96 list-none flex-col gap-2 overflow-y-auto p-0" aria-live="polite">
            {thread.data.items.length === 0 ? <li className="text-sm text-muted">{t('noMessages')}</li> : null}
            {thread.data.items.map((m) => (
              <li
                key={m.id}
                className={cn(
                  'flex max-w-[85%] flex-col gap-1 rounded-xl px-3 py-2',
                  m.mine ? 'self-end bg-brand-soft' : 'self-start bg-surface-2',
                )}
              >
                {m.body ? (
                  <span className="whitespace-pre-wrap break-words" dir="auto">
                    {m.body}
                  </span>
                ) : null}
                {m.attachment?.url ? (
                  <a href={m.attachment.url} target="_blank" rel="noopener noreferrer" className="text-brand">
                    {t('attachment')}
                  </a>
                ) : null}
                <span className="text-xs text-muted">{formatDateTime(m.created_at, locale)}</span>
              </li>
            ))}
          </ul>
        )}
        {open ? (
          <form className="flex flex-col gap-2" onSubmit={submit}>
            <TextArea label={t('reply')} value={text} maxLength={2000} rows={3} onChange={(e) => setText(e.target.value)} />
            <div>
              <Button type="submit" variant="primary" loading={send.isPending} disabled={!text.trim()}>
                {t('send')}
              </Button>
            </div>
          </form>
        ) : (
          <p className="m-0 text-sm text-muted">{t('chatClosed')}</p>
        )}
      </div>
    </Panel>
  );
}

function PrescriptionPanel({ booking }: { booking: InboxBooking }) {
  const t = useTranslations('telemed.inbox');
  const notifyError = useNotifyError();
  const rx = usePrescription(booking.id);
  const write = useRxWrite(booking.id);
  const [drug, setDrug] = useState('');
  const [dose, setDose] = useState('');
  const [form, setForm] = useState<string>(PRESCRIPTION_FORMS[0]);
  const [times, setTimes] = useState('08:00');
  const [days, setDays] = useState('');
  const [instructions, setInstructions] = useState('');
  const open = booking.chat_open;

  const add = (e: FormEvent) => {
    e.preventDefault();
    write.mutate(
      {
        body: {
          drug: drug.trim(),
          dose: dose.trim() || null,
          form,
          times: times.split(',').map((x) => x.trim()).filter(Boolean),
          duration_days: days ? Number(days) : null,
          instructions: instructions.trim() || null,
        },
      },
      {
        onSuccess: () => {
          setDrug('');
          setDose('');
          setDays('');
          setInstructions('');
          toast.success(t('rxAdded'));
        },
        onError: notifyError,
      },
    );
  };
  const remove = async (item: number) => {
    if (!(await confirm({ message: t('rxConfirmDelete'), confirmLabel: t('rxDelete'), tone: 'danger' }))) return;
    write.mutate({ item }, { onError: notifyError });
  };

  return (
    <Panel title={t('prescription')}>
      <div className="flex flex-col gap-3">
        {rx.isPending ? (
          <Skeleton className="h-12 w-full" />
        ) : rx.error ? (
          <ErrorState error={rx.error} onRetry={() => rx.refetch()} />
        ) : rx.data.items.length === 0 ? (
          <p className="m-0 text-sm text-muted">{t('rxEmpty')}</p>
        ) : (
          <ul className="m-0 flex list-none flex-col gap-2 p-0">
            {rx.data.items.map((it) => (
              <li key={it.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-line p-3">
                <span className="flex flex-col" dir="auto">
                  <span className="font-semibold">
                    {it.drug}
                    {it.dose ? ` · ${it.dose}` : ''}
                  </span>
                  <span className="text-xs text-ink-3">
                    {it.form ? t(`forms.${it.form}` as 'forms.tablet') : ''} · {it.times.join(' ، ')}
                    {it.duration_days ? ` · ${t('rxDays', { count: it.duration_days })}` : ''}
                  </span>
                </span>
                {open ? (
                  <Button size="sm" variant="ghost" onClick={() => remove(it.id)}>
                    {t('rxDelete')}
                  </Button>
                ) : null}
              </li>
            ))}
          </ul>
        )}
        {open ? (
          <form className="grid gap-3 sm:grid-cols-2" onSubmit={add}>
            <TextInput label={t('drug')} value={drug} maxLength={255} required onChange={(e) => setDrug(e.target.value)} />
            <TextInput label={t('dose')} value={dose} maxLength={50} onChange={(e) => setDose(e.target.value)} />
            <Select label={t('form')} value={form} onChange={(e) => setForm(e.target.value)} options={PRESCRIPTION_FORMS.map((f) => ({ value: f, label: t(`forms.${f}` as 'forms.tablet') }))} />
            <TextInput label={t('times')} hint={t('timesHint')} value={times} dir="ltr" required onChange={(e) => setTimes(e.target.value)} />
            <TextInput label={t('durationDays')} type="number" value={days} onChange={(e) => setDays(e.target.value)} />
            <TextInput label={t('instructions')} value={instructions} onChange={(e) => setInstructions(e.target.value)} />
            <div className="sm:col-span-2">
              <Button type="submit" variant="primary" loading={write.isPending} disabled={!drug.trim()}>
                {t('rxAdd')}
              </Button>
            </div>
          </form>
        ) : null}
      </div>
    </Panel>
  );
}

function VideoPanel({ id }: { id: number }) {
  const t = useTranslations('telemed.inbox');
  const locale = useLocale();
  const video = useVideo(id);
  return (
    <Panel title={t('video')}>
      {video.isPending ? (
        <Skeleton className="h-8 w-48" />
      ) : video.error ? (
        <ErrorState error={video.error} onRetry={() => video.refetch()} />
      ) : video.data.video.active && video.data.video.url ? (
        <a href={video.data.video.url} target="_blank" rel="noopener noreferrer" className="btn btn-primary">
          {t('joinVideo')}
        </a>
      ) : (
        <p className="m-0 text-sm text-muted">
          {video.data.video.opens_at ? t('videoOpens', { time: formatDateTime(video.data.video.opens_at, locale) }) : t('videoClosed')}
        </p>
      )}
    </Panel>
  );
}
