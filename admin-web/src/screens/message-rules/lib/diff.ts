/** Word-level text diff (LCS) for the approved ↔ draft comparison. */
export interface DiffPart {
  type: 'same' | 'add' | 'del';
  text: string;
}

export function diffWords(before: string, after: string): DiffPart[] {
  const a = before.split(/\s+/).filter(Boolean);
  const b = after.split(/\s+/).filter(Boolean);
  const dp: number[][] = Array.from({ length: a.length + 1 }, () => new Array<number>(b.length + 1).fill(0));
  for (let i = a.length - 1; i >= 0; i--) {
    for (let j = b.length - 1; j >= 0; j--) {
      dp[i]![j] = a[i] === b[j] ? dp[i + 1]![j + 1]! + 1 : Math.max(dp[i + 1]![j]!, dp[i]![j + 1]!);
    }
  }
  const out: DiffPart[] = [];
  const push = (type: DiffPart['type'], text: string) => {
    const last = out[out.length - 1];
    if (last && last.type === type) last.text += ` ${text}`;
    else out.push({ type, text });
  };
  let i = 0;
  let j = 0;
  while (i < a.length && j < b.length) {
    if (a[i] === b[j]) {
      push('same', a[i]!);
      i++;
      j++;
    } else if (dp[i + 1]![j]! >= dp[i]![j + 1]!) push('del', a[i++]!);
    else push('add', b[j++]!);
  }
  while (i < a.length) push('del', a[i++]!);
  while (j < b.length) push('add', b[j++]!);
  return out;
}

export interface TextContent {
  a?: { title?: string; body?: string; cta_label?: string };
  b?: { title?: string; body?: string; cta_label?: string } | null;
  variants?: Record<string, { title?: string; body?: string; cta_label?: string }>;
  seen?: string;
  confidence?: string;
  contact?: string;
}

/** Every user-facing text of a content block as `label → text`. */
export function contentTexts(c: TextContent | undefined): Record<string, string> {
  const out: Record<string, string> = {};
  if (!c) return out;
  const v = (k: string, x?: { title?: string; body?: string; cta_label?: string } | null) => {
    if (!x) return;
    out[`${k}.title`] = x.title ?? '';
    out[`${k}.body`] = x.body ?? '';
    if (x.cta_label) out[`${k}.cta_label`] = x.cta_label;
  };
  v('a', c.a);
  v('b', c.b);
  for (const [k, x] of Object.entries(c.variants ?? {})) v(k, x);
  if (c.seen) out.seen = c.seen;
  if (c.confidence) out.confidence = c.confidence;
  if (c.contact) out.contact = c.contact;
  return out;
}

/** The fields whose text differs between the approved and the draft content. */
export function contentDiff(approved: TextContent | undefined, draft: TextContent | undefined): { label: string; parts: DiffPart[] }[] {
  const x = contentTexts(approved);
  const y = contentTexts(draft);
  return Array.from(new Set([...Object.keys(x), ...Object.keys(y)]))
    .filter((k) => (x[k] ?? '') !== (y[k] ?? ''))
    .map((label) => ({ label, parts: diffWords(x[label] ?? '', y[label] ?? '') }));
}
