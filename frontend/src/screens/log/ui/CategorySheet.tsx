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
  onClose: () => void;
}

/**
 * Bottom sheet for one log category (the Figma "Add Log" step sheets). Edits
 * write straight into the shared draft, so closing keeps every change — there's
 * no per-sheet save. RTL-safe and fully i18n'd (CLAUDE.md §6, §12).
 */
export function CategorySheet({ category, enums, draft, onChange, onClose }: CategorySheetProps) {
  const t = useTranslations('log');

  return (
    <div className="sheet-backdrop" onClick={onClose}>
      <div className="sheet" onClick={(e) => e.stopPropagation()}>
        <div className="sheet-grip" />

        <div className="cats-head">
          <div className="text-start">
            <div className="cats-head-t">
              {t(`categories.${category.key}`)}
            </div>
            <p className="sub cats-head-s">{t('sheetHint')}</p>
          </div>
          <button className="iconbtn" onClick={onClose} aria-label={t('done')}>
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

        <button className="btn btn-primary cats-done" onClick={onClose}>
          {t('done')}
        </button>
      </div>
    </div>
  );
}
