'use client';

/**
 * A non-destructive suggestion to sync the profile with what recent cycles show
 * (spec §12): the app never changes the profile silently, it only proposes.
 */
export interface CycleValuesSuggestion {
  /** e.g. "Recent logs suggest ~30 days". */
  text: string;
  /** e.g. "Update to 30 days". */
  ctaLabel: string;
  onSync: () => void;
}

export interface CycleValuesCardProps {
  title: string;
  loggedLabel: string;
  /** Pre-formatted (localized, pluralized) logged cycle length, e.g. "28 days". */
  loggedValue: string;
  /** Present only when recent data meaningfully differs from the profile. */
  suggestion: CycleValuesSuggestion | null;
  /** e.g. "Current prediction is based on: your recent logs". */
  basedOnText: string;
}

/**
 * Surfaces the spec §12 three layers to the user: the value they logged in their
 * profile, an optional "recent logs suggest …" nudge, and which layer the current
 * prediction actually uses. Presentational and locale-agnostic — the host screen
 * formats every string and owns the sync action (FSD: entities never call features).
 */
export function CycleValuesCard({
  title,
  loggedLabel,
  loggedValue,
  suggestion,
  basedOnText,
}: CycleValuesCardProps) {
  return (
    <div className="card" style={{ margin: '0 16px 14px', padding: '16px 14px', textAlign: 'start' }}>
      <div style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)', marginBottom: 12 }}>{title}</div>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 12 }}>
        <span style={{ fontSize: 13, color: 'var(--muted)' }}>{loggedLabel}</span>
        <span style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)' }}>{loggedValue}</span>
      </div>

      {suggestion && (
        <div style={{ marginTop: 12, background: 'var(--line)', borderRadius: 12, padding: 12 }}>
          <p style={{ fontSize: 12.5, color: 'var(--steel)', margin: '0 0 10px', lineHeight: 1.7 }}>
            {suggestion.text}
          </p>
          <button
            type="button"
            onClick={suggestion.onSync}
            style={{
              border: 'none',
              cursor: 'pointer',
              background: 'var(--brand)',
              color: 'var(--on-accent)',
              fontSize: 13,
              fontWeight: 700,
              borderRadius: 10,
              padding: '9px 14px',
            }}
          >
            {suggestion.ctaLabel}
          </button>
        </div>
      )}

      <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 12 }}>{basedOnText}</div>
    </div>
  );
}
