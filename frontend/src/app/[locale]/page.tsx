import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';

import { getDefaultLocale, isSupportedLocale } from '@/shared/i18n';

interface HomePageProps {
  params: Promise<{ locale: string }>;
}

export default async function RootLocalePage({ params }: HomePageProps) {
  const { locale } = await params;
  if (!(await isSupportedLocale(locale))) redirect(`/${await getDefaultLocale()}/splash`);

  const jar = await cookies();
  if (jar.has('ritme_onboarded')) {
    redirect(`/${locale}/home`);
  }
  redirect(`/${locale}/splash`);
}
