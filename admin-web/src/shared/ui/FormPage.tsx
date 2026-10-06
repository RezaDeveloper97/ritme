'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import type { FormEvent, ReactNode } from 'react';

import { Button } from './Button';
import { PageHeader } from './PageHeader';
import { Panel } from './Panel';

/**
 * A create/edit screen: header with a way back, one panel holding the fields,
 * and a save bar that stays in view on long forms. `after` renders below the
 * form (secondary panels such as "regenerate").
 */
export function FormPage({
  title,
  backHref,
  backLabel,
  meta,
  headerActions,
  onSubmit,
  submitLabel,
  saving,
  children,
  after,
  readOnly = false,
}: {
  title: ReactNode;
  backHref: string;
  backLabel: string;
  meta?: ReactNode;
  headerActions?: ReactNode;
  onSubmit: () => void;
  submitLabel: string;
  saving: boolean;
  children: ReactNode;
  after?: ReactNode;
  /** No save button (the viewer may read but not write, e.g. an editor on super-only content). */
  readOnly?: boolean;
}) {
  const t = useTranslations('crud');
  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!saving && !readOnly) onSubmit();
  };
  return (
    <div className="flex flex-col gap-5">
      <PageHeader title={title} backHref={backHref} backLabel={backLabel} meta={meta} actions={headerActions} />
      <form onSubmit={submit}>
        <Panel bodyClassName="">
          <div className="flex flex-col gap-5 p-4 sm:p-5">{children}</div>
          <div className="form-actions">
            {readOnly ? null : (
              <Button type="submit" variant="primary" loading={saving}>
                {submitLabel}
              </Button>
            )}
            <Link href={backHref} className="btn btn-ghost">
              {t('cancel')}
            </Link>
          </div>
        </Panel>
      </form>
      {after}
    </div>
  );
}
