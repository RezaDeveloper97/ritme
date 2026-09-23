'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { RequireSuper } from '@/features/auth';
import { fieldError } from '@/shared/api';
import { formatNumber, toIntOrNull } from '@/shared/lib';
import {
  Badge,
  Button,
  confirm,
  FormPage,
  LoadGate,
  PageHeader,
  Panel,
  Select,
  Switch,
  TextInput,
  toast,
  useNotifyError,
} from '@/shared/ui';

import { languagesApi, provisionSchema, type Language, type LanguageOptions } from '../api/languages';
import { useDirectionLabel } from './direction';

/** /languages/new and /languages/:id (+ regenerate) — Blade languages.form. */
export function LanguageFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('languages');
  const detail = languagesApi.useDetail(id);
  const options = languagesApi.useOptions();
  return (
    <RequireSuper>
      <LoadGate
        queries={[detail, options]}
        header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/languages" backLabel={t('backToList')} />}
      >
        {() => (options.data ? <LanguageForm id={id} row={detail.data?.language ?? null} options={options.data} /> : null)}
      </LoadGate>
    </RequireSuper>
  );
}

function useProvisionedToast() {
  const t = useTranslations('languages');
  const locale = useLocale();
  return (data: unknown, key: 'createdToast' | 'regeneratedToast') => {
    const p = provisionSchema.safeParse(data).data?.provisioned;
    toast.success(
      t(key, {
        messages: formatNumber(p?.messages ?? 0, locale),
        smart: formatNumber(p?.smart_messages ?? 0, locale),
      }),
    );
  };
}

function LanguageForm({ id, row, options }: { id: number | null; row: Language | null; options: LanguageOptions }) {
  const t = useTranslations('languages');
  const tc = useTranslations('crud');
  const router = useRouter();
  const direction = useDirectionLabel();
  const notifyError = useNotifyError();
  const provisioned = useProvisionedToast();
  const save = languagesApi.useSave(id);

  const [code, setCode] = useState(row?.code ?? '');
  const [dir, setDir] = useState(row?.direction ?? 'ltr');
  const [name, setName] = useState(row?.name ?? '');
  const [englishName, setEnglishName] = useState(row?.english_name ?? '');
  const [copyFrom, setCopyFrom] = useState(options.default_code);
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? options.next_sort_order));
  const [active, setActive] = useState(row?.is_active ?? true);
  const [isDefault, setIsDefault] = useState(row?.is_default ?? false);
  const err = (field: string) => fieldError(save.error, field);

  const submit = () =>
    save.mutate(
      {
        code: code.trim(),
        direction: dir,
        name: name.trim(),
        english_name: englishName.trim(),
        sort_order: toIntOrNull(sortOrder) ?? 0,
        is_active: active || isDefault,
        is_default: isDefault,
        ...(id === null ? { copy_from: copyFrom } : {}),
      },
      {
        onSuccess: (data) => {
          if (id === null) provisioned(data, 'createdToast');
          else toast.success(tc('saved'));
          router.push('/languages');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : row?.name ?? t('edit')}
      backHref="/languages"
      backLabel={t('backToList')}
      meta={row?.is_default ? <Badge tone="brand">{t('default')}</Badge> : null}
      onSubmit={submit}
      submitLabel={id === null ? t('create') : tc('saveChanges')}
      saving={save.isPending}
      after={row ? <RegeneratePanel language={row} options={options} /> : null}
    >
      <div className="form-grid">
        <TextInput
          label={t('code')}
          hint={id === null ? t('codeHint') : t('codeLocked')}
          value={code}
          onChange={(e) => setCode(e.target.value)}
          required
          readOnly={id !== null}
          maxLength={12}
          placeholder="ar"
          dir="ltr"
          error={err('code')}
        />
        <Select
          label={t('direction')}
          hint={t('directionHint')}
          value={dir}
          onChange={(e) => setDir(e.target.value)}
          options={options.directions.map((d) => ({ value: d, label: direction(d) }))}
          required
          error={err('direction')}
        />
        <TextInput
          label={t('nameNative')}
          hint={t('nameNativeHint')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          maxLength={60}
          placeholder="العربية"
          dir={dir === 'rtl' ? 'rtl' : 'ltr'}
          error={err('name')}
        />
        <TextInput
          label={t('englishName')}
          value={englishName}
          onChange={(e) => setEnglishName(e.target.value)}
          required
          maxLength={60}
          placeholder="Arabic"
          dir="ltr"
          error={err('english_name')}
        />
        {id === null ? (
          <Select
            className="span-2"
            label={t('copyFrom')}
            hint={t('copyFromHint')}
            value={copyFrom}
            onChange={(e) => setCopyFrom(e.target.value)}
            options={options.sources.map((s) => ({ value: s.code, label: `${s.name} (${s.code})` }))}
            error={err('copy_from')}
          />
        ) : null}
        <TextInput
          label={tc('sortOrder')}
          hint={tc('sortOrderHint')}
          type="number"
          value={sortOrder}
          onChange={(e) => setSortOrder(e.target.value)}
          error={err('sort_order')}
        />
        <div className="flex flex-col gap-3">
          <Switch label={tc('isActive')} checked={active || isDefault} disabled={isDefault} onChange={setActive} />
          <Switch label={t('isDefault')} hint={t('isDefaultHint')} checked={isDefault} onChange={setIsDefault} />
        </div>
      </div>
    </FormPage>
  );
}

/** Rebuild a language's UI bundles from another language (overwrites its translations). */
function RegeneratePanel({ language, options }: { language: Language; options: LanguageOptions }) {
  const t = useTranslations('languages');
  const notifyError = useNotifyError();
  const provisioned = useProvisionedToast();
  const regenerate = languagesApi.useAction('regenerate');
  const sources = options.sources.filter((s) => s.code !== language.code);
  const [source, setSource] = useState(
    sources.some((s) => s.code === options.default_code) ? options.default_code : (sources[0]?.code ?? ''),
  );

  const run = async () => {
    if (!(await confirm({ message: t('regenerateConfirm'), confirmLabel: t('regenerate'), tone: 'danger' }))) return;
    regenerate.mutate(
      { id: language.id, body: source ? { copy_from: source } : {} },
      { onSuccess: (data) => provisioned(data, 'regeneratedToast'), onError: notifyError },
    );
  };

  return (
    <Panel title={t('regenerateTitle')}>
      <div className="flex flex-col gap-3">
        <p className="field-hint m-0">{t('regenerateHint')}</p>
        <div className="flex flex-wrap items-end gap-3">
          <Select
            label={t('copyFrom')}
            className="w-64"
            value={source}
            onChange={(e) => setSource(e.target.value)}
            options={sources.map((s) => ({ value: s.code, label: `${s.name} (${s.code})` }))}
          />
          <Button onClick={run} loading={regenerate.isPending} disabled={!source}>
            {t('regenerate')}
          </Button>
        </div>
      </div>
    </Panel>
  );
}
