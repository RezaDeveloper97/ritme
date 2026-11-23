'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import {
  BodySkeleton,
  DoctorAvatar,
  LoadError,
  formatClock,
  usePrescription,
  useBooking,
  useDoctorTitle,
  useTelemedError,
  useThread,
  useVideoLink,
  whenLabel,
  type ChatMessage,
} from '@/entities/doctor';
import { useAddPrescriptionReminder, useSendMessage } from '@/features/book-visit';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import { Card, EmptyState, HeaderButton, Icon, PrimaryButton, SkyLayer } from '@/shared/ui';
import '@/shared/ui/styles/dr.css';

const clock = (iso: string, locale: Locale) => {
  const d = new Date(iso);
  return formatClock(`${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`, locale);
};

/**
 * `/services/bookings/[id]/chat` — encrypted-in-transit thread with the doctor (nbl_/nbd_v17_Chat): polling thread,
 * attachments as links, the e-prescription with «افزودن به یادآور» per drug, the video join button while the window
 * is open, and a composer that locks when the chat window closes.
 */
export function DoctorChatPage({ bookingId }: { bookingId: number }) {
  const t = useTranslations('doctors');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const title = useDoctorTitle();
  const errorText = useTelemedError();
  const booking = useBooking(bookingId);
  const thread = useThread(bookingId);
  const rx = usePrescription(bookingId);
  const send = useSendMessage(bookingId);
  const addReminder = useAddPrescriptionReminder(bookingId);
  const video = useVideoLink(bookingId, booking.data?.mode === 'video' && booking.data.status === 'confirmed');
  const [text, setText] = useState('');
  const endRef = useRef<HTMLDivElement>(null);
  const count = thread.data?.items.length ?? 0;
  useEffect(() => {
    endRef.current?.scrollIntoView({ block: 'end' });
  }, [count]);
  const back = () => router.push(`/services/bookings/${bookingId}`);

  if (booking.isPending || (thread.isPending && !thread.isError)) {
    return (
      <div className="view dr-view dr-chat">
        <SkyLayer />
        <div className="scroll"><BodySkeleton rows={4} /></div>
      </div>
    );
  }
  if (booking.isError || thread.isError || !booking.data || !thread.data) {
    const gone = getApiErrorStatus(booking.error) === 404 || getApiErrorStatus(thread.error) === 404;
    return (
      <div className="view dr-view dr-chat">
        <SkyLayer />
        <div className="scroll">
          <div className="dr-body">
            {gone ? (
              <Card>
                <EmptyState icon="chat" title={t('chat.unavailable')} action={<PrimaryButton onClick={() => router.push('/services/bookings')}>{t('bookings.title')}</PrimaryButton>} />
              </Card>
            ) : (
              <LoadError onRetry={() => { void booking.refetch(); void thread.refetch(); }} busy={thread.isFetching} />
            )}
          </div>
        </div>
      </div>
    );
  }

  const b = booking.data;
  const open = thread.data.chat.open;
  const submit = () => {
    const body = text.trim();
    if (!body || send.isPending) return;
    send.mutate(body, { onSuccess: () => setText('') });
  };
  const bubble = (m: ChatMessage) => (
    <div key={m.id} className={clsx('dr-msg', m.mine ? 'is-mine' : 'is-theirs')}>
      <div className="dr-bubble">
        {m.body ? <p>{m.body}</p> : null}
        {m.attachment ? (
          <a className="dr-attach" href={m.attachment.url} target="_blank" rel="noopener noreferrer">
            <Icon name="note" size={18} />
            <span>{t('chat.attachment', { kb: formatNumber(Math.max(1, Math.round(m.attachment.size_bytes / 1024)), locale) })}</span>
          </a>
        ) : null}
      </div>
      <time className="dr-time" dateTime={m.created_at}>{clock(m.created_at, locale)}</time>
    </div>
  );

  return (
    <div className="view dr-view dr-chat">
      <header className="dr-chat-head">
        <HeaderButton label={t('common.back')} icon="arrowR" onClick={back} />
        <DoctorAvatar name={b.doctor.name} photoUrl={b.doctor.photo_url} size="sm" />
        <div className="dr-chat-id">
          <strong>{title(b.doctor)}</strong>
          <span>{open ? t('chat.open') : t('chat.closed')}</span>
        </div>
        {video.data?.active && video.data.url ? (
          <a className="dr-join" href={video.data.url} target="_blank" rel="noopener noreferrer" aria-label={t('chat.join')}>
            <Icon name="video" size={20} />
          </a>
        ) : (
          <span className="dr-join is-off" aria-hidden><Icon name="video" size={20} /></span>
        )}
      </header>

      <div className="scroll dr-thread">
        <Link href={`/services/bookings/${b.id}`} className="dr-thread-ctx">
          {`${t(`mode.${b.mode}`)} · ${whenLabel(b.date, b.time, locale)} · `}
          <b>{t('chat.details')}</b>
        </Link>
        {video.data?.active && video.data.url ? (
          <a className="dr-joinbar" href={video.data.url} target="_blank" rel="noopener noreferrer">
            <Icon name="video" size={18} />
            {t('chat.join')}
          </a>
        ) : null}
        {thread.data.items.length === 0 ? <p className="dr-fine is-center">{t('chat.empty')}</p> : thread.data.items.map(bubble)}

        {rx.data && rx.data.items.length > 0 ? (
          <section className="dr-rx" aria-label={t('chat.rx')}>
            <h2 className="dr-h3">{t('chat.rx')}</h2>
            {rx.data.items.map((it) => (
              <div key={it.id} className="dr-rx-item">
                <span className="dr-rx-icon"><Icon name="pill" size={20} /></span>
                <span className="dr-rx-text">
                  <strong>{it.drug}</strong>
                  <span>{[it.dose, it.times.length ? t('chat.perDay', { count: formatNumber(it.times.length, locale) }) : null, it.instructions].filter(Boolean).join(' · ')}</span>
                </span>
                {it.reminder_id ? (
                  <span className="dr-rx-done"><Icon name="check" size={14} />{t('chat.added')}</span>
                ) : (
                  <button type="button" className="dr-link" disabled={addReminder.isPending} {...pilot(`doctor-chat.add-reminder:${it.id}`)} onClick={() => addReminder.mutate(it.id)}>
                    {t('chat.addReminder')}
                  </button>
                )}
              </div>
            ))}
            {addReminder.isError ? <p role="alert" className="dr-error">{errorText(addReminder.error)}</p> : null}
          </section>
        ) : null}
        <p className="dr-fine is-center">{t('chat.encrypted')}</p>
        <div ref={endRef} />
      </div>

      <form
        className="dr-composer"
        onSubmit={(e) => {
          e.preventDefault();
          submit();
        }}
      >
        {send.isError ? <p role="alert" className="dr-error">{errorText(send.error)}</p> : null}
        <div className="dr-composer-row">
          <input
            className="dr-input"
            value={text}
            maxLength={2000}
            disabled={!open}
            onChange={(e) => setText(e.target.value)}
            placeholder={open ? t('chat.placeholder') : t('chat.closedNote')}
            aria-label={t('chat.placeholder')}
          />
          <button type="submit" className="dr-send" disabled={!open || !text.trim() || send.isPending} aria-label={t('chat.send')}>
            <Icon name="send" size={20} className="dr-flip" />
          </button>
        </div>
      </form>
    </div>
  );
}
