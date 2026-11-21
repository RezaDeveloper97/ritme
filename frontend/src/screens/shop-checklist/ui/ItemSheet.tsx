'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  CHECKLIST_GROUPS,
  CHECKLIST_STATES,
  type ChecklistGroupKey,
  type ChecklistState,
  type ShopChecklistItem,
  useAddCustomItem,
  useDeleteCustomItem,
  useSetChecklistItemState,
  useUpdateCustomItem,
} from '@/entities/shop';
import type { Locale } from '@/shared/i18n';
import { pilot, pilotOption } from '@/shared/pilot';
import { AppSheet } from '@/shared/sheet';
import { ChipGroup, NumberStepper, PillChip, PrimaryButton, SecondaryButton } from '@/shared/ui';
import './shc.css';

interface Props {
  /** The row being edited; null = add a custom item. */
  item: ShopChecklistItem | null;
  /** Group a new item starts in. */
  defaultGroup: ChecklistGroupKey;
  onClose: () => void;
}

/** Edit one row's state + quantity (template or custom), or add a custom row. Her list only — never the family view. */
export function ItemSheet({ item, defaultGroup, onClose }: Props) {
  const t = useTranslations('shop.checklist');
  const locale = useLocale() as Locale;
  const setState = useSetChecklistItemState();
  const add = useAddCustomItem();
  const update = useUpdateCustomItem();
  const remove = useDeleteCustomItem();

  const [title, setTitle] = useState(item?.title ?? '');
  const [group, setGroup] = useState<ChecklistGroupKey>(defaultGroup);
  const [state, setStateValue] = useState<ChecklistState>(item?.state ?? 'todo');
  const [qty, setQty] = useState(item?.quantity ?? item?.suggested_qty ?? 1);

  const busy = setState.isPending || add.isPending || update.isPending || remove.isPending;
  const failed = setState.isError || add.isError || update.isError || remove.isError;
  const showQty = state === 'have' || state === 'in_cart' || state === 'on_the_way';
  const canSave = item ? true : title.trim().length > 0;

  function save() {
    const quantity = showQty ? qty : null;
    const done = { onSuccess: onClose };
    if (!item) {
      add.mutate({ title: title.trim(), group, state, quantity }, done);
    } else if (item.is_custom && item.id !== null) {
      update.mutate({ id: item.id, state, quantity }, done);
    } else if (item.layette_item_id !== null) {
      setState.mutate({ layetteItemId: item.layette_item_id, state, quantity }, done);
    }
  }

  return (
    <AppSheet
      open
      onClose={busy ? () => undefined : onClose}
      size="half"
      title={item ? item.title : t('add.title')}
      footer={
        <div className="shc-sheet-actions">
          <PrimaryButton loading={busy} disabled={!canSave} onClick={save} {...pilot('shop-checklist.save')}>
            {t('sheet.save')}
          </PrimaryButton>
          {item?.is_custom && item.id !== null ? (
            <SecondaryButton variant="text" disabled={busy} onClick={() => remove.mutate(item.id as number, { onSuccess: onClose })}>
              {t('sheet.remove')}
            </SecondaryButton>
          ) : (
            <SecondaryButton variant="text" disabled={busy} onClick={onClose}>
              {t('sheet.cancel')}
            </SecondaryButton>
          )}
        </div>
      }
    >
      <div className="shc-sheet">
        {!item ? (
          <>
            <label className="lbl" htmlFor="shc-title">
              {t('add.name')}
            </label>
            <div className="field">
              <input id="shc-title" value={title} maxLength={120} onChange={(e) => setTitle(e.target.value)} placeholder={t('add.placeholder')} {...pilot('shop-checklist.title', 'text')} />
            </div>
            <ChipGroup label={t('add.group')} pilot={pilot('shop-checklist.group', 'choice')}>
              {CHECKLIST_GROUPS.map((g) => (
                <PillChip key={g} pressed={group === g} onPressedChange={() => setGroup(g)} {...pilotOption(g)}>
                  {t(`groups.${g}`)}
                </PillChip>
              ))}
            </ChipGroup>
          </>
        ) : null}
        {item?.note ? <p className="shc-sheet-note">{item.note}</p> : null}
        <ChipGroup label={t('sheet.state')} pilot={pilot('shop-checklist.state', 'choice')}>
          {CHECKLIST_STATES.map((s) => (
            <PillChip key={s} pressed={state === s} onPressedChange={() => setStateValue(s)} {...pilotOption(s)}>
              {t(`state.${s}`)}
            </PillChip>
          ))}
        </ChipGroup>
        {showQty ? (
          <NumberStepper
            value={qty}
            onChange={setQty}
            pilot={pilot('shop-checklist.qty')}
            min={1}
            max={99}
            label={t('sheet.qty')}
            decrementLabel={t('sheet.less')}
            incrementLabel={t('sheet.more')}
            locale={locale}
          />
        ) : null}
        {failed ? (
          <p className="shc-error" role="alert">
            {t('sheet.error')}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}
