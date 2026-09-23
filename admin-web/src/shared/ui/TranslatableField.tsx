'use client';

import { useTranslations } from 'next-intl';
import { useId } from 'react';

import { useContentLanguages, type ContentLanguage } from '@/shared/i18n';

import { ErrorState } from './ErrorState';
import { RichTextEditor } from './RichTextEditorLazy';
import { Skeleton } from './Skeleton';

/** `{fa: '…', en: '…'}` — keyed by content-language code (admin-api.md §3). */
export type Translations = Record<string, string>;

export type TranslatableKind = 'text' | 'textarea' | 'rich';

export interface TranslatableFieldProps {
  /** Field name as the API knows it (`title`); errors are read as `title.<code>`. */
  name: string;
  label: string;
  value: Translations;
  onChange: (value: Translations) => void;
  kind?: TranslatableKind;
  /** Only the DEFAULT language becomes required (frontend/CLAUDE.md §6.3). */
  required?: boolean;
  /** Error bag from an ApiError (`error.fieldErrors`) or the form's own. */
  errors?: Record<string, readonly string[] | undefined>;
  maxLength?: number;
}

/**
 * One input per active content language, from GET /api/v1/languages. Adding a
 * language in the admin grows every form with no code change — never pass a
 * hard-coded language list.
 */
export function TranslatableField(props: TranslatableFieldProps) {
  const query = useContentLanguages();
  if (query.error && !query.data) return <ErrorState error={query.error} onRetry={() => query.refetch()} />;
  if (!query.data) {
    return (
      <div className="field" aria-busy="true">
        <span className="field-label">{props.label}</span>
        <Skeleton className="h-9 w-full" />
      </div>
    );
  }
  return <TranslatableInputs {...props} languages={query.data.languages} />;
}

/** Presentational half, given the language rows (also used by tests). */
export function TranslatableInputs({
  name,
  label,
  value,
  onChange,
  kind = 'text',
  required = false,
  errors,
  maxLength,
  languages,
}: TranslatableFieldProps & { languages: readonly ContentLanguage[] }) {
  const t = useTranslations('form');
  const baseId = useId();
  // Default first, then the table's order.
  const ordered = [...languages].sort((a, b) => Number(b.is_default) - Number(a.is_default));

  return (
    <fieldset className="field m-0 min-w-0 border-0 p-0">
      <legend className="field-label mb-1.5 p-0">{label}</legend>
      <div className={kind === 'text' ? 'grid gap-3 md:grid-cols-2' : 'flex flex-col gap-3'}>
        {ordered.map((lang) => {
          const id = `${baseId}-${lang.code}`;
          const isRequired = required && lang.is_default;
          const error = errors?.[`${name}.${lang.code}`]?.[0];
          const describedBy = error ? `${id}-error` : lang.is_default ? undefined : `${id}-hint`;
          const set = (next: string) => onChange({ ...value, [lang.code]: next });
          const common = {
            id,
            dir: lang.direction,
            lang: lang.code,
            'aria-invalid': error ? true : undefined,
            'aria-describedby': describedBy,
          } as const;
          return (
            <div key={lang.code} className="field" data-lang={lang.code}>
              <label htmlFor={id} className="lang-tag">
                {lang.name}
                <span className="lang-code">{lang.code}</span>
                {lang.is_default ? <span className="badge badge-brand">{t('default')}</span> : null}
                {isRequired ? (
                  <span className="field-required" aria-hidden="true">
                    *
                  </span>
                ) : null}
              </label>
              {kind === 'rich' ? (
                <RichTextEditor
                  id={id}
                  dir={lang.direction}
                  value={value[lang.code] ?? ''}
                  onChange={set}
                  invalid={Boolean(error)}
                  ariaLabel={`${label} — ${lang.name}`}
                  ariaDescribedBy={describedBy}
                />
              ) : kind === 'textarea' ? (
                <textarea
                  {...common}
                  name={`${name}[${lang.code}]`}
                  className="input"
                  required={isRequired}
                  maxLength={maxLength}
                  value={value[lang.code] ?? ''}
                  onChange={(e) => set(e.target.value)}
                />
              ) : (
                <input
                  {...common}
                  type="text"
                  name={`${name}[${lang.code}]`}
                  className="input"
                  required={isRequired}
                  maxLength={maxLength}
                  value={value[lang.code] ?? ''}
                  onChange={(e) => set(e.target.value)}
                />
              )}
              {error ? (
                <span className="field-error" id={`${id}-error`} role="alert">
                  {error}
                </span>
              ) : !lang.is_default ? (
                <span className="field-hint" id={`${id}-hint`}>
                  {t('optionalTranslation')}
                </span>
              ) : null}
            </div>
          );
        })}
      </div>
    </fieldset>
  );
}
