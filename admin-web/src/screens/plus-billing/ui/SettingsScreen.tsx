'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState, type FormEvent } from 'react';

import { fieldError } from '@/shared/api';
import { formatNumber, toIntOrNull } from '@/shared/lib';
import { Badge, Button, LoadGate, PageHeader, Panel, TextInput, toast, useNotifyError } from '@/shared/ui';

import { useSaveSettings, useSettings, type PlusSettings } from '../api/billing';
import { bpsToPercentText, percentToBps } from '../lib/money';
import { useCanManage } from './parts';

/** /plus/settings — trial-offer percent and the VAT override (B-N2-09). */
export function SettingsScreen() {
  const t = useTranslations('plus.settings');
  const query = useSettings();
  return (
    <LoadGate queries={[query]} header={<PageHeader title={t('title')} />}>
      {() => (query.data ? <SettingsForm settings={query.data.settings} /> : null)}
    </LoadGate>
  );
}

function SettingsForm({ settings }: { settings: PlusSettings }) {
  const t = useTranslations('plus.settings');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const canManage = useCanManage();
  const notifyError = useNotifyError();
  const save = useSaveSettings();
  const [percent, setPercent] = useState(String(settings.trial_offer_percent));
  const [vat, setVat] = useState(settings.vat_override_bps === null ? '' : bpsToPercentText(settings.vat_override_bps));
  const err = (name: string) => fieldError(save.error, name);
  const pct = (bps: number) => formatNumber(bps / 100, locale);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!canManage || save.isPending) return;
    save.mutate(
      { trial_offer_percent: toIntOrNull(percent) ?? -1, vat_rate_bps: vat.trim() === '' ? null : (percentToBps(vat) ?? -1) },
      { onSuccess: () => toast.success(tc('saved')), onError: notifyError },
    );
  };

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title={t('title')}
        meta={
          <>
            <span>{t('vatNow', { rate: pct(settings.vat_rate_bps) })}</span>
            <Badge tone={settings.vat_source === 'admin' ? 'brand' : 'neutral'}>{t(`source.${settings.vat_source}`)}</Badge>
          </>
        }
      />
      {!canManage ? <p className="m-0 text-sm text-ink-3">{t('readOnly')}</p> : null}
      <form onSubmit={submit}>
        <Panel bodyClassName="">
          <div className="flex flex-col gap-5 p-4 sm:p-5">
            <div className="form-grid">
              <TextInput
                label={t('trialOffer')}
                hint={t('trialOfferHint', { days: formatNumber(settings.trial_days, locale) })}
                type="number"
                value={percent}
                onChange={(e) => setPercent(e.target.value)}
                readOnly={!canManage}
                required
                error={err('trial_offer_percent')}
              />
              <TextInput
                label={t('vat')}
                hint={t('vatHint', { rate: pct(settings.vat_env_rate_bps) })}
                type="number"
                step="0.01"
                value={vat}
                placeholder={bpsToPercentText(settings.vat_env_rate_bps)}
                onChange={(e) => setVat(e.target.value)}
                readOnly={!canManage}
                error={err('vat_rate_bps')}
              />
            </div>
            <p className="m-0 text-sm text-ink-3">{t('snapshotNote')}</p>
          </div>
          {canManage ? (
            <div className="form-actions">
              <Button type="submit" variant="primary" loading={save.isPending}>
                {tc('saveChanges')}
              </Button>
            </div>
          ) : null}
        </Panel>
      </form>
    </div>
  );
}
