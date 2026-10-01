'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { RequireSuper } from '@/features/auth';
import { fieldError } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { isoToLocalInput, localInputToApi, toIntOrNull } from '@/shared/lib';
import { CheckboxGrid, FormPage, LoadGate, PageHeader, Select, Switch, TextInput, toast, useNotifyError } from '@/shared/ui';

import { discountsApi, plansApi, type Discount, type Plan } from '../api/billing';
import { rialsToToman, tomanToRials } from '../lib/money';

/** /plus/discount-codes/new and /plus/discount-codes/:id (super admin). */
export function DiscountFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('plus.discounts');
  const detail = discountsApi.useDetail(id);
  const plans = plansApi.useList();
  return (
    <RequireSuper>
      <LoadGate
        queries={id === null ? [plans] : [detail, plans]}
        header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/plus/discount-codes" backLabel={t('backToList')} />}
      >
        {() => <DiscountForm id={id} row={detail.data?.discount_code ?? null} plans={plans.data?.items ?? []} />}
      </LoadGate>
    </RequireSuper>
  );
}

function DiscountForm({ id, row, plans }: { id: number | null; row: Discount | null; plans: Plan[] }) {
  const t = useTranslations('plus.discounts');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const notifyError = useNotifyError();
  const save = discountsApi.useSave(id);

  const [code, setCode] = useState(row?.code ?? '');
  const [kind, setKind] = useState<'percent' | 'amount'>(row?.kind ?? 'percent');
  const [value, setValue] = useState(row ? String(row.kind === 'amount' ? rialsToToman(row.value) : row.value) : '');
  const [maxUses, setMaxUses] = useState(row?.max_redemptions ? String(row.max_redemptions) : '');
  const [perUser, setPerUser] = useState(row ? (row.per_user_limit ? String(row.per_user_limit) : '') : '1');
  const [planIds, setPlanIds] = useState<string[]>((row?.plan_ids ?? []).map(String));
  const [startsAt, setStartsAt] = useState(isoToLocalInput(row?.starts_at));
  const [expiresAt, setExpiresAt] = useState(isoToLocalInput(row?.expires_at));
  const [active, setActive] = useState(row?.is_active ?? true);
  const err = (name: string) => fieldError(save.error, name);

  const submit = () =>
    save.mutate(
      {
        ...(id === null ? { code: code.trim() } : {}),
        kind,
        value: kind === 'amount' ? tomanToRials(value) : toIntOrNull(value),
        max_redemptions: toIntOrNull(maxUses),
        per_user_limit: toIntOrNull(perUser),
        plan_ids: planIds.map(Number),
        starts_at: localInputToApi(startsAt),
        expires_at: localInputToApi(expiresAt),
        is_active: active,
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/plus/discount-codes');
        },
        onError: notifyError,
      },
    );

  return (
    <FormPage
      title={id === null ? t('new') : row?.code ?? t('edit')}
      backHref="/plus/discount-codes"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <div className="form-grid">
        <TextInput
          label={t('code')}
          hint={id === null ? t('codeHint') : t('codeFixed')}
          value={code}
          onChange={(e) => setCode(e.target.value.toUpperCase())}
          dir="ltr"
          maxLength={64}
          required
          disabled={id !== null}
          error={err('code')}
        />
        <Select
          label={t('kind')}
          value={kind}
          onChange={(e) => setKind(e.target.value === 'amount' ? 'amount' : 'percent')}
          options={[
            { value: 'percent', label: t('kindPercent') },
            { value: 'amount', label: t('kindAmount') },
          ]}
          error={err('kind')}
        />
        <TextInput
          label={kind === 'percent' ? t('valuePercent') : t('valueToman')}
          hint={kind === 'percent' ? t('valuePercentHint') : t('valueTomanHint')}
          type="number"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          required
          error={err('value')}
        />
        <TextInput
          label={t('maxRedemptions')}
          hint={t('maxRedemptionsHint')}
          type="number"
          value={maxUses}
          onChange={(e) => setMaxUses(e.target.value)}
          error={err('max_redemptions')}
        />
        <TextInput
          label={t('perUserLimit')}
          hint={t('perUserLimitHint')}
          type="number"
          value={perUser}
          onChange={(e) => setPerUser(e.target.value)}
          error={err('per_user_limit')}
        />
        <Switch label={tc('isActive')} checked={active} onChange={setActive} />
        <TextInput
          label={t('startsAt')}
          hint={t('startsAtHint')}
          type="datetime-local"
          value={startsAt}
          onChange={(e) => setStartsAt(e.target.value)}
          error={err('starts_at')}
        />
        <TextInput
          label={t('expiresAt')}
          hint={t('expiresAtHint')}
          type="datetime-local"
          value={expiresAt}
          min={startsAt || undefined}
          onChange={(e) => setExpiresAt(e.target.value)}
          error={err('expires_at')}
        />
      </div>
      <CheckboxGrid
        label={t('plans')}
        hint={t('plansHint')}
        options={plans.map((p) => ({ value: String(p.id), label: localize(p.title) || p.code }))}
        value={planIds}
        onChange={setPlanIds}
        error={err('plan_ids')}
      />
    </FormPage>
  );
}
