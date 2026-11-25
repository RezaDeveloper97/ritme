'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useId, useState } from 'react';

import {
  BodySkeleton,
  CONSENT_SCOPES,
  DoctorAvatar,
  LoadError,
  ModeTag,
  formatToman,
  startsAtParam,
  useBookingQuote,
  useDoctorTitle,
  useTelemedError,
  whenLabel,
  mode as modeSchema,
  type ConsentScope,
  type ForWhom,
} from '@/entities/doctor';
import { goToGateway, useCreateBooking } from '@/features/book-visit';
import { getApiErrorCode } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { Card, EmptyState, Icon, InfoNote, PillChip, PrimaryButton, ScreenHeader, SkyLayer, Switch, type IconName } from '@/shared/ui';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/dr.css';

const SCOPE_ICON: Record<ConsentScope, IconName> = { cycle_summary: 'chart', bbt_lh: 'thermo', assistant_summaries: 'sparkle' };
const DATE = /^\d{4}-\d{2}-\d{2}$/;
const TIME = /^\d{2}:\d{2}$/;

/**
 * `/services/doctors/[id]/book?mode=&date=&time=` — review & book (nbl_/nbd_v17_ReviewBook): who the visit is for,
 * reason, note, per-scope data sharing (all OFF by default, only for «خودم» visits — QUESTIONS #125), the price
 * summary and «پرداخت و رزرو» (hold → gateway, or straight to the booking when it is free).
 */
export function DoctorBookPage({ doctorId }: { doctorId: number }) {
  const t = useTranslations('doctors');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const params = useSearchParams();
  const title = useDoctorTitle();
  const errorText = useTelemedError();
  const noteId = useId();
  const mode = modeSchema.safeParse(params.get('mode'));
  const date = params.get('date') ?? '';
  const time = params.get('time') ?? '';
  const valid = mode.success && DATE.test(date) && TIME.test(time);
  const q = useBookingQuote(doctorId, mode.success ? mode.data : null);
  const create = useCreateBooking();

  const [forWhom, setForWhom] = useState<ForWhom>('self');
  const [childId, setChildId] = useState<number>();
  const [patient, setPatient] = useState('');
  const [reason, setReason] = useState<string>();
  const [note, setNote] = useState('');
  const [share, setShare] = useState<Record<ConsentScope, boolean>>({ cycle_summary: false, bbt_lh: false, assistant_summaries: false });

  const back = () => router.back(`/services/doctors/${doctorId}`);
  const shell = (body: React.ReactNode, footer?: React.ReactNode) => (
    <div className="view dr-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('book.title')}
          onBack={back}
          backLabel={t('common.back')}
          action={
            <span className="dr-secure">
              <Icon name="lock" size={14} />
              {t('book.secure')}
            </span>
          }
        />
        {body}
        {footer}
        <div className="page-tail" />
      </div>
    </div>
  );

  if (!valid || !mode.success) {
    return shell(
      <div className="dr-body">
        <Card>
          <EmptyState icon="calendar" title={t('book.invalidTitle')} body={t('book.invalidBody')} action={<PrimaryButton onClick={back}>{t('book.pickAgain')}</PrimaryButton>} />
        </Card>
      </div>,
    );
  }
  if (q.isPending) return shell(<BodySkeleton rows={4} />);
  if (q.isError) return shell(<div className="dr-body"><LoadError onRetry={() => void q.refetch()} busy={q.isFetching} /></div>);

  const quote = q.data;
  const hasChildren = quote.children.length > 0;
  const options: { value: ForWhom; label: string }[] = [
    { value: 'self', label: t('book.self') },
    ...(hasChildren ? [{ value: 'child' as const, label: t('book.child') }] : []),
    { value: 'other', label: t('book.other') },
  ];
  const ready = forWhom === 'self' || (forWhom === 'child' && childId !== undefined) || (forWhom === 'other' && patient.trim().length > 1);
  const slotGone = getApiErrorCode(create.error) === 'slot_unavailable';

  const submit = () => {
    create.mutate(
      {
        doctorId,
        mode: mode.data,
        startsAt: startsAtParam(date, time),
        forWhom,
        childId,
        patientName: patient.trim(),
        reason,
        note,
        share: CONSENT_SCOPES.filter((s) => share[s]),
      },
      {
        onSuccess: (r) => {
          if (r.payment && r.booking.status === 'held') {
            if (!goToGateway(r.payment.redirect_url)) router.replace(`/services/bookings/${r.booking.id}`);
          } else router.replace(`/services/bookings/${r.booking.id}?new=1`);
        },
      },
    );
  };

  return shell(
    <div className="dr-body">
      <Card as="section" className="dr-summary">
        <DoctorAvatar name={quote.doctor.name} photoUrl={quote.doctor.photo_url} size="md" />
        <div className="dr-card-main">
          <h2 className="dr-card-name">{title(quote.doctor)}</h2>
          <p className="dr-card-sub">{quote.doctor.specialty.title}</p>
        </div>
        <div className="dr-when-box">
          <ModeTag mode={mode.data} />
          <span>{whenLabel(date, time, locale)}</span>
        </div>
      </Card>

      <Card as="section" aria-labelledby={`${noteId}-who`} className="dr-form">
        <h2 id={`${noteId}-who`} className="dr-h2">{t('book.forWhom')}</h2>
        <div role="radiogroup" aria-labelledby={`${noteId}-who`} className="dr-radios" {...pilot('doctor-book.for', 'choice')}>
          {options.map((o) => (
            <button key={o.value} type="button" role="radio" aria-checked={forWhom === o.value} {...pilotOption(o.value)} className={`dr-radiorow${forWhom === o.value ? ' is-on' : ''}`} onClick={() => setForWhom(o.value)}>
              <span>{o.label}</span>
              <span className="dr-radio" aria-hidden />
            </button>
          ))}
        </div>
        {forWhom === 'child' ? (
          <div className="dr-chiprow is-wrap" role="group" aria-label={t('book.child')} {...pilot('doctor-book.child', 'choice')}>
            {quote.children.map((c) => (
              <PillChip key={c.id} pressed={childId === c.id} onPressedChange={() => setChildId(c.id)} {...pilotOption(String(c.id))}>
                {c.name}
              </PillChip>
            ))}
          </div>
        ) : null}
        {forWhom === 'other' ? (
          <input className="dr-input" value={patient} maxLength={80} onChange={(e) => setPatient(e.target.value)} placeholder={t('book.patientName')} aria-label={t('book.patientName')} {...pilot('doctor-book.patient', 'text')} />
        ) : null}
      </Card>

      <Card as="section" aria-labelledby={`${noteId}-why`} className="dr-form">
        <h2 id={`${noteId}-why`} className="dr-h2">{t('book.reason')}</h2>
        <div className="dr-chiprow is-wrap" role="group" aria-labelledby={`${noteId}-why`} {...pilot('doctor-book.reason', 'choice')}>
          {quote.reasons.map((r) => (
            <PillChip key={r.code} pressed={reason === r.code} onPressedChange={(on) => setReason(on ? r.code : undefined)} {...pilotOption(r.code)}>
              {r.title}
            </PillChip>
          ))}
        </div>
        <label htmlFor={noteId} className="dr-h3">{t('book.note')}</label>
        <textarea id={noteId} className="dr-input" rows={3} maxLength={1000} value={note} onChange={(e) => setNote(e.target.value)} placeholder={t('book.notePlaceholder')} {...pilot('doctor-book.note', 'text')} />
      </Card>

      {forWhom === 'self' ? (
        <Card as="section" aria-labelledby={`${noteId}-share`} className="dr-form">
          <h2 id={`${noteId}-share`} className="dr-h2">{t('book.share')}</h2>
          {CONSENT_SCOPES.map((s) => (
            <div key={s} className="dr-share-row">
              <span className="dr-share-icon"><Icon name={SCOPE_ICON[s]} size={18} /></span>
              <span className="dr-share-text">
                <strong id={`${noteId}-${s}`}>{t(`consent.${s}.title`)}</strong>
                <span>{t(`consent.${s}.hint`)}</span>
              </span>
              <Switch checked={share[s]} onCheckedChange={(v) => setShare((p) => ({ ...p, [s]: v }))} labelledBy={`${noteId}-${s}`} pilot={pilot(`doctor-book.share:${s}`)} />
            </div>
          ))}
          <p className="dr-fine">{t('book.shareNote')}</p>
        </Card>
      ) : null}

      <Card as="section" className="dr-pricebox" aria-label={t('book.price')}>
        <div className="dr-price-row">
          <span>{t('book.visitLine', { minutes: formatNumber(quote.visit_type.duration_minutes, locale) })}</span>
          <span>{formatToman(quote.price.price_rials, locale)}</span>
        </div>
        {quote.price.discount_rials > 0 ? (
          <div className="dr-price-row is-discount">
            <span>{t('book.discount')}</span>
            <span dir="ltr">{`-${formatToman(quote.price.discount_rials, locale)}`}</span>
          </div>
        ) : null}
        <div className="dr-price-row is-total">
          <span>{t('book.total')}</span>
          <span>{t('book.toman', { price: formatToman(quote.price.total_rials, locale) })}</span>
        </div>
      </Card>

      {create.isError ? (
        <p role="alert" className="dr-error">
          {errorText(create.error)}
          {slotGone ? (
            <button type="button" className="dr-link" onClick={back}>
              {t('book.pickAgain')}
            </button>
          ) : null}
        </p>
      ) : null}
      {!quote.payments_available && quote.price.total_rials > 0 ? <InfoNote>{t('errors.payment_unavailable')}</InfoNote> : null}
      <PrimaryButton
        icon="lock"
        loading={create.isPending}
        disabled={!ready || (!quote.payments_available && quote.price.total_rials > 0)}
        onClick={submit}
      >
        {quote.price.total_rials > 0 ? t('book.pay') : t('book.confirm')}
      </PrimaryButton>
      <p className="dr-fine is-center">{t('book.cancelNote', { hours: formatNumber(quote.cancel_window_hours, locale) })}</p>
    </div>,
  );
}
