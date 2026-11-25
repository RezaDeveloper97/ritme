'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import {
  EmergencyCardView,
  healthRecordKeys,
  type OwnerEmergencyCard,
  useDisableEmergencyLink,
  useEmergencyCard,
  useEnableEmergencyLink,
  useSaveEmergencyCard,
  useSaveRecordExtras,
} from '@/entities/health-record';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  Card,
  EmptyState,
  InfoNote,
  ListGroup,
  ListRow,
  PrimaryButton,
  QrCode,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  Switch,
} from '@/shared/ui';
import { pilot } from '@/shared/pilot';

import { emergencyLinkUrl } from '../model/forms';
import { ContactSheet, InsuranceSheet } from './EmergencySheets';
import '@/shared/ui/styles/rem.css';

type SheetId = 'contact' | 'insurance' | null;

/**
 * `/record/emergency` «کارت اضطراری» (CB-REC-05, nbl_Rec_Emergency): the owner's card (bloom's CB-PRIV-01
 * `EmergencyCardView`), «نمایش روی صفحه قفل» + «نمایش وضعیت بارداری» (+ the CB-REC-01 allergy flag), the emergency
 * contact and masked insurance editors, and the public link + QR (token shown once, kept in mutation memory only).
 * Back header, no bottom nav.
 */
export function RecordEmergencyPage() {
  const t = useTranslations('record.emergency');
  const router = useRouter();
  const query = useEmergencyCard();
  const [sheet, setSheet] = useState<SheetId>(null);

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="rem-skel">
        <Skeleton shape="card" className="rem-skel-card" />
        <Skeleton shape="card" className="rem-skel-row" />
      </SkeletonGroup>
    );
  } else if (query.isError || !query.data) {
    body = (
      <EmptyState
        icon="warning"
        title={t('loadError')}
        action={
          <PrimaryButton icon="refresh" block={false} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <EmergencyContent card={query.data} onSheet={setSheet} />;
  }

  return (
    <div className="view rem-page">
      <SkyLayer />
      <div className="scroll rem-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.back('/record')} backLabel={t('back')} />
        {body}
      </div>
      {query.data ? (
        <>
          <ContactSheet open={sheet === 'contact'} onClose={() => setSheet(null)} card={query.data} />
          <InsuranceSheet open={sheet === 'insurance'} onClose={() => setSheet(null)} card={query.data} />
        </>
      ) : null}
    </div>
  );
}

function EmergencyContent({ card, onSheet }: { card: OwnerEmergencyCard; onSheet: (s: SheetId) => void }) {
  const t = useTranslations('record.emergency');
  const save = useSaveEmergencyCard();
  const extras = useSaveRecordExtras();
  const queryClient = useQueryClient();
  const enable = useEnableEmergencyLink();
  const disable = useDisableEmergencyLink();
  const lockId = useId();
  const pregId = useId();
  const allergyId = useId();

  const pending = save.isPending ? save.variables : undefined;
  const lockOn = pending?.show_on_lock_screen ?? card.showOnLockScreen;
  const pregOn = pending?.show_pregnancy ?? card.showPregnancy;
  const allergyOn =
    extras.isPending && extras.variables?.allergiesOnEmergencyCard !== undefined
      ? extras.variables.allergiesOnEmergencyCard
      : card.allergiesOnCard;

  const origin = typeof window !== 'undefined' ? window.location.origin : '';
  // the fresh link only while this screen is mounted and the link is still the current one
  const link = enable.data && card.publicLink.enabled ? emergencyLinkUrl(origin, enable.data.path) : null;

  const contact = card.contact;
  const contactValue = contact ? [contact.name, contact.relation].filter(Boolean).join(' · ') : t('notSet');
  const insuranceValue = card.insurance
    ? `${card.insurance.label ? `${card.insurance.label} · ` : ''}⁦${card.insurance.masked}⁩`
    : t('notSet');

  return (
    <>
      <EmergencyCardView card={card}>
        {link ? <QrCode value={link} label={t('qrLabel')} className="rem-qr" /> : null}
      </EmergencyCardView>

      <ListGroup className="rem-list">
        <ListRow
          id={lockId}
          title={t('lockTitle')}
          description={t('lockBody')}
          trailing={
            <Switch
              checked={lockOn}
              labelledBy={`${lockId}-title`}
              disabled={save.isPending}
              onCheckedChange={(next) => save.mutate({ show_on_lock_screen: next })}
              pilot={pilot('recem.lock')}
            />
          }
        />
        <ListRow
          id={pregId}
          title={t('pregnancyToggle')}
          trailing={
            <Switch
              checked={pregOn}
              labelledBy={`${pregId}-title`}
              disabled={save.isPending}
              onCheckedChange={(next) => save.mutate({ show_pregnancy: next })}
              pilot={pilot('recem.pregnancy')}
            />
          }
        />
        <ListRow
          id={allergyId}
          title={t('allergyToggle')}
          trailing={
            <Switch
              checked={allergyOn}
              labelledBy={`${allergyId}-title`}
              disabled={extras.isPending}
              pilot={pilot('recem.allergies')}
              onCheckedChange={(next) =>
                extras.mutate(
                  { allergiesOnEmergencyCard: next },
                  // the card shows or hides the allergies with the flag (and so does the lock screen)
                  { onSuccess: () => void queryClient.invalidateQueries({ queryKey: healthRecordKeys.all }) },
                )
              }
            />
          }
        />
      </ListGroup>
      {save.isError || extras.isError ? (
        <p className="rem-error" role="alert">
          {t('saveError')}
        </p>
      ) : null}

      <ListGroup title={t('details')} className="rem-list">
        <ListRow icon="phone" title={t('contact')} description={contactValue} onClick={() => onSheet('contact')} pilot={pilot('recem.contact')} />
        <ListRow icon="card" title={t('insurance')} description={insuranceValue} onClick={() => onSheet('insurance')} pilot={pilot('recem.insurance')} />
      </ListGroup>

      <PublicLinkCard
        card={card}
        link={link}
        busy={enable.isPending || disable.isPending}
        failed={enable.isError || disable.isError}
        onCreate={() => enable.mutate()}
        onRevoke={() => {
          enable.reset();
          disable.mutate();
        }}
      />

      <InfoNote icon="shield">{t('note')}</InfoNote>
    </>
  );
}

interface PublicLinkCardProps {
  card: OwnerEmergencyCard;
  link: string | null;
  busy: boolean;
  failed: boolean;
  onCreate: () => void;
  onRevoke: () => void;
}

function PublicLinkCard({ card, link, busy, failed, onCreate, onRevoke }: PublicLinkCardProps) {
  const t = useTranslations('record.emergency');
  const loc = useLocale() as Locale;
  const [copied, setCopied] = useState(false);
  const on = card.publicLink.enabled;
  const views = card.publicLink.views;

  const copy = async () => {
    if (!link) return;
    try {
      await navigator.clipboard.writeText(link);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  };

  return (
    <Card className="rem-link">
      <p className="rem-link-title">{t('link')}</p>
      <p className="rem-link-body">{on ? t('linkOn', { count: views, n: formatNumber(views, loc) }) : t('linkOff')}</p>
      {on ? <p className="rem-link-hint">{link ? t('linkOnce') : t('linkHidden')}</p> : null}
      {link ? (
        <output className="rem-link-url" dir="ltr">
          {link}
        </output>
      ) : null}
      <div className="rem-link-btns">
        {link ? (
          <PrimaryButton icon={copied ? 'check' : 'copy'} onClick={() => void copy()}>
            {copied ? t('copied') : t('copy')}
          </PrimaryButton>
        ) : (
          <PrimaryButton icon="qr" loading={busy} onClick={onCreate}>
            {on ? t('linkRotate') : t('linkCreate')}
          </PrimaryButton>
        )}
        {on ? (
          <SecondaryButton variant="text" disabled={busy} onClick={onRevoke}>
            {t('linkRevoke')}
          </SecondaryButton>
        ) : null}
      </div>
      {failed ? (
        <p className="rem-error" role="alert">
          {t('linkError')}
        </p>
      ) : null}
    </Card>
  );
}
