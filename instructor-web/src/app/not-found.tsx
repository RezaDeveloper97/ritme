import { getTranslations } from 'next-intl/server';
import Link from 'next/link';

export default async function NotFound() {
  const t = await getTranslations('common');
  return (
    <div className="gate-center">
      <div className="state-card">
        <p className="state-card__title">{t('notFound')}</p>
        <Link href="/" className="btn btn--ghost">
          {t('home')}
        </Link>
      </div>
    </div>
  );
}
