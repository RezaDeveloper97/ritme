'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useEffect } from 'react';

import { INBOX_RULES_HREF } from '../lib/redirect';

/** /pregnancy-alert-rules — retired list (T-M10-13): the rules are inbox rules now, so this forwards to the filtered list. */
export function AlertRulesScreen() {
  const t = useTranslations('pregnancyAlertRules');
  const router = useRouter();
  useEffect(() => {
    router.replace(INBOX_RULES_HREF);
  }, [router]);
  return (
    <p className="panel-note" role="status">
      <Link href={INBOX_RULES_HREF}>{t('movedToInbox')}</Link>
    </p>
  );
}
