'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  JOIN_CATEGORY_ORDER,
  type JoinApplication,
  useCityServiceOptions,
  useDeleteJoinDraft,
  useSaveJoinStep,
} from '@/entities/city-service';
import { Checkbox, ChipGroup, Icon, InfoNote, PillChip, PrimaryButton, SecondaryButton } from '@/shared/ui';
import { useRouter } from '@/shared/i18n';

import { step1Body, step1From, toggleItem } from '../model/join';
import { useAutosave } from '../model/use-autosave';
import { SaveHint, useErrorText } from './parts';

/** Step 1 «معرفی» (nbl_Dir_Join): what the venue gets, the cooperation-terms placeholder and the basic details. */
export function StepIntro({ app, onNext, onSaveError }: { app: JoinApplication; onNext: () => void; onSaveError: (e: unknown) => void }) {
  const t = useTranslations('cityJoin');
  const tc = useTranslations('cityJoin.categories');
  const errorText = useErrorText();
  const router = useRouter();
  const options = useCityServiceOptions();
  const save = useSaveJoinStep(app.id);
  const del = useDeleteJoinDraft();
  const [form, setForm] = useState(() => step1From(app));
  const [askDelete, setAskDelete] = useState(false);
  const { state, flush } = useAutosave(form, (v) => save.mutateAsync({ step: 1, body: step1Body(v) }).catch((e) => {
    onSaveError(e);
    throw e;
  }));
  const set = <K extends keyof typeof form>(key: K, value: (typeof form)[K]) => setForm((f) => ({ ...f, [key]: value }));
  const cities = options.data?.cities ?? [];

  const next = async () => {
    if (await flush()) onNext();
  };

  return (
    <div className="cjn-stack">
      <section className="cjn-hero">
        <h2 className="cjn-hero-title">{t('intro.heroTitle')}</h2>
        <p className="cjn-hero-body">{t('intro.heroBody')}</p>
        <ul className="cjn-benefits">
          {(['map', 'book'] as const).map((k) => (
            <li key={k} className="cjn-benefit">
              <span className="cjn-benefit-ic">
                <Icon name={k === 'map' ? 'mapPin' : 'calendar'} size={18} />
              </span>
              <span>
                <strong className="cjn-benefit-t">{t(`intro.${k}Title`)}</strong>
                <span className="cjn-benefit-s">{t(`intro.${k}Body`)}</span>
              </span>
            </li>
          ))}
        </ul>
      </section>

      <InfoNote icon="info">
        <strong>{t('intro.termsTitle')}</strong> {t('intro.termsBody')}
      </InfoNote>

      <section className="card fld-card cjn-fields">
        <label className="fld-label">
          <span className="fld-label-t">{t('intro.name')}</span>
          <span className="field">
            <input type="text" maxLength={160} value={form.businessName} placeholder={t('intro.namePh')} onChange={(e) => set('businessName', e.target.value)} />
          </span>
        </label>

        <div className="cjn-group">
          <span className="fld-label-t">
            {t('intro.kind')} <span className="cjn-hint">· {t('intro.kindHint')}</span>
          </span>
          <ChipGroup label={t('intro.kind')}>
            {JOIN_CATEGORY_ORDER.map((slug) => (
              <PillChip
                key={slug}
                mode="multi"
                pressed={form.categories.includes(slug)}
                onPressedChange={() => set('categories', toggleItem(form.categories, slug))}
              >
                {tc(slug)}
              </PillChip>
            ))}
          </ChipGroup>
        </div>

        <div className="cjn-group">
          <span className="fld-label-t">{t('intro.city')}</span>
          {cities.length === 0 && !options.isPending ? (
            <p className="cjn-hint">{t('intro.cityNone')}</p>
          ) : (
            <ChipGroup label={t('intro.city')}>
              {cities.map((c) => (
                <PillChip key={c.id} mode="single" pressed={form.cityId === c.id} onPressedChange={() => set('cityId', c.id)}>
                  {c.name}
                </PillChip>
              ))}
            </ChipGroup>
          )}
        </div>

        <label className="fld-label">
          <span className="fld-label-t">{t('intro.district')}</span>
          <span className="field">
            <input type="text" maxLength={120} value={form.district} placeholder={t('intro.districtPh')} onChange={(e) => set('district', e.target.value)} />
          </span>
        </label>
        <label className="fld-label">
          <span className="fld-label-t">{t('intro.contact')}</span>
          <span className="field">
            <Icon name="user" size={18} />
            <input type="text" autoComplete="name" maxLength={120} value={form.contactName} onChange={(e) => set('contactName', e.target.value)} />
          </span>
        </label>
        <label className="fld-label">
          <span className="fld-label-t">{t('intro.phone')}</span>
          <span className="field">
            <Icon name="phone" size={18} />
            <input
              type="tel"
              inputMode="tel"
              autoComplete="tel"
              dir="ltr"
              maxLength={32}
              value={form.contactPhone}
              placeholder={t('intro.phonePh')}
              onChange={(e) => set('contactPhone', e.target.value)}
            />
          </span>
        </label>
        <Checkbox checked={form.ownerConfirmed} onCheckedChange={(v) => set('ownerConfirmed', v)} label={t('intro.owner')} />
      </section>

      <SaveHint state={state} />
      {save.isError ? <p className="cjn-err" role="alert">{errorText(save.error)}</p> : null}
      <PrimaryButton onClick={() => void next()}>
        {t('next')}
      </PrimaryButton>

      {askDelete ? (
        <div className="cjn-confirm" role="alertdialog" aria-label={t('intro.delete')}>
          <p>{t('intro.deleteAsk')}</p>
          <div className="cjn-row">
            <SecondaryButton block={false} onClick={() => setAskDelete(false)}>
              {t('intro.deleteNo')}
            </SecondaryButton>
            <PrimaryButton
              block={false}
              loading={del.isPending}
              onClick={() => del.mutate(app.id, { onSuccess: () => router.replace('/city-services') })}
            >
              {t('intro.deleteYes')}
            </PrimaryButton>
          </div>
        </div>
      ) : (
        <button type="button" className="cjn-link is-danger" onClick={() => setAskDelete(true)}>
          {t('intro.delete')}
        </button>
      )}
    </div>
  );
}
