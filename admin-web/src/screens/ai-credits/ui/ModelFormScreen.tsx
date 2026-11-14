'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { RequireSuper } from '@/features/auth';
import { fieldError } from '@/shared/api';
import { CheckboxGrid, confirm, FormPage, LoadGate, PageHeader, Select, Switch, TextInput, toast, useNotifyError } from '@/shared/ui';

import { CAPABILITIES, modelsApi, type AiModel } from '../api/ai-credits';
import { milliToUsd, usdToMilli } from '../lib/report';

/** /ai/models/new and /ai/models/:id (super admin). */
export function ModelFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('aiCredits.models');
  const detail = modelsApi.useDetail(id);
  const list = modelsApi.useList();
  return (
    <RequireSuper>
      <LoadGate
        queries={id === null ? [list] : [detail, list]}
        header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/ai/models" backLabel={t('backToList')} />}
      >
        {() => (
          <ModelForm
            id={id}
            row={detail.data?.model ?? null}
            slots={detail.data?.slots ?? []}
            providers={list.data?.providers ?? []}
            configured={list.data?.configured_providers ?? []}
          />
        )}
      </LoadGate>
    </RequireSuper>
  );
}

function ModelForm({
  id,
  row,
  slots,
  providers,
  configured,
}: {
  id: number | null;
  row: AiModel | null;
  slots: string[];
  providers: string[];
  configured: string[];
}) {
  const t = useTranslations('aiCredits.models');
  const tcap = useTranslations('aiCredits.capabilities');
  const tc = useTranslations('crud');
  const router = useRouter();
  const notifyError = useNotifyError();
  const save = modelsApi.useSave(id);

  const [code, setCode] = useState(row?.code ?? '');
  const [provider, setProvider] = useState(row?.provider ?? providers[0] ?? '');
  const [upstream, setUpstream] = useState(row?.upstream_model ?? '');
  const [title, setTitle] = useState(row?.title ?? '');
  const [caps, setCaps] = useState<string[]>(row?.capabilities ?? ['chat']);
  const [input, setInput] = useState(row ? String(milliToUsd(row.input_price_milli)) : '0');
  const [output, setOutput] = useState(row ? String(milliToUsd(row.output_price_milli)) : '0');
  const [audio, setAudio] = useState(row ? String(milliToUsd(row.audio_price_milli)) : '0');
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (name: string) => fieldError(save.error, name);
  const providerLabel = (p: string) => (t.has(`providers.${p}` as 'providers.arvan') ? t(`providers.${p}` as 'providers.arvan') : p);

  const submit = async () => {
    if (row && row.is_active && !active && slots.length > 0) {
      if (!(await confirm({ message: t('confirmDeactivate', { slots: slots.join('، ') }), tone: 'danger' }))) return;
    }
    save.mutate(
      {
        ...(id === null ? { code: code.trim() } : {}),
        provider,
        upstream_model: upstream.trim(),
        title: title.trim(),
        capabilities: caps,
        input_price_milli: usdToMilli(input) ?? -1,
        output_price_milli: usdToMilli(output) ?? -1,
        audio_price_milli: usdToMilli(audio) ?? -1,
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/ai/models');
        },
        onError: notifyError,
      },
    );
  };

  return (
    <FormPage
      title={id === null ? t('new') : row?.title || t('edit')}
      backHref="/ai/models"
      backLabel={t('backToList')}
      onSubmit={() => void submit()}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      {slots.length > 0 ? <p className="m-0 text-sm text-ink-3">{t('usedBy', { slots: slots.join('، ') })}</p> : null}
      <div className="form-grid">
        <TextInput
          label={t('code')}
          hint={id === null ? t('codeHint') : t('codeFixed')}
          value={code}
          onChange={(e) => setCode(e.target.value)}
          dir="ltr"
          maxLength={64}
          required
          disabled={id !== null}
          error={err('code')}
        />
        <TextInput label={t('titleField')} value={title} onChange={(e) => setTitle(e.target.value)} maxLength={128} required error={err('title')} />
        <Select
          label={t('provider')}
          hint={configured.includes(provider) ? t('providerConfigured') : t('providerNotConfigured')}
          value={provider}
          onChange={(e) => setProvider(e.target.value)}
          options={providers.map((p) => ({ value: p, label: providerLabel(p) }))}
          required
          error={err('provider')}
        />
        <TextInput
          label={t('upstream')}
          hint={t('upstreamHint')}
          value={upstream}
          onChange={(e) => setUpstream(e.target.value)}
          dir="ltr"
          maxLength={128}
          required
          error={err('upstream_model')}
        />
      </div>
      <CheckboxGrid
        label={t('capabilities')}
        hint={t('capabilitiesHint')}
        error={err('capabilities') ?? err('capabilities.0')}
        options={CAPABILITIES.map((c) => ({ value: c, label: tcap(c) }))}
        value={caps}
        onChange={setCaps}
      />
      <p className="m-0 text-sm text-ink-3">{t('pricesHint')}</p>
      <div className="form-grid">
        <TextInput label={t('inputPrice')} type="number" step="0.001" min={0} value={input} onChange={(e) => setInput(e.target.value)} dir="ltr" error={err('input_price_milli')} />
        <TextInput label={t('outputPrice')} type="number" step="0.001" min={0} value={output} onChange={(e) => setOutput(e.target.value)} dir="ltr" error={err('output_price_milli')} />
        <TextInput label={t('audioPrice')} type="number" step="0.001" min={0} value={audio} onChange={(e) => setAudio(e.target.value)} dir="ltr" error={err('audio_price_milli')} />
        <Switch label={tc('isActive')} hint={t('activeHint')} checked={active} onChange={setActive} />
      </div>
    </FormPage>
  );
}
