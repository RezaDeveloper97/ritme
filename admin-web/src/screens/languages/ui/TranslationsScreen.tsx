'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useState } from 'react';

import { RequireSuper } from '@/features/auth';
import { useContentLanguages } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib';
import {
  ErrorState,
  FormPage,
  LinkTabs,
  PageHeader,
  Panel,
  Skeleton,
  toast,
  useNotifyError,
} from '@/shared/ui';

import { useSaveTranslations, useTranslationsPage, type TranslationsPage } from '../api/languages';

/** /languages/:id/translations?namespace= — the app's UI strings for one language (Blade translations.index). */
export function TranslationsScreen({ id }: { id: number }) {
  return (
    <RequireSuper>
      <TranslationsLoader id={id} />
    </RequireSuper>
  );
}

function TranslationsLoader({ id }: { id: number }) {
  const t = useTranslations('translations');
  const namespace = useSearchParams().get('namespace') ?? '';
  const query = useTranslationsPage(id, namespace);
  const header = <PageHeader title={t('title')} backHref="/languages" backLabel={t('backToLanguages')} />;

  if (query.error && !query.data) {
    return (
      <div className="flex flex-col gap-4">
        {header}
        <Panel>
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        </Panel>
      </div>
    );
  }
  if (!query.data) {
    return (
      <div className="flex flex-col gap-4" aria-busy="true">
        {header}
        <Skeleton className="h-10 w-full" />
        <Panel>
          <div className="flex flex-col gap-3">
            {Array.from({ length: 6 }, (_, i) => (
              <Skeleton key={i} className="h-9 w-full" />
            ))}
          </div>
        </Panel>
      </div>
    );
  }
  // Remount per namespace so the draft starts from that namespace's values.
  return <TranslationsEditor key={`${query.data.namespace}:${query.dataUpdatedAt}`} id={id} page={query.data} />;
}

function TranslationsEditor({ id, page }: { id: number; page: TranslationsPage }) {
  const t = useTranslations('translations');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const save = useSaveTranslations(id);
  const languages = useContentLanguages().data?.languages;
  const initial = Object.fromEntries(page.rows.map((r) => [r.key, r.value ?? '']));
  const [draft, setDraft] = useState<Record<string, string>>(initial);
  const changed = page.rows.filter((r) => (draft[r.key] ?? '') !== (r.value ?? '')).length;
  const { language } = page;
  const dir = language.direction === 'rtl' ? 'rtl' : 'ltr';
  const refDir = languages?.find((l) => l.code === page.default_code)?.direction;

  const submit = () =>
    save.mutate(
      { namespace: page.namespace, rows: page.rows.map((r) => ({ key: r.key, value: draft[r.key] ?? '' })) },
      {
        onSuccess: (data) => toast.success(t('savedToast', { count: formatNumber(data.saved, locale), namespace: page.namespace })),
        onError: notifyError,
      },
    );

  return (
    <div className="flex flex-col gap-4">
      <FormPage
        title={t('titleFor', { language: language.name })}
        backHref="/languages"
        backLabel={t('backToLanguages')}
        meta={
          <span>
            {t('section', { namespace: page.namespace })}
            {changed ? <span className="ms-2 font-semibold text-amber-deep">{t('unsaved', { count: changed })}</span> : null}
          </span>
        }
        onSubmit={submit}
        submitLabel={t('saveSection', { namespace: page.namespace })}
        saving={save.isPending}
      >
        <LinkTabs
          label={t('namespaces')}
          items={page.namespaces.map((ns) => ({
            key: ns,
            href: `/languages/${id}/translations?namespace=${encodeURIComponent(ns)}`,
            label: ns,
            active: ns === page.namespace,
          }))}
        />
        <p className="field-hint m-0">
          {page.is_default_locale ? t('hintDefault') : t('hintOther', { language: page.default_name })}{' '}
          {t('placeholders')}
        </p>
        {page.rows.length === 0 ? (
          <p className="m-0 text-center text-muted">{t('empty')}</p>
        ) : (
          <div className="table-wrap -mx-4 sm:-mx-5">
            <table className="data-table">
              <caption className="sr-only">{t('section', { namespace: page.namespace })}</caption>
              <thead>
                <tr>
                  <th scope="col" className="w-[26%]">
                    {t('key')}
                  </th>
                  {page.is_default_locale ? null : (
                    <th scope="col" className="w-[32%]">
                      {page.default_name}
                    </th>
                  )}
                  <th scope="col">{language.name}</th>
                </tr>
              </thead>
              <tbody>
                {page.rows.map((row) => {
                  const ref = row.reference ?? '';
                  const value = draft[row.key] ?? '';
                  return (
                    <tr key={row.key}>
                      <td className="align-top">
                        <span dir="ltr" className="i18n-key">
                          {row.key}
                        </span>
                      </td>
                      {page.is_default_locale ? null : (
                        <td className="align-top">
                          <span dir={refDir} className="i18n-ref">
                            {ref || '—'}
                          </span>
                        </td>
                      )}
                      <td>
                        <textarea
                          className="input i18n-input"
                          aria-label={`${row.key} (${language.name})`}
                          dir={dir}
                          rows={Array.from(ref || value).length > 90 ? 3 : 1}
                          placeholder={page.is_default_locale ? '' : ref}
                          value={value}
                          onChange={(e) => setDraft((d) => ({ ...d, [row.key]: e.target.value }))}
                        />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </FormPage>
    </div>
  );
}
