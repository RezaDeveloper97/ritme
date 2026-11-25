'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { localizeFeedText, slotItems, telHref, useAckAlert, useMessageFeed, type FeedItem } from '@/entities/message';
import type { Locale } from '@/shared/i18n';
import { PrimaryButton, UrgentCard } from '@/shared/ui';
import '@/shared/ui/styles/del.css';

const keyOf = (item: Pick<FeedItem, 'id' | 'alertId'>) => `${item.id}:${item.alertId ?? ''}`;

/**
 * The urgent health alert (SPEC §5.4.1), over whatever screen is showing — the home, the pregnancy home, the
 * analysis page, an open log sheet. It used to live inside the cycle home only, so an alert a save raised elsewhere
 * (spotting logged in pregnancy) appeared only later, on the next visit to that home (even after a switch to another
 * mode). Mounted once in the root layout; it reads the same feed cache as the bell, which every log save invalidates,
 * so the alert shows the moment the save answers.
 */
export function UrgentAlertHost() {
  const feed = useMessageFeed().data ?? null;
  const ack = useAckAlert();
  // Acknowledged this session: hidden at once, before the feed refetch drops the item.
  const [acked, setAcked] = useState<ReadonlySet<string>>(() => new Set());
  const item = slotItems(feed, 'urgent_modal').find((i) => !acked.has(keyOf(i))) ?? null;
  if (!item) return null;
  return (
    <UrgentModal
      key={keyOf(item)}
      item={item}
      pending={ack.isPending}
      failed={ack.isError}
      onAck={() =>
        ack.mutate(
          { ruleId: item.id, alertId: item.alertId },
          { onSuccess: () => setAcked((prev) => new Set(prev).add(keyOf(item))) },
        )
      }
    />
  );
}

interface UrgentModalProps {
  item: FeedItem;
  /** Posts `POST /alerts/{id}/ack`; the modal closes when the feed no longer carries the item. */
  onAck: () => void;
  pending: boolean;
  failed: boolean;
}

/**
 * The urgent health message (SPEC §5.4.1). It cannot be closed: no backdrop tap, no Escape, no close button; the
 * only way out is «دیدم», which acknowledges the alert on the server. The call button (115 or the doctor) is a
 * plain `tel:` link, always available.
 */
function UrgentModal({ item, onAck, pending, failed }: UrgentModalProps) {
  const t = useTranslations('alerts');
  const loc = useLocale() as Locale;
  const dialogRef = useRef<HTMLDivElement>(null);
  const contact = item.evidence?.contact ?? '';
  const tel = telHref(contact);
  const callLabel = t('actions.call', { contact: localizeFeedText(contact, loc) });

  // Focus lands inside the dialog; Escape is swallowed so it cannot dismiss anything underneath.
  useEffect(() => {
    dialogRef.current?.querySelector<HTMLElement>('.nb-urgent-action button')?.focus({ preventScroll: true });
    const block = (e: KeyboardEvent) => {
      if (e.key === 'Escape') e.preventDefault();
    };
    window.addEventListener('keydown', block);
    return () => window.removeEventListener('keydown', block);
  }, []);

  return (
    <div ref={dialogRef} role="alertdialog" aria-modal="true" aria-label={item.title} className="del-backdrop is-urgent" data-feed-slot="urgent_modal">
      <div className="del-card">
        <UrgentCard
          title={localizeFeedText(item.title, loc)}
          hotlines={
            tel
              ? [{ number: tel.slice(4), label: callLabel, display: localizeFeedText(contact, loc) }]
              : undefined
          }
          hotlinesLabel={callLabel}
          action={
            <PrimaryButton loading={pending} onClick={onAck}>
              {t('actions.ack')}
            </PrimaryButton>
          }
        >
          <p>{localizeFeedText(item.body, loc)}</p>
          {!tel && contact ? <p>{localizeFeedText(contact, loc)}</p> : null}
          <p>{t('urgentHint')}</p>
          {item.evidence?.seen ? (
            <p>
              <b>{t('whatWeSaw')}: </b>
              {localizeFeedText(item.evidence.seen, loc)}
            </p>
          ) : null}
          {item.evidence?.confidence ? (
            <p>
              <b>{t('howSure')}: </b>
              {localizeFeedText(item.evidence.confidence, loc)}
            </p>
          ) : null}
        </UrgentCard>
        {failed ? (
          <p className="del-error" role="alert">
            {t('saveError')}
          </p>
        ) : null}
      </div>
    </div>
  );
}
