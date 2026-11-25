import { redirect } from 'next/navigation';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/alerts` — alerts have no screen any more: urgent ones come up as a modal and every alert lives in the
 * notifications sheet. Old links, push taps and message buttons pointing here open that sheet on «امروز»: `/home` is
 * mode-aware and carries `?sheet=` through its hand-off (pregnancy, postpartum, IVF, companion). The bare `/{locale}`
 * redirects again and would drop the query. */
export default async function AlertsRoute({ params }: Props) {
  const { locale } = await params;
  redirect(`/${locale}/home?sheet=notifications`);
}
