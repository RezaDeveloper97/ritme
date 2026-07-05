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

        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10 }}>
          <div style={{ textAlign: 'start' }}>
            <div style={{ fontSize: 17, fontWeight: 800, color: 'var(--ink)' }}>
              {t(`categories.${category.key}`)}
            </div>
            <p className="sub" style={{ margin: '4px 0 0' }}>{t('sheetHint')}</p>
          </div>
          <button className="iconbtn" onClick={onClose} aria-label={t('done')}>
            <Icon name="x" size={20} />
          </button>
        </div>

        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 18,
            margin: '18px 0 4px',
            maxHeight: '58vh',
            overflowY: 'auto',
            paddingInlineEnd: 2,
          }}
        >
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
              <div style={{ display: 'flex', justifyContent: 'center', padding: 24, color: 'var(--muted)' }}>
                {t('loading')}
              </div>
            )}
        </div>

        <button className="btn btn-primary" onClick={onClose} style={{ borderRadius: 14, marginTop: 16 }}>
          {t('done')}
        </button>
      </div>
    </div>
  );
}
