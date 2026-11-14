/**
 * The tick inside a checkbox box — the one tick for every check control in the
 * app. A plain line SVG on purpose: `Icon name="check"` resolves to the 3D art
 * image, which ignores `color`, so an unchecked box still showed it. Always
 * rendered; `on` draws the stroke in (`.nb-tick.is-on`, pathLength=1) and its
 * removal retracts it, so the box is empty when off and both directions animate.
 */
export function CheckTick({ on, size = 14 }: { on: boolean; size?: number }) {
  return (
    <svg
      className={on ? 'nb-tick is-on' : 'nb-tick'}
      viewBox="0 0 24 24"
      width={size}
      height={size}
      fill="none"
      stroke="currentColor"
      strokeWidth={3.2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      <path d="M5 12.5l4.5 4.5L19 7.5" pathLength={1} />
    </svg>
  );
}
