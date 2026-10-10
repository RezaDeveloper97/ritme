'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useRef, useState } from 'react';

import { type AssistantMessage, assistantKeys, messageSchema, type TriageLevel } from '@/entities/assistant';
import { getApiErrorCode, getApiErrorStatus } from '@/shared/api';
import { type PlusDenial, plusDenialOf } from '@/shared/ui/plus-gate';

import { streamMessage } from '../api/stream';

export interface ChatItem {
  key: string;
  role: 'user' | 'assistant' | 'card';
  text: string;
  card: AssistantMessage['card'];
  chips: string[];
  triage: TriageLevel | null;
  hasPhoto: boolean;
  /** `streaming` while deltas arrive; `partial` = the server cut the answer; `failed` = a local-only error bubble. */
  state: 'streaming' | 'complete' | 'partial' | 'failed';
}

const fromMessage = (m: AssistantMessage): ChatItem => ({
  key: `m${m.id}`,
  role: m.role,
  text: m.text,
  card: m.card,
  chips: m.chips,
  triage: m.triage_level,
  hasPhoto: m.photo !== null,
  state: m.status === 'partial' ? 'partial' : 'complete',
});

export const itemsOf = (messages: AssistantMessage[]): ChatItem[] => messages.map(fromMessage);

export interface ChatState {
  items: ChatItem[];
  busy: boolean;
  /** The text the last failed send carried, for the retry button. */
  failedText: string | null;
  errorCode: string | null;
  /** 402 from the Plus gate (free quota spent / locked). */
  denial: PlusDenial | null;
  /** 403 consent_required — the `ai_assistant` consent must be accepted first. */
  needsConsent: boolean;
  emergency: boolean;
}

/** Marks the streaming bubble `key` partial (it has text) or drops it (nothing arrived). */
const settle = (items: ChatItem[], key: string, empty: 'drop' | 'failed' = 'drop'): ChatItem[] =>
  items.flatMap((i): ChatItem[] => {
    if (i.key !== key) return [i];
    if (i.text) return [{ ...i, state: 'partial' }];
    return empty === 'drop' ? [] : [{ ...i, state: 'failed' }];
  });

let seq = 0;
const nextKey = () => `l${++seq}`;

/**
 * Drives one conversation: sends a message over SSE, renders deltas as they arrive, shows the 115 card at once, parses
 * the final message (triage, chips), and exposes stop / retry. Everything is reset when `conversationId` changes.
 */
export function useChat(conversationId: number, initial: ChatItem[], initialEmergency: boolean) {
  const client = useQueryClient();
  const [state, setState] = useState<ChatState>(() => ({
    items: initial,
    busy: false,
    failedText: null,
    errorCode: null,
    denial: null,
    needsConsent: false,
    emergency: initialEmergency,
  }));
  const abort = useRef<AbortController | null>(null);
  useEffect(() => () => abort.current?.abort(), []);

  const send = useCallback(
    async (text: string, photoId?: number | null) => {
      const body = text.trim();
      if (!body || abort.current) return;
      const ctl = new AbortController();
      abort.current = ctl;
      const userKey = nextKey();
      const botKey = nextKey();
      const patchBot = (fn: (i: ChatItem) => ChatItem) =>
        setState((s) => ({ ...s, items: s.items.map((i) => (i.key === botKey ? fn(i) : i)) }));
      setState((s) => ({
        ...s,
        busy: true,
        failedText: null,
        errorCode: null,
        denial: null,
        needsConsent: false,
        items: [
          ...s.items.map((i) => ({ ...i, chips: [] })),
          { key: userKey, role: 'user', text: body, card: null, chips: [], triage: null, hasPhoto: !!photoId, state: 'complete' },
          { key: botKey, role: 'assistant', text: '', card: null, chips: [], triage: null, hasPhoto: false, state: 'streaming' },
        ],
      }));
      let gotMeta = false;
      // Before `meta` the server saved nothing: drop the optimistic bubbles (the retry re-adds them).
      const fail = (code: string | null, extra?: Partial<ChatState>) =>
        setState((s) => ({
          ...s,
          busy: false,
          failedText: body,
          errorCode: code,
          items: s.items.filter((i) => i.key !== botKey && (gotMeta || i.key !== userKey)),
          ...extra,
        }));
      try {
        let ended = false;
        for await (const ev of streamMessage(conversationId, { text: body, photoId }, ctl.signal)) {
          const d = (ev.data ?? {}) as Record<string, unknown>;
          if (ev.event === 'meta') gotMeta = true;
          else if (ev.event === 'delta' && typeof d.text === 'string') {
            const chunk = d.text;
            patchBot((i) => ({ ...i, text: i.text + chunk }));
          } else if (ev.event === 'card') {
            const m = messageSchema.safeParse(d.message);
            if (m.success)
              setState((s) => ({
                ...s,
                emergency: true,
                items: [...s.items.filter((i) => i.key !== botKey), fromMessage(m.data)],
              }));
          } else if (ev.event === 'done') {
            const m = messageSchema.safeParse(d.message);
            ended = true;
            setState((s) => ({
              ...s,
              busy: false,
              emergency: s.emergency || d.emergency === true,
              // an emergency answer already put its card in the list; a model answer replaces the streaming bubble
              items:
                m.success && m.data.role === 'assistant'
                  ? s.items.map((i) => (i.key === botKey ? fromMessage(m.data) : i))
                  : s.items.filter((i) => i.key !== botKey),
            }));
          } else if (ev.event === 'error') {
            ended = true;
            const code = typeof d.error_code === 'string' ? d.error_code : null;
            setState((s) => ({
              ...s,
              busy: false,
              failedText: body,
              errorCode: code ?? 'ai_failed',
              items: settle(s.items, botKey),
            }));
          }
        }
        if (!ended) fail('network');
      } catch (e) {
        if (ctl.signal.aborted) {
          // stop button: keep what streamed so far
          setState((s) => ({ ...s, busy: false, items: settle(s.items, botKey) }));
        } else {
          const denial = plusDenialOf(e);
          const code = getApiErrorCode(e);
          if (denial) fail(null, { denial });
          else if (getApiErrorStatus(e) === 403 && code === 'consent_required') fail(null, { needsConsent: true });
          else fail(code ?? 'network');
        }
      } finally {
        abort.current = null;
        void client.invalidateQueries({ queryKey: assistantKeys.conversation(conversationId) });
        void client.invalidateQueries({ queryKey: assistantKeys.hub() });
      }
    },
    [client, conversationId],
  );

  const stop = useCallback(() => abort.current?.abort(), []);
  return { ...state, send, stop };
}
