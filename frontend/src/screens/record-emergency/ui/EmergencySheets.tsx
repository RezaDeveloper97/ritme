'use client';

import { useTranslations } from 'next-intl';
import { type FormEvent, type ReactNode, useEffect, useId, useState } from 'react';

import { type OwnerEmergencyCard, useSaveEmergencyCard } from '@/entities/health-record';
import { AppSheet } from '@/shared/sheet';
import { PrimaryButton, SecondaryButton } from '@/shared/ui';
import { pilot } from '@/shared/pilot';

import { isValidLast4, isValidPhone, normalizeLast4, normalizePhone } from '../model/forms';
import '@/shared/ui/styles/rem.css';
import '@/shared/ui/styles/hrec.css';

interface SheetProps {
  open: boolean;
  onClose: () => void;
  card: OwnerEmergencyCard;
}

/** «تماس اضطراری»: name, relation, phone (4–15 digits). «حذف» clears it (PUT `emergency_contact: null`). */
export function ContactSheet({ open, onClose, card }: SheetProps) {
  const t = useTranslations('record.emergency');
  const save = useSaveEmergencyCard();
  const formId = useId();
  const [name, setName] = useState('');
  const [relation, setRelation] = useState('');
  const [phone, setPhone] = useState('');
  const [touched, setTouched] = useState(false);

  useEffect(() => {
    if (!open) return;
    setName(card.contact?.name ?? '');
    setRelation(card.contact?.relation ?? '');
    setPhone(card.contact?.phone ?? '');
    setTouched(false);
    save.reset();
    // only when the sheet opens
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const phoneOk = isValidPhone(phone);
  const valid = name.trim().length > 0 && phoneOk;
  const submit = (e: FormEvent) => {
    e.preventDefault();
    setTouched(true);
    if (!valid) return;
    save.mutate(
      { emergency_contact: { name: name.trim(), relation: relation.trim() || null, phone: normalizePhone(phone) } },
      { onSuccess: onClose },
    );
  };

  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="half"
      title={t('contactTitle')}
      footer={
        <div className="rem-sheet-btns">
          <PrimaryButton type="submit" form={formId} loading={save.isPending} {...pilot('recem.contact-save')}>
            {t('save')}
          </PrimaryButton>
          {card.contact ? (
            <SecondaryButton
              variant="text"
              disabled={save.isPending}
              onClick={() => save.mutate({ emergency_contact: null }, { onSuccess: onClose })}
            >
              {t('remove')}
            </SecondaryButton>
          ) : null}
        </div>
      }
    >
      <form id={formId} className="rem-form" onSubmit={submit} noValidate>
        <Field label={t('contactName')}>
          {(id) => (
            <input id={id} className="hrec-input" value={name} maxLength={60} autoComplete="off" onChange={(e) => setName(e.target.value)} {...pilot('recem.contact-name', 'text')} />
          )}
        </Field>
        <Field label={t('contactRelation')}>
          {(id) => (
            <input id={id} className="hrec-input" value={relation} maxLength={30} autoComplete="off" onChange={(e) => setRelation(e.target.value)} {...pilot('recem.contact-relation', 'text')} />
          )}
        </Field>
        <Field label={t('contactPhone')} error={touched && !phoneOk ? t('phoneError') : null}>
          {(id, describedBy) => (
            <input
              id={id}
              className="hrec-input"
              dir="ltr"
              inputMode="tel"
              autoComplete="off"
              maxLength={20}
              value={phone}
              aria-invalid={touched && !phoneOk}
              aria-describedby={describedBy}
              onChange={(e) => setPhone(e.target.value)}
              {...pilot('recem.contact-phone', 'text')}
            />
          )}
        </Field>
        {save.isError ? (
          <p className="rem-error" role="alert">
            {t('saveError')}
          </p>
        ) : null}
      </form>
    </AppSheet>
  );
}

/** «بیمه»: an optional label + exactly the last 4 digits (the server stores nothing more). */
export function InsuranceSheet({ open, onClose, card }: SheetProps) {
  const t = useTranslations('record.emergency');
  const save = useSaveEmergencyCard();
  const formId = useId();
  const [label, setLabel] = useState('');
  const [last4, setLast4] = useState('');
  const [touched, setTouched] = useState(false);

  useEffect(() => {
    if (!open) return;
    setLabel(card.insurance?.label ?? '');
    setLast4(card.insuranceLast4 ?? '');
    setTouched(false);
    save.reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const ok = isValidLast4(last4);
  const submit = (e: FormEvent) => {
    e.preventDefault();
    setTouched(true);
    if (!ok) return;
    save.mutate({ insurance: { label: label.trim() || null, last4: normalizeLast4(last4) } }, { onSuccess: onClose });
  };

  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="half"
      title={t('insuranceTitle')}
      footer={
        <div className="rem-sheet-btns">
          <PrimaryButton type="submit" form={formId} loading={save.isPending} {...pilot('recem.insurance-save')}>
            {t('save')}
          </PrimaryButton>
          {card.insurance ? (
            <SecondaryButton variant="text" disabled={save.isPending} onClick={() => save.mutate({ insurance: null }, { onSuccess: onClose })}>
              {t('remove')}
            </SecondaryButton>
          ) : null}
        </div>
      }
    >
      <form id={formId} className="rem-form" onSubmit={submit} noValidate>
        <Field label={t('insuranceLabel')}>
          {(id) => (
            <input id={id} className="hrec-input" value={label} maxLength={60} autoComplete="off" onChange={(e) => setLabel(e.target.value)} {...pilot('recem.insurance-label', 'text')} />
          )}
        </Field>
        <Field label={t('insuranceLast4')} hint={t('insuranceHint')} error={touched && !ok ? t('last4Error') : null}>
          {(id, describedBy) => (
            <input
              id={id}
              className="hrec-input"
              dir="ltr"
              inputMode="numeric"
              autoComplete="off"
              maxLength={4}
              value={last4}
              aria-invalid={touched && !ok}
              aria-describedby={describedBy}
              onChange={(e) => setLast4(e.target.value)}
              {...pilot('recem.insurance-last4', 'text')}
            />
          )}
        </Field>
        {save.isError ? (
          <p className="rem-error" role="alert">
            {t('saveError')}
          </p>
        ) : null}
      </form>
    </AppSheet>
  );
}

function Field({
  label,
  hint,
  error,
  children,
}: {
  label: string;
  hint?: string;
  error?: string | null;
  children: (id: string, describedBy: string | undefined) => ReactNode;
}) {
  const id = useId();
  const noteId = `${id}-note`;
  const note = error ?? hint;
  return (
    <div className="rem-field">
      <label htmlFor={id} className="hrec-label">
        {label}
      </label>
      {children(id, note ? noteId : undefined)}
      {note ? (
        <p id={noteId} className={error ? 'rem-error' : 'rem-hint'} role={error ? 'alert' : undefined}>
          {note}
        </p>
      ) : null}
    </div>
  );
}
