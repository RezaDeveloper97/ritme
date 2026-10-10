'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useEffect, useRef } from 'react';

import { EmergencyCard, TriagePill } from '@/entities/assistant';
import { Icon } from '@/shared/ui';

import type { ChatItem } from '../model/useChat';

interface Props {
  items: ChatItem[];
  busy: boolean;
  /** Tapping a chip sends it as the next message. */
  onChip: (text: string) => void;
}

/** The message list: user bubbles, assistant bubbles with chips + triage pill, the 115 card, streaming caret. */
export function ChatThread({ items, busy, onChip }: Props) {
  const t = useTranslations('assistant');
  const endRef = useRef<HTMLDivElement>(null);
  const last = items[items.length - 1];
  const tick = `${items.length}:${last?.text.length ?? 0}`;
  useEffect(() => {
    endRef.current?.scrollIntoView({ block: 'end' });
  }, [tick]);
  return (
    <div className="as-thread" role="log" aria-live="polite" aria-busy={busy} aria-label={t('chat.thread')}>
      {items.map((m, idx) => {
        if (m.role === 'card') return <EmergencyCard key={m.key} card={m.card} />;
        if (m.role === 'user')
          return (
            <div key={m.key} className="as-msg is-mine">
              {m.hasPhoto ? (
                <div className="as-photo" aria-label={t('chat.photoAttached')}>
                  <Icon name="camera" size={26} />
                </div>
              ) : null}
              <div className="as-bubble">{m.text}</div>
            </div>
          );
        const isLast = idx === items.length - 1;
        return (
          <div key={m.key} className="as-msg is-bot">
            <span className="as-bot-avatar" aria-hidden><Icon name="sparkle" size={14} /></span>
            <div className="as-bubble-col">
              <div className={clsx('as-bubble', m.state === 'streaming' && 'is-streaming')}>
                {m.text || (m.state === 'streaming' ? <span className="as-typing" aria-label={t('chat.typing')}><i /><i /><i /></span> : null)}
                {m.state === 'streaming' && m.text ? <span className="as-caret" aria-hidden /> : null}
                {m.state === 'partial' ? <p className="as-partial">{t('chat.partial')}</p> : null}
                {m.triage && m.state !== 'streaming' ? (
                  <div className="as-triage"><TriagePill level={m.triage} /></div>
                ) : null}
                {m.chips.length && isLast && !busy ? (
                  <div className="as-chips">
                    {m.chips.map((c) => (
                      <button key={c} type="button" className="as-chip" onClick={() => onChip(c)}>{c}</button>
                    ))}
                  </div>
                ) : null}
              </div>
            </div>
          </div>
        );
      })}
      <div ref={endRef} />
    </div>
  );
}
