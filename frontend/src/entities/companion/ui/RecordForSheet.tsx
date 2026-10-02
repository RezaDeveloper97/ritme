'use client';

import { useTranslations } from 'next-intl';
import { type KeyboardEvent, useRef } from 'react';

import { useDirection } from '@/shared/i18n';
import { AppSheet } from '@/shared/sheet';
import { Icon, InfoNote, PrimaryButton } from '@/shared/ui';

import type { RecordSection, RecordTarget } from '../lib/record-for';
import { PersonBubble } from './FamilyStrip';

/*
 * «برای چه کسی ثبت می‌کنی؟» (B-N4-06, nbl_/nbd_Hamdam_RecordFor): the viewer
 * («خودم», violet initial) and every owner who granted edit on this section
 * (rose initial), as radio cards; the info note; the CTA saves for the chosen
 * one («ثبت برای سارا»). Only rendered when `targets` is non-empty — the form
 * decides that with `showRecordForPicker`.
 */

/** null = the viewer's own record. */
type Target = number | null;

interface RecordForSheetProps {
  open: boolean;
  onClose: () => void;
  section: RecordSection;
  targets: readonly RecordTarget[];
  /** The viewer's own name, for the «خودم» initial. */
  selfName: string | null;
  value: Target;
  onChange: (value: Target) => void;
  /** Save for `value`. */
  onConfirm: () => void;
  pending?: boolean;
  error?: string | null;
}

export function RecordForSheet({
  open,
  onClose,
  section,
  targets,
  selfName,
  value,
  onChange,
  onConfirm,
  pending = false,
  error,
}: RecordForSheetProps) {
  const t = useTranslations('companions.recordFor');
  const rtl = useDirection() === 'rtl';
  const refs = useRef<Array<HTMLButtonElement | null>>([]);
  const options: Array<{ value: Target; name: string | null; title: string; desc: string }> = [
    { value: null, name: selfName, title: t('self'), desc: t(`selfDesc.${section}`) },
    ...targets.map((target) => {
      const name = target.name ?? t('someone');
      return { value: target.ownerId, name: target.name, title: name, desc: t(`ownerDesc.${section}`, { name }) };
    }),
  ];
  const checkedIndex = Math.max(0, options.findIndex((o) => o.value === value));
  const chosen = targets.find((target) => target.ownerId === value);

  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    const forward = rtl ? 'ArrowLeft' : 'ArrowRight';
    const backward = rtl ? 'ArrowRight' : 'ArrowLeft';
    let next: number | null = null;
    if (event.key === 'ArrowDown' || event.key === forward) next = (index + 1) % options.length;
    else if (event.key === 'ArrowUp' || event.key === backward) next = (index - 1 + options.length) % options.length;
    if (next === null) return;
    event.preventDefault();
    onChange(options[next].value);
    refs.current[next]?.focus();
  };

  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="half"
      title={t('title')}
      className="rcf-sheet"
      footer={
        <PrimaryButton loading={pending} onClick={onConfirm}>
          {chosen ? t('confirmFor', { name: chosen.name ?? t('someone') }) : t('confirmSelf')}
        </PrimaryButton>
      }
    >
      <div className="rcf-body">
        <div role="radiogroup" aria-label={t('title')} className="nb-rcards rcf-options">
          {options.map((option, index) => (
            <button
              key={option.value ?? 'self'}
              ref={(el) => {
                refs.current[index] = el;
              }}
              type="button"
              role="radio"
              aria-checked={option.value === value}
              tabIndex={index === checkedIndex ? 0 : -1}
              className="nb-rcard rcf-option"
              onClick={() => onChange(option.value)}
              onKeyDown={(event) => onKeyDown(event, index)}
            >
              <PersonBubble name={option.name} tone={option.value === null ? 'companion' : 'self'} />
              <span className="nb-rcard-text">
                <span className="nb-rcard-title">{option.title}</span>
                <span className="nb-rcard-desc">{option.desc}</span>
              </span>
              <span className="nb-rcard-dot" aria-hidden />
            </button>
          ))}
        </div>
        <InfoNote className="rcf-note">{t('note')}</InfoNote>
        {error ? (
          <p role="alert" className="rcf-error">
            <Icon name="info" size={16} />
            {error}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}

interface RecordForRowProps {
  section: RecordSection;
  /** The chosen owner, or null for the viewer. */
  target: RecordTarget | null;
  selfName: string | null;
  /** Opens the sheet; omitted = a fixed target (editing an owner's record). */
  onChange?: () => void;
}

/**
 * The form's summary of who the record is for, at the top of the form:
 * «ثبت برای: سارا · تغییر», or — on an owner's record — a fixed line
 * «داروی سارا · با اجازه ویرایش او».
 */
export function RecordForRow({ section, target, selfName, onChange }: RecordForRowProps) {
  const t = useTranslations('companions.recordFor');
  const name = target ? (target.name ?? t('someone')) : t('self');
  return (
    <section className="nb-card rcf-row" aria-live="polite">
      <PersonBubble name={target ? target.name : selfName} tone={target ? 'self' : 'companion'} />
      <span className="rcf-row-text">
        <span className="rcf-row-label">{onChange ? t('rowLabel') : t(`editingLabel.${section}`)}</span>
        <b className="rcf-row-name">{onChange ? name : t(`editingFor.${section}`, { name })}</b>
      </span>
      {onChange ? (
        <button type="button" className="rcf-change" onClick={onChange} aria-label={`${t('rowLabel')} ${name}: ${t('change')}`}>
          {t('change')}
        </button>
      ) : null}
    </section>
  );
}

interface RecordedForProps {
  section: RecordSection;
  name: string | null;
  /** «برگشت» — where the form came from. */
  onDone: () => void;
  doneLabel: string;
}

/** After a delegated save: «برای سارا ثبت شد» and the way back. */
export function RecordedFor({ section, name, onDone, doneLabel }: RecordedForProps) {
  const t = useTranslations('companions.recordFor');
  const who = name ?? t('someone');
  return (
    <section className="nb-card rcf-done" role="status">
      <span className="rcf-done-icon" aria-hidden>
        <Icon name="check" size={26} strokeWidth={2.2} />
      </span>
      <h2 className="rcf-done-title">{t('savedFor', { name: who })}</h2>
      <p className="rcf-done-body">{t(`savedBody.${section}`, { name: who })}</p>
      <PrimaryButton onClick={onDone}>{doneLabel}</PrimaryButton>
    </section>
  );
}
