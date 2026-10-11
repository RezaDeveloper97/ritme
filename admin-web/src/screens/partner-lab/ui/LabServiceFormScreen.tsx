'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { toIntOrNull } from '@/shared/lib';
import { FormPage, LoadGate, PageHeader, Select, Switch, TextInput, toast, TranslatableField, useNotifyError, type Translations } from '@/shared/ui';

import { labServicesApi, type LabServiceRow } from '../api/partnerLab';

/** /lab-services/new and /lab-services/:id. */
export function LabServiceFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('partnerLab.services');
  const detail = labServicesApi.useDetail(id);
  const options = labServicesApi.useList();
  return (
    <LoadGate
      queries={id === null ? [options] : [detail, options]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/lab-services" backLabel={t('back')} />}
    >
      {() => <ServiceForm id={id} row={detail.data ?? null} partnerCodes={options.data?.partners.codes ?? ['fake']} kinds={options.data?.kinds ?? ['home_draw', 'checkup']} />}
    </LoadGate>
  );
}

function ServiceForm({ id, row, partnerCodes, kinds }: { id: number | null; row: LabServiceRow | null; partnerCodes: string[]; kinds: string[] }) {
  const t = useTranslations('partnerLab.services');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = labServicesApi.useSave(id);
  const [code, setCode] = useState(row?.code ?? '');
  const [kind, setKind] = useState(row?.kind ?? kinds[0] ?? 'home_draw');
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [description, setDescription] = useState<Translations>(row?.description ?? {});
  const [partnerName, setPartnerName] = useState(row?.partner_name ?? '');
  const [partnerCode, setPartnerCode] = useState(row?.partner_code ?? 'fake');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? 0));
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (name: string) => fieldError(save.error, name);

  const submit = () =>
    save.mutate(
      { code: code.trim(), kind, title, description, partner_name: partnerName.trim(), partner_code: partnerCode, sort_order: toIntOrNull(sortOrder) ?? 0, is_active: active },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/lab-services');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.title) || t('edit')}
      backHref="/lab-services"
      backLabel={t('back')}
      onSubmit={submit}
      submitLabel={id === null ? t('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <TranslatableField name="title" label={t('titleField')} value={title} onChange={setTitle} maxLength={120} errors={fieldErrorsOf(save.error)} />
      <TranslatableField name="description" label={t('description')} value={description} onChange={setDescription} maxLength={500} errors={fieldErrorsOf(save.error)} />
      <div className="form-grid">
        <TextInput label={t('code')} hint={t('codeHint')} value={code} onChange={(e) => setCode(e.target.value)} dir="ltr" maxLength={40} required error={err('code')} />
        <Select label={t('kind')} value={kind} onChange={(e) => setKind(e.target.value)} options={kinds.map((k) => ({ value: k, label: t(`kinds.${k}` as 'kinds.home_draw') }))} error={err('kind')} />
        <TextInput label={t('partnerName')} hint={t('partnerNameHint')} value={partnerName} onChange={(e) => setPartnerName(e.target.value)} maxLength={120} required error={err('partner_name')} />
        <Select label={t('partnerCode')} hint={t('partnerCodeHint')} value={partnerCode} onChange={(e) => setPartnerCode(e.target.value)} options={partnerCodes.map((c) => ({ value: c, label: c }))} error={err('partner_code')} />
        <TextInput label={tc('sortOrder')} hint={tc('sortOrderHint')} type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} error={err('sort_order')} />
        <Switch label={tc('isActive')} checked={active} onChange={setActive} />
      </div>
    </FormPage>
  );
}
