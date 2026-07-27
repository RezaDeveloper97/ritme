import type { AccentToken, IntroIllustration } from '../model/slides';

interface Props {
  variant: IntroIllustration;
  accentFrom: AccentToken;
  accentTo: AccentToken;
  /** Unique seed so gradient/clip ids don't collide across the three slides. */
  uid: string;
}

/**
 * The intro carousel's per-slide artwork. Deliberately abstract and calm (no
 * literal figures) so it reads as one system and stays non-alarming for a
 * sensitive-health first impression (CLAUDE.md §11). Everything is drawn with
 * theme tokens (`var(--…)`) rather than fixed colours, so light and dark both
 * look right; only the accent gradient changes per slide.
 *
 * Direction-agnostic by construction — pure geometry, no left/right text — so
 * it needs no RTL handling.
 */
export function IntroIllustration({ variant, accentFrom, accentTo, uid }: Props) {
  const grad = `introGrad-${uid}`;
  const soft = `introSoft-${uid}`;

  return (
    <svg
      viewBox="0 0 220 176"
      width="100%"
      role="presentation"
      style={{ display: 'block', maxWidth: 300, height: 'auto' }}
    >
      <defs>
        <linearGradient id={grad} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor={`var(${accentFrom})`} />
          <stop offset="1" stopColor={`var(${accentTo})`} />
        </linearGradient>
        <linearGradient id={soft} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor={`var(${accentTo})`} stopOpacity="0.18" />
          <stop offset="1" stopColor={`var(${accentTo})`} stopOpacity="0" />
        </linearGradient>
      </defs>

      {variant === 'rhythm' && <Rhythm grad={grad} soft={soft} />}
      {variant === 'track' && <Track grad={grad} />}
      {variant === 'insight' && <Insight grad={grad} soft={soft} />}
    </svg>
  );
}

/* — Slide 1: concentric "cycle rhythm" rings with orbiting chips — */
function Rhythm({ grad, soft }: { grad: string; soft: string }) {
  const cx = 110;
  const cy = 88;
  return (
    <g fill="none" strokeLinecap="round">
      <circle cx={cx} cy={cy} r="70" fill={`url(#${soft})`} stroke="none" />
      <circle cx={cx} cy={cy} r="58" stroke="var(--line)" strokeWidth="2" />
      <circle cx={cx} cy={cy} r="42" stroke="var(--field-border)" strokeWidth="2" />
      {/* Highlighted arc = the tracked cycle segment */}
      <circle
        cx={cx}
        cy={cy}
        r="58"
        stroke={`url(#${grad})`}
        strokeWidth="5"
        strokeDasharray="120 300"
        strokeDashoffset="-30"
        transform={`rotate(-90 ${cx} ${cy})`}
      />
      <circle cx={cx} cy={cy} r="22" fill={`url(#${grad})`} stroke="none" />
      {/* Orbiting chips */}
      <Chip x={cx} y={cy - 58} grad={grad}>
        <path d="M0 3.4a3 3 0 0 0-5-2l-.6.6-.6-.6a3 3 0 1 0-4.2 4.2L-5 9l5-5" fill="var(--on-accent)" stroke="none" transform="translate(2.6 -1.6)" />
      </Chip>
      <Chip x={cx - 50} y={cy + 30} grad={grad}>
        <path d="M4 .6A5 5 0 1 1 -1.2 -4 4 4 0 0 0 4 .6z" fill="var(--on-accent)" stroke="none" transform="translate(-1 0)" />
      </Chip>
      <Chip x={cx + 50} y={cy + 30} grad={grad}>
        <path d="M-4.5 1c1 1.4 2.6 2.2 4.5 2.2S3.5 2.4 4.5 1" stroke="var(--on-accent)" strokeWidth="1.6" />
        <circle cx="-2.4" cy="-2.2" r="1" fill="var(--on-accent)" stroke="none" />
        <circle cx="2.4" cy="-2.2" r="1" fill="var(--on-accent)" stroke="none" />
      </Chip>
    </g>
  );
}

function Chip({
  x,
  y,
  grad,
  children,
}: {
  x: number;
  y: number;
  grad: string;
  children: React.ReactNode;
}) {
  return (
    <g transform={`translate(${x} ${y})`}>
      <circle r="14" fill="var(--surface)" stroke="var(--line)" strokeWidth="1.5" />
      <circle r="14" fill={`url(#${grad})`} fillOpacity="0.9" stroke="none" />
      {children}
    </g>
  );
}

/* — Slide 2: a phone card with mini calendar, mood faces, symptom dots — */
function Track({ grad }: { grad: string }) {
  const cols = 5;
  const cell = 22;
  const gx = 40;
  const gy = 44;
  const cells = Array.from({ length: 15 }, (_, i) => i);
  // A few "logged" days (period) and one "today" ring.
  const period = new Set([6, 7, 8]);
  const today = 12;
  return (
    <g>
      <rect x="34" y="18" width="152" height="140" rx="20" fill="var(--surface)" stroke="var(--line)" strokeWidth="2" />
      <rect x="52" y="30" width="52" height="8" rx="4" fill="var(--field-border)" />
      {cells.map((i) => {
        const col = i % cols;
        const row = Math.floor(i / cols);
        const x = gx + col * cell;
        const y = gy + row * cell;
        const isPeriod = period.has(i);
        return (
          <g key={i}>
            <rect
              x={x}
              y={y}
              width="16"
              height="16"
              rx="6"
              fill={isPeriod ? `url(#${grad})` : 'var(--surface-2)'}
            />
            {i === today && (
              <rect x={x - 2} y={y - 2} width="20" height="20" rx="8" fill="none" stroke={`url(#${grad})`} strokeWidth="2" />
            )}
          </g>
        );
      })}
      {/* Mood + symptom row */}
      <g transform="translate(52 126)" fill="none">
        <MoodFace x={0} grad={grad} />
        <MoodFace x={26} grad={grad} />
        <circle cx="66" cy="4" r="5" fill={`url(#${grad})`} />
        <circle cx="82" cy="4" r="5" fill="var(--field-border)" />
        <circle cx="98" cy="4" r="5" fill="var(--field-border)" />
      </g>
    </g>
  );
}

function MoodFace({ x, grad }: { x: number; grad: string }) {
  return (
    <g transform={`translate(${x} 0)`}>
      <circle cx="4" cy="4" r="9" fill="var(--surface)" stroke={`url(#${grad})`} strokeWidth="1.6" />
      <circle cx="1.2" cy="2.4" r="1" fill="var(--ink)" />
      <circle cx="6.8" cy="2.4" r="1" fill="var(--ink)" />
      <path d="M1 6c.8 1 1.8 1.5 3 1.5S6.2 7 7 6" stroke="var(--ink)" strokeWidth="1.4" fill="none" strokeLinecap="round" />
    </g>
  );
}

/* — Slide 3: an insight card — line chart + AI spark chip — */
function Insight({ grad, soft }: { grad: string; soft: string }) {
  return (
    <g>
      <rect x="26" y="26" width="168" height="124" rx="20" fill="var(--surface)" stroke="var(--line)" strokeWidth="2" />
      {/* grid */}
      <g stroke="var(--line)" strokeWidth="1.5">
        <line x1="44" y1="60" x2="176" y2="60" />
        <line x1="44" y1="92" x2="176" y2="92" />
        <line x1="44" y1="124" x2="176" y2="124" />
      </g>
      {/* area + line: a calm wave (mood across the cycle) */}
      <path d="M44 112 C 70 112 78 66 104 66 S 150 118 176 84 L176 124 L44 124 Z" fill={`url(#${soft})`} stroke="none" />
      <path d="M44 112 C 70 112 78 66 104 66 S 150 118 176 84" fill="none" stroke={`url(#${grad})`} strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" />
      <circle cx="104" cy="66" r="4.5" fill="var(--surface)" stroke={`url(#${grad})`} strokeWidth="3" />
      <circle cx="176" cy="84" r="4.5" fill="var(--surface)" stroke={`url(#${grad})`} strokeWidth="3" />
      {/* AI spark chip */}
      <g transform="translate(168 40)">
        <circle r="16" fill={`url(#${grad})`} />
        <path d="M0 -8l2 6 6 2-6 2-2 6-2-6-6-2 6-2z" fill="var(--on-accent)" />
        <path d="M8 4l1 3 3 1-3 1-1 3-1-3-3-1 3-1z" fill="var(--on-accent)" opacity="0.9" />
      </g>
    </g>
  );
}
