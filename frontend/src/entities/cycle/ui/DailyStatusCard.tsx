'use client';

import type { DailyCard, FertilityLevel } from '../model/types';

/**
 * Brand-token colour per calendar fertility level (spec §17 read-out). Purely a
 * presentation choice; the level itself is informational, never diagnostic (§11).
 * Foregrounds are darkened to clear WCAG AA (≥4.5:1) on their pale backgrounds.
 */
const FERTILITY_COLOR: Record<FertilityLevel, { bg: string; fg: string }> = {
  very_high: { bg: '#FFE3F0', fg: '#A6005A' },
  high: { bg: '#FFF1F7', fg: '#B4004E' },
  medium: { bg: '#FFF4E5', fg: '#8A5200' },
  low: { bg: 'var(--line, #EEF1F4)', fg: '#47555F' },
  very_low: { bg: 'var(--line, #EEF1F4)', fg: '#5B6B7B' },
};

/** Neutral style for a fertility level the backend added that this build doesn't know. */
const FERTILITY_FALLBACK = FERTILITY_COLOR.low;

export interface DailyStatusCardProps {
  card: DailyCard;
  /** Fired with a call-to-action's stable `type` code when the user taps it. */
  onAction?: (type: string) => void;
  /** Hide the CTA row when the host screen already provides its own action buttons. */
  showActions?: boolean;
  /** Disable the CTAs while a triggered action is in flight (prevents double-taps). */
  pending?: boolean;
  className?: string;
}

/**
 * Renders the backend's render-ready day-status card (spec §19): title, subtitle,
 * fertility read-out, badges and calls to action. Every string is already localized
 * by the backend (via `Accept-Language`), so this component only lays it out —
 * RTL-safe with logical alignment — and forwards action taps to the parent, which
 * owns the actual log-period / reminder features (FSD: entities never call features).
 */
export function DailyStatusCard({ card, onAction, showActions = true, pending = false, className }: DailyStatusCardProps) {
  const { primaryAction, secondaryActions } = card;
  // Fall back to a neutral style for an unknown level — never crash the day sheet
  // on a fertility value a newer backend added (the boundary parser stays lenient).
  const fertility = FERTILITY_COLOR[card.fertilityLevel] ?? FERTILITY_FALLBACK;

  return (
    <div className={className ?? 'card'} style={{ padding: '16px 14px', textAlign: 'start' }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 12 }}>
        <h3 style={{ fontSize: 16, fontWeight: 800, color: 'var(--ink)', margin: 0, lineHeight: 1.5 }}>
          {card.title}
        </h3>
        {card.fertilityLabel && (
          <span
            style={{
              flexShrink: 0,
              fontSize: 11,
              fontWeight: 700,
              borderRadius: 20,
              padding: '4px 10px',
              background: fertility.bg,
              color: fertility.fg,
            }}
          >
            {card.fertilityLabel}
          </span>
        )}
      </div>

      {card.subtitle && (
        <p style={{ fontSize: 13, color: 'var(--muted)', margin: '8px 0 0', lineHeight: 1.7 }}>
          {card.subtitle}
        </p>
      )}

      {card.badges.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginTop: 12 }}>
          {card.badges.map((badge) => (
            <span
              key={badge}
              style={{
                fontSize: 11,
                fontWeight: 600,
                color: 'var(--steel)',
                background: 'var(--line)',
                borderRadius: 8,
                padding: '4px 10px',
              }}
            >
              {badge}
            </span>
          ))}
        </div>
      )}

      {showActions && (primaryAction || secondaryActions.length > 0) && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10, marginTop: 16 }}>
          {primaryAction && (
            <button
              type="button"
              onClick={() => onAction?.(primaryAction.type)}
              disabled={pending}
              style={{
                flex: '1 1 auto',
                minWidth: 140,
                border: 'none',
                cursor: pending ? 'default' : 'pointer',
                opacity: pending ? 0.6 : 1,
                background: 'var(--brand, #E91E63)',
                color: '#fff',
                fontSize: 13.5,
                fontWeight: 700,
                borderRadius: 12,
                padding: '11px 14px',
              }}
            >
              {primaryAction.label}
            </button>
          )}
          {secondaryActions.map((action) => (
            <button
              key={action.type}
              type="button"
              onClick={() => onAction?.(action.type)}
              disabled={pending}
              style={{
                flex: '0 1 auto',
                cursor: pending ? 'default' : 'pointer',
                opacity: pending ? 0.6 : 1,
                background: 'transparent',
                color: 'var(--steel)',
                fontSize: 13,
                fontWeight: 700,
                borderRadius: 12,
                padding: '10px 14px',
                border: '1px solid var(--line)',
              }}
            >
              {action.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
