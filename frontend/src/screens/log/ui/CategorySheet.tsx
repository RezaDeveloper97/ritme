'use client';

import { useTranslations } from 'next-intl';

import type {
  CategoryDef,
  HealthLogEnums,
  HealthLogField,
  HealthLogInput,
} from '@/entities/health-log';
import { Icon } from '@/shared/ui';

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
    <div className="sheet-backdrop" onClick={onCancel}>
      <div className="sheet" onClick={(e) => e.stopPropagation()}>
        <div className="sheet-grip" />

        <div className="cats-head">
          <div className="text-start">
            <div className="cats-head-t">
              {t(`categories.${category.key}`)}
            </div>
            <p className="sub cats-head-s">{t('sheetHint')}</p>
          </div>
          <button className="iconbtn" onClick={onCancel} aria-label={t('close')}>
            <Icon name="x" size={20} />
          </button>
        </div>

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

        <button
          className="btn btn-primary cats-done"
          onClick={onSubmit}
          disabled={!enums || isSaving}
        >
          {isSaving ? t('saving') : t('submit')}
        </button>
      </div>
    </div>
  );
}
