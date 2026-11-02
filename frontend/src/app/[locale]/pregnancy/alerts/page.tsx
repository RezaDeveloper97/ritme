import { redirect } from 'next/navigation';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/pregnancy/alerts` — the old pregnancy-only screen; every mode now shares `/alerts` (T-M10-15). The URL keeps working. */
export default async function PregnancyAlertsRoute({ params }: Props) {
  const { locale } = await params;
  redirect(`/${locale}/alerts`);
}
