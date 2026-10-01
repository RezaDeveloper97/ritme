'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type FormEvent, useId, useState } from 'react';

import type { LogCustomItem, LogPreferences } from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { ChipGroup, Icon, PillChip, PrimaryButton, SecondaryButton } from '@/shared/ui';

import { useAddLogCustomItem, useDeleteLogCustomItem, useRenameLogCustomItem } from '../api/mutations';
import { CUSTOM_LABEL_MAX, labelProblem, normalizeLabel } from '../model/draft';
import { customItemError, type CustomItemError } from '../model/errors';

interface CustomItemsProps {
  prefs: LogPreferences;
  mode?: string;
  /** Announces a result through the screen's live region. */
  announce: (text: string) => void;
}

function useErrorText() {
  const t = useTranslations('logCustomize');
  const loc = useLocale() as Locale;
  return (e: CustomItemError, maxItems: number) =>
    t(`errors.${e}`, { max: formatNumber(e === 'tooLong' ? CUSTOM_LABEL_MAX : maxItems, loc) });
}

/** The labels a new or renamed item may not reuse: her other active items of the same category. */
function siblings(items: LogCustomItem[], category: string, exceptId?: number): string[] {
  return items.filter((i) => i.category === category && i.id !== exceptId).map((i) => i.label);
}

/**
 * «افزودن مورد سفارشی» (nbl_Log_Customize): her custom items with rename / delete, and the add form
 * (name + host category). Writes go out at once — they are not part of the order/visibility draft.
 */
export function CustomItems({ prefs, mode, announce }: CustomItemsProps) {
  const t = useTranslations('logCustomize');
  const loc = useLocale() as Locale;
  const errorText = useErrorText();
  const add = useAddLogCustomItem(mode);
  const rename = useRenameLogCustomItem(mode);
  const remove = useDeleteLogCustomItem(mode);
  const formId = useId();

  const hosts = prefs.categories.filter((c) => c.customParam);
  const items = prefs.customItems;
  const max = prefs.maxCustomItems;
  const full = items.length >= max;
  const catLabel = (code: string) => prefs.categories.find((c) => c.code === code)?.label ?? code;

  const [adding, setAdding] = useState(false);
  const [label, setLabel] = useState('');
  const [category, setCategory] = useState<string | null>(null);
  const [addError, setAddError] = useState<CustomItemError | null>(null);
  const chosen = category ?? hosts.find((h) => h.code === 'custom')?.code ?? hosts[0]?.code ?? null;

  const [editing, setEditing] = useState<{ id: number; value: string; error: CustomItemError | null } | null>(null);
  const [confirmId, setConfirmId] = useState<number | null>(null);
  const [rowError, setRowError] = useState<{ id: number; error: CustomItemError } | null>(null);

  if (!hosts.length) return null;

  const closeForm = () => {
    setAdding(false);
    setLabel('');
    setAddError(null);
  };

  const submitAdd = (e: FormEvent) => {
    e.preventDefault();
    if (!chosen) return setAddError('category');
    const problem = labelProblem(label, siblings(items, chosen));
    if (problem) return setAddError(problem);
    const clean = normalizeLabel(label);
    add.mutate(
      { category: chosen, label: clean },
      {
        onSuccess: () => {
          announce(t('custom.added', { label: clean }));
          closeForm();
        },
        onError: (err) => setAddError(customItemError(err)),
      },
    );
  };

  const submitRename = (item: LogCustomItem) => {
    if (!editing) return;
    const problem = labelProblem(editing.value, siblings(items, item.category, item.id));
    if (problem) return setEditing({ ...editing, error: problem });
    const clean = normalizeLabel(editing.value);
    if (clean === item.label) return setEditing(null);
    setEditing(null);
    rename.mutate(
      { id: item.id, label: clean },
      { onError: (err) => setRowError({ id: item.id, error: customItemError(err) }) },
    );
  };

  const confirmDelete = (item: LogCustomItem) => {
    setConfirmId(null);
    remove.mutate(item.id, {
      onSuccess: () => announce(t('custom.removed', { label: item.label })),
      onError: (err) => setRowError({ id: item.id, error: customItemError(err) }),
    });
  };

  return (
    <>
      {items.length ? (
        <section className="nb-card lcz-custom" aria-labelledby={`${formId}-title`}>
          <header className="lcz-custom-head">
            <h2 id={`${formId}-title`} className="lcz-custom-title">
              {t('custom.title')}
            </h2>
            <span className="lcz-count">
              {t('custom.count', { count: formatNumber(items.length, loc), max: formatNumber(max, loc) })}
            </span>
          </header>
          <ul className="lcz-items">
            {items.map((item) => {
              const isEditing = editing?.id === item.id;
              const errId = `${formId}-err-${item.id}`;
              const err = isEditing ? editing.error : rowError?.id === item.id ? rowError.error : null;
              return (
                <li key={item.id} className="lcz-item">
                  {isEditing ? (
                    <form
                      className="lcz-item-edit"
                      onSubmit={(e) => {
                        e.preventDefault();
                        submitRename(item);
                      }}
                    >
                      <input
                        className="lcz-input"
                        value={editing.value}
                        maxLength={CUSTOM_LABEL_MAX + 10}
                        aria-label={t('custom.label')}
                        aria-invalid={err ? true : undefined}
                        aria-describedby={err ? errId : undefined}
                         
                        autoFocus
                        onChange={(e) => setEditing({ ...editing, value: e.target.value, error: null })}
                        onKeyDown={(e) => e.key === 'Escape' && setEditing(null)}
                      />
                      <button type="submit" className="lcz-icon-btn is-brand" aria-label={t('custom.renameSave')}>
                        <Icon name="check" size={18} strokeWidth={2.4} />
                      </button>
                      <button
                        type="button"
                        className="lcz-icon-btn"
                        aria-label={t('custom.cancel')}
                        onClick={() => setEditing(null)}
                      >
                        <Icon name="x" size={18} />
                      </button>
                    </form>
                  ) : confirmId === item.id ? (
                    <div className="lcz-item-confirm" role="group" aria-label={t('custom.delete', { label: item.label })}>
                      <p className="lcz-item-q">{t('custom.deleteConfirm', { label: item.label })}</p>
                      <div className="lcz-item-acts">
                        <button type="button" className="lcz-text-btn is-danger" onClick={() => confirmDelete(item)}>
                          {t('custom.deleteYes')}
                        </button>
                        <button type="button" className="lcz-text-btn" onClick={() => setConfirmId(null)}>
                          {t('custom.deleteNo')}
                        </button>
                      </div>
                    </div>
                  ) : (
                    <>
                      <span className="lcz-item-text">
                        <span className="lcz-item-label">{item.label}</span>
                        <span className="lcz-item-cat">{t('custom.inCategory', { category: catLabel(item.category) })}</span>
                      </span>
                      <button
                        type="button"
                        className="lcz-icon-btn"
                        aria-label={t('custom.rename', { label: item.label })}
                        onClick={() => {
                          setRowError(null);
                          setEditing({ id: item.id, value: item.label, error: null });
                        }}
                      >
                        <Icon name="pencil" size={17} />
                      </button>
                      <button
                        type="button"
                        className="lcz-icon-btn"
                        aria-label={t('custom.delete', { label: item.label })}
                        onClick={() => {
                          setRowError(null);
                          setConfirmId(item.id);
                        }}
                      >
                        <Icon name="trash" size={17} />
                      </button>
                    </>
                  )}
                  {err ? (
                    <p id={errId} className="lcz-field-err" role="alert">
                      {errorText(err, max)}
                    </p>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </section>
      ) : null}

      {adding ? (
        <form className="nb-card lcz-add-form" onSubmit={submitAdd} aria-labelledby={`${formId}-form`} noValidate>
          <h2 id={`${formId}-form`} className="lcz-custom-title">
            {t('custom.formTitle')}
          </h2>
          <label className="lcz-field">
            <span className="lcz-field-label">{t('custom.label')}</span>
            <input
              className="lcz-input"
              value={label}
              placeholder={t('custom.placeholder')}
              maxLength={CUSTOM_LABEL_MAX + 10}
              aria-invalid={addError === 'empty' || addError === 'tooLong' || addError === 'duplicate' ? true : undefined}
              aria-describedby={addError ? `${formId}-adderr` : undefined}
               
              autoFocus
              onChange={(e) => {
                setLabel(e.target.value);
                setAddError(null);
              }}
            />
          </label>
          <span className="lcz-field-label" aria-hidden>
            {t('custom.category')}
          </span>
          <ChipGroup label={t('custom.category')}>
            {hosts.map((h) => (
              <PillChip
                key={h.code}
                pressed={chosen === h.code}
                onPressedChange={() => {
                  setCategory(h.code);
                  setAddError(null);
                }}
              >
                {h.label}
              </PillChip>
            ))}
          </ChipGroup>
          {addError ? (
            <p id={`${formId}-adderr`} className="lcz-field-err" role="alert">
              {errorText(addError, max)}
            </p>
          ) : null}
          <div className="lcz-add-acts">
            <PrimaryButton type="submit" loading={add.isPending} block={false} className="lcz-add-submit">
              {t('custom.submit')}
            </PrimaryButton>
            <SecondaryButton variant="text" block={false} onClick={closeForm}>
              {t('custom.cancel')}
            </SecondaryButton>
          </div>
        </form>
      ) : (
        <button
          type="button"
          className="nb-card lcz-add"
          disabled={full}
          aria-describedby={full ? `${formId}-full` : undefined}
          onClick={() => setAdding(true)}
        >
          <Icon name="plus" size={17} strokeWidth={2.4} className="lcz-add-icon" />
          <span>{t('custom.add')}</span>
        </button>
      )}
      {full ? (
        <p id={`${formId}-full`} className="lcz-note">
          {t('custom.full', { max: formatNumber(max, loc) })}
        </p>
      ) : null}
    </>
  );
}
