import { getTranslations } from 'next-intl/server';
import Link from 'next/link';

export default async function NotFound() {
  const t = await getTranslations('errors');
  const nav = await getTranslations('nav');
  return (
    <main className="grid min-h-dvh place-items-center p-6 text-center">
      <div className="flex flex-col items-center gap-3">
        <p className="m-0 text-ink-3">{t('not_found')}</p>
        <Link href="/" className="btn btn-primary">
          {nav('dashboard')}
        </Link>
      </div>
    </main>
  );
}
