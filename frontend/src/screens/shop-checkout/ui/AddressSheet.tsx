'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { type ShopAddress, useDeleteAddress, useSaveAddress } from '@/entities/shop';
import { AppSheet } from '@/shared/sheet';
import { Checkbox, PrimaryButton, SecondaryButton } from '@/shared/ui';
import '@/shared/ui/styles/shco.css';

interface Props {
  addresses: ShopAddress[];
  selectedId: number | null;
  onSelect: (id: number) => void;
  onClose: () => void;
}

type Mode = { kind: 'pick' } | { kind: 'form'; address: ShopAddress | null };

/** Pick, add, edit or delete a delivery address. Address and phone are PII: held in state only, never logged. */
export function AddressSheet({ addresses, selectedId, onSelect, onClose }: Props) {
  const t = useTranslations('shop.address');
  const [mode, setMode] = useState<Mode>(addresses.length ? { kind: 'pick' } : { kind: 'form', address: null });
  const save = useSaveAddress();
  const del = useDeleteAddress();
  const editing = mode.kind === 'form' ? mode.address : null;

  const [label, setLabel] = useState('');
  const [name, setName] = useState('');
  const [phone, setPhone] = useState('');
  const [province, setProvince] = useState('');
  const [city, setCity] = useState('');
  const [line, setLine] = useState('');
  const [postal, setPostal] = useState('');
  const [isDefault, setIsDefault] = useState(false);

  const openForm = (a: ShopAddress | null) => {
    setLabel(a?.label ?? '');
    setName(a?.recipient_name ?? '');
    setPhone(a?.recipient_phone ?? '');
    setProvince(a?.province ?? '');
    setCity(a?.city ?? '');
    setLine(a?.address_line ?? '');
    setPostal(a?.postal_code ?? '');
    setIsDefault(a?.is_default ?? addresses.length === 0);
    save.reset();
    setMode({ kind: 'form', address: a });
  };

  const canSave = name.trim() && phone.trim() && city.trim() && line.trim();
  const submit = () =>
    save.mutate(
      {
        id: editing?.id,
        input: {
          label: label.trim() || null,
          recipient_name: name.trim(),
          recipient_phone: phone.trim(),
          province: province.trim() || null,
          city: city.trim(),
          address_line: line.trim(),
          postal_code: postal.trim() || null,
          is_default: isDefault,
        },
      },
      {
        onSuccess: (a) => {
          onSelect(a.id);
          onClose();
        },
      },
    );

  if (mode.kind === 'pick') {
    return (
      <AppSheet open onClose={onClose} size="half" title={t('pick')}>
        <div className="shco-sheet">
          <ul className="shco-addr-list">
            {addresses.map((a) => (
              <li key={a.id} className={`shco-addr${a.id === selectedId ? ' is-on' : ''}`}>
                <button type="button" className="shco-addr-main" aria-pressed={a.id === selectedId} onClick={() => { onSelect(a.id); onClose(); }}>
                  <span className="shco-addr-label">{a.label || a.city}</span>
                  <span className="shco-addr-line">{a.address_line}</span>
                </button>
                <button type="button" className="shco-addr-edit" aria-label={t('edit')} onClick={() => openForm(a)}>
                  {t('edit')}
                </button>
              </li>
            ))}
          </ul>
          <SecondaryButton onClick={() => openForm(null)}>{t('add')}</SecondaryButton>
        </div>
      </AppSheet>
    );
  }

  return (
    <AppSheet
      open
      onClose={save.isPending ? () => undefined : onClose}
      size="full"
      title={editing ? t('editTitle') : t('newTitle')}
      footer={
        <div className="shco-sheet-actions">
          <PrimaryButton loading={save.isPending} disabled={!canSave} onClick={submit}>
            {t('save')}
          </PrimaryButton>
          {editing ? (
            <SecondaryButton
              variant="text"
              disabled={save.isPending || del.isPending}
              onClick={() => del.mutate(editing.id, { onSuccess: onClose })}
            >
              {t('delete')}
            </SecondaryButton>
          ) : (
            <SecondaryButton variant="text" onClick={addresses.length ? () => setMode({ kind: 'pick' }) : onClose}>
              {t('cancel')}
            </SecondaryButton>
          )}
        </div>
      }
    >
      <div className="shco-sheet">
        <Field id="shco-label" label={t('label')} value={label} onChange={setLabel} placeholder={t('labelPlaceholder')} max={40} />
        <Field id="shco-name" label={t('name')} value={name} onChange={setName} max={120} autoComplete="name" />
        <Field id="shco-phone" label={t('phone')} value={phone} onChange={setPhone} ltr inputMode="tel" autoComplete="tel" max={20} />
        <Field id="shco-province" label={t('province')} value={province} onChange={setProvince} max={60} />
        <Field id="shco-city" label={t('city')} value={city} onChange={setCity} max={60} />
        <Field id="shco-line" label={t('line')} value={line} onChange={setLine} max={500} autoComplete="street-address" />
        <Field id="shco-postal" label={t('postal')} value={postal} onChange={setPostal} ltr inputMode="numeric" max={12} />
        <Checkbox checked={isDefault} onCheckedChange={setIsDefault} label={t('default')} />
        {save.isError ? (
          <p className="shco-error" role="alert">
            {t('saveFailed')}
          </p>
        ) : null}
        <p className="shco-note">{t('note')}</p>
      </div>
    </AppSheet>
  );
}

function Field({
  id,
  label,
  value,
  onChange,
  placeholder,
  max,
  ltr,
  inputMode,
  autoComplete,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  max: number;
  ltr?: boolean;
  inputMode?: 'tel' | 'numeric';
  autoComplete?: string;
}) {
  return (
    <>
      <label className="lbl" htmlFor={id}>
        {label}
      </label>
      <div className="field">
        <input
          id={id}
          value={value}
          maxLength={max}
          dir={ltr ? 'ltr' : undefined}
          inputMode={inputMode}
          autoComplete={autoComplete}
          placeholder={placeholder}
          onChange={(e) => onChange(e.target.value)}
        />
      </div>
    </>
  );
}
