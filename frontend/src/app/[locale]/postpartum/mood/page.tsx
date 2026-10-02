import { setRequestLocale } from 'next-intl/server';

import { PostpartumMoodPage } from '@/screens/postpartum-mood';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ kind?: string }>;
}

/** `/postpartum/mood?kind=short|full` — EPDS check + safety path (nbl_v15_MoodCheck). Back header, no bottom nav. */
export default async function PostpartumMoodRoute({ params, searchParams }: Props) {
  const { locale } = await params;
  const { kind } = await searchParams;
  setRequestLocale(locale);
  return (
    <RouteMessages route="postpartumMood">
      <PostpartumMoodPage key={kind ?? 'due'} kind={kind === 'full' || kind === 'short' ? kind : null} />
    </RouteMessages>
  );
}
