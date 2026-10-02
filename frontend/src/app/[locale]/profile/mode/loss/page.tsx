import { redirect } from 'next/navigation';

interface Props {
  params: Promise<{ locale: string }>;
}

/**
 * `/profile/mode/loss` — bloom's calm pregnancy exit (B-N2-03) now opens the
 * full loss path (CB-LOSS-02) instead of a second, shorter exit; kept as a
 * redirect so an old link still lands on `/loss`.
 */
export default async function ProfileModeLossRoute({ params }: Props) {
  const { locale } = await params;
  redirect(`/${locale}/loss`);
}
