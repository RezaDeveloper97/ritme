'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { RequireSuper } from '@/features/auth';
import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { formatNumber, toIntOrNull } from '@/shared/lib';
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

import { plansApi, type Plan } from '../api/billing';
import { rialsToToman, tomanToRials } from '../lib/money';

/** /plus/plans/new and /plus/plans/:id (super admin). */
export function PlanFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('plus.plans');
  const detail = plansApi.useDetail(id);
  const list = plansApi.useList();
  return (
    <RequireSuper>
      <LoadGate
        queries={id === null ? [list] : [detail, list]}
        header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/plus/plans" backLabel={t('backToList')} />}
      >
        {() => <PlanForm id={id} row={detail.data?.plan ?? null} nextSort={list.data?.next_sort_order ?? 1} />}
      </LoadGate>
    </RequireSuper>
  );
}

function PlanForm({ id, row, nextSort }: { id: number | null; row: Plan | null; nextSort: number }) {
  const t = useTranslations('plus.plans');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = plansApi.useSave(id);

  const [code, setCode] = useState(row?.code ?? '');
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [badge, setBadge] = useState<Translations>(row?.badge ?? {});
  const [months, setMonths] = useState(String(row?.duration_months ?? 1));
  const [price, setPrice] = useState(row ? String(rialsToToman(row.price_rials)) : '');
  const [monthly, setMonthly] = useState(row?.monthly_display_rials ? String(rialsToToman(row.monthly_display_rials)) : '');
  const [sortOrder, setSortOrder] = useState(String(row?.sort_order ?? nextSort));
  const [highlighted, setHighlighted] = useState(row?.is_highlighted ?? false);
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (name: string) => fieldError(save.error, name);

  const priceRials = tomanToRials(price);
  const monthsN = toIntOrNull(months);
  const perMonth = priceRials && monthsN ? Math.floor(priceRials / monthsN) : null;

  const submit = () =>
    save.mutate(
      {
        ...(id === null ? { code: code.trim() } : {}),
        title,
        badge,
        duration_months: monthsN,
        price_rials: priceRials,
        monthly_display_rials: tomanToRials(monthly),
        sort_order: toIntOrNull(sortOrder),
        is_highlighted: highlighted,
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/plus/plans');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.title) || t('edit')}
      backHref="/plus/plans"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      {row && row.invoices_count > 0 ? <p className="m-0 text-sm text-ink-3">{t('priceChangeNote')}</p> : null}
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
          label={t('duration')}
          type="number"
          value={months}
          onChange={(e) => setMonths(e.target.value)}
          required
          error={err('duration_months')}
        />
      </div>
      <TranslatableField name="title" label={t('titleField')} required value={title} onChange={setTitle} maxLength={64} errors={fieldErrorsOf(save.error)} />
      <TranslatableField name="badge" label={t('badgeField')} value={badge} onChange={setBadge} maxLength={32} errors={fieldErrorsOf(save.error)} />
      <div className="form-grid">
        <TextInput
          label={t('priceToman')}
          hint={perMonth ? t('perMonthHint', { amount: formatNumber(rialsToToman(perMonth), locale) }) : t('priceHint')}
          type="number"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
          required
          error={err('price_rials')}
        />
        <TextInput
          label={t('monthlyToman')}
          hint={t('monthlyHint')}
          type="number"
          value={monthly}
          onChange={(e) => setMonthly(e.target.value)}
          error={err('monthly_display_rials')}
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
