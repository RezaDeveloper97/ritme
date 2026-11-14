'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { RequireSuper } from '@/features/auth';
import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { formatNumber, rialsToToman, toIntOrNull, tomanToRials } from '@/shared/lib';
import {
  FormPage,
  LoadGate,
  PageHeader,
  Switch,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import { packsApi, type CreditPack } from '../api/ai-credits';

/** /ai/credit-packs/new and /ai/credit-packs/:id (super admin). Prices are entered in toman, sent as rials. */
export function PackFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('aiCredits.packs');
  const detail = packsApi.useDetail(id);
  const list = packsApi.useList();
  return (
    <RequireSuper>
      <LoadGate
        queries={id === null ? [list] : [detail, list]}
        header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/ai/credit-packs" backLabel={t('backToList')} />}
      >
        {() => <PackForm id={id} row={detail.data?.pack ?? null} nextSort={list.data?.next_sort_order ?? 1} />}
      </LoadGate>
    </RequireSuper>
  );
}

function PackForm({ id, row, nextSort }: { id: number | null; row: CreditPack | null; nextSort: number }) {
  const t = useTranslations('aiCredits.packs');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = packsApi.useSave(id);

  const [code, setCode] = useState(row?.code ?? '');
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [badge, setBadge] = useState<Translations>(row?.badge ?? {});
  const [credits, setCredits] = useState(row ? String(row.credits) : '');
  const [price, setPrice] = useState(row ? String(rialsToToman(row.price_rials)) : '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? nextSort));
  const [highlighted, setHighlighted] = useState(row?.is_highlighted ?? false);
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (name: string) => fieldError(save.error, name);

  const priceRials = tomanToRials(price);
  const creditsN = toIntOrNull(credits);
  const perCredit = priceRials && creditsN ? Math.floor(priceRials / creditsN) : null;

  const submit = () =>
    save.mutate(
      {
        ...(id === null ? { code: code.trim() } : {}),
        title,
        badge,
        credits: creditsN,
        price_rials: priceRials,
        sort_order: toIntOrNull(sortOrder),
        is_highlighted: highlighted,
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/ai/credit-packs');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.title) || t('edit')}
      backHref="/ai/credit-packs"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      {row && row.invoices_count > 0 ? <p className="m-0 text-sm text-ink-3">{t('changeNote')}</p> : null}
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
        <TextInput
          label={t('credits')}
          type="number"
          min={1}
          value={credits}
          onChange={(e) => setCredits(e.target.value)}
          required
          error={err('credits')}
        />
      </div>
      <TranslatableField name="title" label={t('titleField')} required value={title} onChange={setTitle} maxLength={64} errors={fieldErrorsOf(save.error)} />
      <TranslatableField name="badge" label={t('badgeField')} value={badge} onChange={setBadge} maxLength={32} errors={fieldErrorsOf(save.error)} />
      <div className="form-grid">
        <TextInput
          label={t('priceToman')}
          hint={perCredit ? t('perCreditHint', { amount: formatNumber(rialsToToman(perCredit), locale) }) : t('priceHint')}
          type="number"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
          required
          error={err('price_rials')}
        />
        <TextInput
          label={tc('sortOrder')}
          hint={tc('sortOrderHint')}
          type="number"
          value={sortOrder}
          onChange={(e) => setSortOrder(e.target.value)}
          error={err('sort_order')}
        />
        <div className="flex flex-col gap-3">
          <Switch label={t('highlightedField')} hint={t('highlightedHint')} checked={highlighted} onChange={setHighlighted} />
          <Switch label={tc('isActive')} hint={t('activeHint')} checked={active} onChange={setActive} />
        </div>
      </div>
    </FormPage>
  );
}
