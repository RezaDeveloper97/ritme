import type { LocaleTips, TipsBody } from '../api/companions';

/** One editable tip (a fixed slot on the API; empty slots are not shown in the form). */
export interface TipDraft {
  title: string;
  body: string;
}

/** The editable copy of one phase in one language. */
export interface LocaleDraft {
  note: string;
  tips: TipDraft[];
}

/** The form copy of a language: the text the panel shows today, empty tip slots dropped. */
export function draftOf(texts: LocaleTips | undefined): LocaleDraft {
  if (!texts) return { note: '', tips: [] };
  return {
    note: texts.note.body,
    tips: texts.tips.filter((s) => s.title.trim() !== '').map((s) => ({ title: s.title, body: s.body })),
  };
}

export function sameDraft(a: LocaleDraft, b: LocaleDraft): boolean {
  return (
    a.note.trim() === b.note.trim() &&
    a.tips.length === b.tips.length &&
    a.tips.every((t, i) => t.title.trim() === b.tips[i]?.title.trim() && t.body.trim() === b.tips[i]?.body.trim())
  );
}

/** PUT body for the given languages (blank tips are dropped: an empty title would hide the slot anyway). */
export function tipsBody(drafts: Record<string, LocaleDraft>, codes: readonly string[]): TipsBody {
  const texts: TipsBody['texts'] = {};
  for (const code of codes) {
    const d = drafts[code];
    if (!d) continue;
    texts[code] = {
      note: d.note.trim() ? d.note.trim() : null,
      tips: d.tips
        .filter((t) => t.title.trim() !== '' || t.body.trim() !== '')
        .map((t) => ({ title: t.title.trim(), body: t.body.trim() ? t.body.trim() : null })),
    };
  }
  return { texts };
}

/** `{name}` → the sample partner name of the preview. */
export function fillName(text: string, name: string): string {
  return text.split('{name}').join(name);
}

/** Moves item `i` by `delta` (−1 up, +1 down); out-of-range moves return the list unchanged. */
export function move<T>(items: readonly T[], i: number, delta: number): T[] {
  const j = i + delta;
  if (i < 0 || i >= items.length || j < 0 || j >= items.length) return [...items];
  const out = [...items];
  const [it] = out.splice(i, 1);
  out.splice(j, 0, it as T);
  return out;
}
