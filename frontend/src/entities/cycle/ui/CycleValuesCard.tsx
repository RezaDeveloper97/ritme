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
    <div className="card cvc">
      <div className="cvc-title">{title}</div>

      <div className="cvc-row">
        <span className="cvc-row-l">{loggedLabel}</span>
        <span className="cvc-row-v">{loggedValue}</span>
      </div>

      {suggestion && (
        <div className="cvc-sugg">
          <p className="cvc-sugg-t">
            {suggestion.text}
          </p>
          <button
            type="button"
            onClick={suggestion.onSync}
            className="cvc-sugg-cta"
          >
            {suggestion.ctaLabel}
          </button>
        </div>
      )}

      <div className="cvc-note">{basedOnText}</div>
    </div>
  );
}
