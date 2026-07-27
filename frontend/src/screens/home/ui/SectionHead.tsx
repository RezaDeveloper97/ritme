'use client';

/**
 * The title row every home card shares: section name on the start side, an
 * optional action on the end side. Rendered with logical alignment so it flips
 * correctly in both directions (§12).
 */
export function SectionHead({
  title,
  action,
  onAction,
}: {
  title: string;
  action?: string;
  onAction?: () => void;
}) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', margin: '0 2px 12px' }}>
      <span style={{ fontSize: 16, fontWeight: 800, color: 'var(--ink)' }}>{title}</span>
      {action ? (
        <button
          type="button"
          onClick={onAction}
          style={{
            background: 'none',
            border: 0,
            padding: 0,
            fontSize: 12,
            fontWeight: 700,
            color: 'var(--brand)',
            cursor: onAction ? 'pointer' : 'default',
          }}
        >
          {action}
        </button>
      ) : (
        <span />
      )}
    </div>
  );
}
