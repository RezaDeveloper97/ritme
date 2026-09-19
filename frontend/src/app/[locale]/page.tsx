import { cookies, headers } from 'next/headers';
import { redirect } from 'next/navigation';

import { getDefaultLocale, isSupportedLocale } from '@/shared/i18n';
import { isShellUserAgent } from '@/shared/pwa';
import { INTRO_COOKIE, postSplashRoute } from '@/shared/session';

interface HomePageProps {
  params: Promise<{ locale: string }>;
}

export default async function RootLocalePage({ params }: HomePageProps) {
  const { locale } = await params;
  if (!(await isSupportedLocale(locale))) redirect(`/${await getDefaultLocale()}/splash`);

  // Signed-in visitors are not special-cased here: the flag cookie can be
  // missing while the token is still valid, so the client's SessionGuard (which
  // can see the token) moves them off the splash instead.
  const jar = await cookies();

  // The Android shell has already shown a splash of its own and the middleware
  // sends it past this one, so routing it through `/splash` would only buy an
  // extra round-trip on the slowest moment the app has — its cold start.
  if (isShellUserAgent((await headers()).get('user-agent'))) {
    redirect(postSplashRoute(locale, jar.has(INTRO_COOKIE)));
  }

  redirect(`/${locale}/splash`);
}
