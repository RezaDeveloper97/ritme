'use client';

import { useTranslations } from 'next-intl';

import { Link } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

/** «گزارش ۳ ماه برای پزشک» under the treatment card (CB-MENO-11, nbl_Meno_Home → Meno_Report). */
export function MenopauseReportLink() {
  const t = useTranslations('menopause.report');
  return (
    <Link href="/menopause/report" className="nb-btn is-outline is-block mh-report-link">
      <Icon name="fileDoc" size={20} />
      {t('homeCta')}
    </Link>
  );
}
