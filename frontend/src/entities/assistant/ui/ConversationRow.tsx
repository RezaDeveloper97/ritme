'use client';

import { useTranslations } from 'next-intl';

import { Link } from '@/shared/i18n';
import { IconCircle } from '@/shared/ui';

import type { Conversation } from '../api/schema';
import { deptIcon } from '../model/format';
import { TriagePill } from './AssistantBits';
import '@/shared/ui/styles/as.css';

/** Relative day label for a conversation row («امروز», «دیروز», «۲ هفته پیش»). */
export function useWhen() {
  const t = useTranslations('assistant');
  return (iso: string | null): string => {
    if (!iso) return '';
    const days = Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000);
    if (days <= 0) return t('when.today');
    if (days === 1) return t('when.yesterday');
    if (days < 7) return t('when.days', { count: days });
    if (days < 30) return t('when.weeks', { count: Math.floor(days / 7) });
    return t('when.months', { count: Math.max(1, Math.floor(days / 30)) });
  };
}

/** One conversation: department icon, title, «department · when», triage pill. Opens the chat. */
export function ConversationRow({ c }: { c: Conversation }) {
  const t = useTranslations('assistant');
  const when = useWhen();
  return (
    <Link href={`/assistant/chat/${c.id}`} className="as-row">
      <IconCircle icon={deptIcon(c.department.icon)} tone="brand" size="md" />
      <span className="as-row-main">
        <strong>{c.title || t('history.untitled')}</strong>
        <span>{[c.department.name, when(c.last_message_at ?? c.created_at)].filter(Boolean).join(' · ')}</span>
      </span>
      {c.triage_level ? <TriagePill level={c.triage_level} /> : null}
    </Link>
  );
}
