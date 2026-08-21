'use client';

import { useTranslations } from 'next-intl';

import type {
  CategoryDef,
  HealthLogEnums,
  HealthLogField,
  HealthLogInput,
} from '@/entities/health-log';
import { AppSheet } from '@/shared/sheet';

import { FieldRow } from './FieldRow';

interface CategorySheetProps {
  category: CategoryDef;
  enums: HealthLogEnums | undefined;
  draft: HealthLogInput;
  onChange: (key: HealthLogField, value: unknown) => void;
  /** Discard this sheet's edits and close. */
  onCancel: () => void;
  /** Send this sheet's edits to the server and close. */
  onSubmit: () => void;
  isSaving: boolean;
}

/**
 * Bottom sheet for one log category (the Figma "Add Log" step sheets). Edits
 * stay in the local draft while the sheet is open; only "ثبت" (submit) pushes
 * them to the server, so the user stays in control of what gets recorded.
 * RTL-safe and fully i18n'd (CLAUDE.md §6, §12).
 */
export function CategorySheet({
  category,
  enums,
  draft,
  onChange,
  onCancel,
  onSubmit,
  isSaving,
}: CategorySheetProps) {
  const t = useTranslations('log');

  return (
    // Half: a handful of field rows that should size to themselves. A category
    // with many options grows the sheet rather than scrolling it (see AppSheet).
    <AppSheet
      open
      onClose={onCancel}
      size="half"
      title={t(`categories.${category.key}`)}
      footer={
        <button
          className="btn btn-primary"
          onClick={onSubmit}
          disabled={!enums || isSaving}
        >
          {isSaving ? t('saving') : t('submit')}
        </button>
      }
    >
      <p className="sub cats-head-s">{t('sheetHint')}</p>

      <div className="cats-body">
        {enums
          ? category.fields.map((field) => (
              <FieldRow
                key={field.key}
                field={field}
                enums={enums}
                value={draft[field.key]}
                onChange={onChange}
              />
            ))
          : (
            <div className="cats-empty">
              {t('loading')}
            </div>
          )}
      </div>
    </AppSheet>
  );
}
