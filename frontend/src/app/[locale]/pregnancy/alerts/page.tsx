import { redirect } from 'next/navigation';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/pregnancy/alerts` — the old pregnancy-only screen; alerts now live in the notifications sheet. The URL keeps working. */
export default async function PregnancyAlertsRoute({ params }: Props) {
  const { locale } = await params;
  redirect(`/${locale}?sheet=notifications`);
}
