'use client';

import { useTranslations } from 'next-intl';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';

import { AddStudentsForm, type Scope } from '@/features/grant-access';
import { Icon } from '@/shared/ui';

/** /students/add — nbl_Ins_AddUser. `?scope=group|course|standalone&target=ID` pre-selects the target. */
export function AddStudentsScreen() {
  const t = useTranslations('addStudents');
  const params = useSearchParams();
  const raw = params.get('scope');
  const scope: Scope = raw === 'course' || raw === 'standalone' ? raw : 'group';
  const target = Number(params.get('target'));

  return (
    <>
      <div className="page page--narrow">
        <header className="page-head">
          <Link href="/students" className="icon-btn" aria-label={t('back')}>
            <Icon name="back" />
          </Link>
          <div className="page-head__text">
            <h1 className="page-head__title">{t('title')}</h1>
          </div>
        </header>
      </div>
      <AddStudentsForm initialScope={scope} initialTarget={Number.isInteger(target) && target > 0 ? target : null} />
    </>
  );
}
