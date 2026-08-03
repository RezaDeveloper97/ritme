/**
 * Spring-physics engine for the bottom-nav "gooey" highlight.
 *
 * The highlight is a gradient blob rendered under an SVG goo (metaball)
 * filter together with an echo circle sitting behind the FAB. Instead of a
 * pre-scripted timeline, the blob is simulated as two independently sprung
 * edges: the edge leading toward the target rides a snappy under-damped
 * spring, the trailing edge a lazy critically-damped one — the gap between
 * them is what reads as gooey stretch, and squash/jiggle emerge from the
 * width over/undershooting rather than being keyframed.
 *
 * Because it is a simulation (not a timeline), navigation mid-flight simply
 * retargets the springs: position and velocity carry over, so rapid tab
 * hopping never jumps or restarts.
 *
 * Choreography (validated against a Blender metaball previz):
 * - crossing the center FAB: the blob stretches into the FAB and is absorbed
 *   (the goo filter fuses it with the echo circle); the FAB inflates ~16%
 *   and leans slightly toward the entry side, holds a beat, then a small
 *   droplet is kicked out the other side with real exit velocity and springs
 *   into the target tab;
 * - moving to the FAB route itself: absorb only, the blob stays swallowed;
 * - leaving the FAB route: emerge only;
 * - same-side moves: a stretchy velocity-continuous slide.
 *
 * All state lives in a module-scope singleton because each screen mounts its
 * own <BottomNav /> — the simulation survives remounts and just keeps going.
 */

export interface Rect {
  left: number;
  width: number;
}

export interface Frame {
  left: number;
  width: number;
  scaleY: number;
  /** Velocity-driven jelly lean, degrees. */
  skewX: number;
  blobOpacity: number;
  fabScale: number;
  /** FAB lean toward the goo while swallowing / spitting, px. */
  fabX: number;
  echoOpacity: number;
}

export type BlobPos = Rect | 'fab';

const clamp = (v: number, a: number, b: number): number =>
  Math.min(b, Math.max(a, v));

const center = (r: Rect): number => r.left + r.width / 2;

/** Where the blob sits while (dis)appearing into the FAB. */
const droplet = (fab: Rect): Rect => ({
  left: fab.left + fab.width / 2 - 9,
  width: 18,
});

/** Pseudo volume preservation: the longer the blob, the flatter it gets. */
const squash = (width: number, restWidth: number): number =>
  clamp(Math.sqrt(restWidth / Math.max(width, 1)), 0.55, 1.12);

/* ── Springs ──
   Parameterised the Apple way: `response` (seconds to reach the target) and
   damping ratio (1 = no overshoot, <1 = bounce). Integrated with
   semi-implicit Euler which is stable at UI stiffnesses. */

interface SpringCfg {
  k: number;
  c: number;
}

const spring = (response: number, zeta: number): SpringCfg => {
  const omega = (2 * Math.PI) / response;
  return { k: omega * omega, c: 2 * zeta * omega };
};

/** Leading edge: fast with a drop of bounce — the part that lunges. */
const LEAD = spring(0.22, 0.66);
/** Trailing edge: slower and overdamped — the part that gets dragged. */
const TRAIL = spring(0.32, 1);
/** FAB inflation: loose and bouncy so absorb/eject reads as a recoil. */
const FAB_SCALE = spring(0.22, 0.5);
/** FAB sideways lean. */
const FAB_X = spring(0.3, 0.6);

interface Ch {
  x: number;
  v: number;
}

const stepCh = (ch: Ch, target: number, cfg: SpringCfg, dt: number): void => {
  ch.v += (cfg.k * (target - ch.x) - cfg.c * ch.v) * dt;
  ch.x += ch.v * dt;
};

const settledCh = (ch: Ch, target: number, eps: number): boolean =>
  Math.abs(ch.x - target) < eps && Math.abs(ch.v) < eps * 12;

const approach = (x: number, target: number, rate: number, dt: number): number =>
  x + (target - x) * (1 - Math.exp(-rate * dt));

/* ── Choreography constants (tuned against the Blender previz) ── */

/** Kept subtle: the highlight circle and the FAB are the same 56px, and a
    visible inflation makes them read as mismatched sizes at the moment of
    contact. Just a whisper of swell sells the absorb without breaking match. */
const FAB_INFLATE = 1.05;
/** Slight anticipation inflate while the blob is still travelling in. */
const FAB_ANTICIPATE = 1.02;
/** Beat during which the goo is fully inside the FAB, ms. */
const ABSORB_HOLD = 40;
/** Exit velocities: the droplet is squeezed out lead-first, px/s. */
const EJECT_LEAD = 2000;
const EJECT_TRAIL = 900;

type Phase = 'free' | 'toFab' | 'absorbed';

class NavGoo {
  initialized = false;

  private edgeL: Ch = { x: 0, v: 0 };
  private edgeR: Ch = { x: 0, v: 0 };
  private fabScale: Ch = { x: 1, v: 0 };
  private fabX: Ch = { x: 0, v: 0 };
  private blobO = 0;
  private echoO = 0;

  private phase: Phase = 'free';
  private target: Rect = { left: 0, width: 0 };
  private pending: Rect | null = null;
  private restWidth = 64;
  private fab: Rect = { left: 0, width: 56 };
  private absorbedAt: number | null = null;
  private lastT: number | null = null;

  snap(pos: BlobPos, fab: Rect): void {
    this.initialized = true;
    this.fab = fab;
    this.pending = null;
    this.absorbedAt = null;
    this.lastT = null;
    this.fabScale = { x: 1, v: 0 };
    this.fabX = { x: 0, v: 0 };
    if (pos === 'fab') {
      this.phase = 'absorbed';
      this.target = droplet(fab);
      this.placeAt(this.target);
      this.blobO = 0;
      this.echoO = 0;
      return;
    }
    this.phase = 'free';
    this.target = pos;
    this.restWidth = pos.width;
    this.placeAt(pos);
    this.blobO = 1;
    this.echoO = 0;
  }

  retarget(pos: BlobPos, fab: Rect): void {
    if (!this.initialized) {
      this.snap(pos, fab);
      return;
    }
    this.fab = fab;
    if (pos === 'fab') {
      this.pending = null;
      if (this.phase !== 'absorbed') {
        this.phase = 'toFab';
        this.target = droplet(fab);
      }
      return;
    }
    this.restWidth = pos.width;
    if (this.phase === 'absorbed') {
      // Emerge (once the hold beat has passed — checked in step()).
      this.pending = pos;
      return;
    }
    const cur = (this.edgeL.x + this.edgeR.x) / 2;
    const crossesFab =
      (cur - center(fab)) * (center(pos) - center(fab)) < 0;
    if (crossesFab) {
      this.phase = 'toFab';
      this.target = droplet(fab);
      this.pending = pos;
    } else {
      // Same side (including bailing out of a suck-in mid-flight): springs
      // just retarget and carry their velocity.
      this.phase = 'free';
      this.target = pos;
      this.pending = null;
    }
  }

  /**
   * Drag mode (iOS 26 liquid-glass tab bar): the blob chases the finger
   * directly. No FAB absorb choreography — continuous retargets while the
   * finger hovers over the "+" would thrash the absorb/eject state machine.
   * The springs still do the work, so the chase feels liquid, not glued.
   */
  follow(pos: Rect, fab: Rect): void {
    if (!this.initialized) {
      this.snap(pos, fab);
      return;
    }
    this.fab = fab;
    this.phase = 'free';
    this.pending = null;
    this.absorbedAt = null;
    this.restWidth = pos.width;
    this.target = pos;
  }

  frame(nowMs: number): Frame {
    const dt = clamp(
      this.lastT === null ? 1 / 60 : (nowMs - this.lastT) / 1000,
      0,
      1 / 30,
    );
    this.lastT = nowMs;

    if (this.phase === 'toFab' && this.nearTarget(10)) {
      this.phase = 'absorbed';
      this.absorbedAt = nowMs;
      // Blender previz: the merged mass bulges toward the entry side.
      const entryDir = this.pending
        ? (center(this.target) > center(this.pending) ? 1 : -1)
        : Math.sign((this.edgeL.v + this.edgeR.v) / 2) || 1;
      this.fabX.v += entryDir * 60;
    }

    if (
      this.phase === 'absorbed' &&
      this.pending &&
      this.absorbedAt !== null &&
      nowMs - this.absorbedAt >= ABSORB_HOLD
    ) {
      const to = this.pending;
      const dir = center(to) >= center(this.fab) ? 1 : -1;
      this.phase = 'free';
      this.target = to;
      this.pending = null;
      this.placeAt(droplet(this.fab));
      this.edgeL.v = dir * (dir > 0 ? EJECT_TRAIL : EJECT_LEAD);
      this.edgeR.v = dir * (dir > 0 ? EJECT_LEAD : EJECT_TRAIL);
      // Recoil: the FAB spits and jiggles back to rest.
      this.fabScale.v -= 1.4;
      this.fabX.v += dir * 90;
    }

    const tL = this.target.left;
    const tR = this.target.left + this.target.width;
    const dir = center(this.target) >= (this.edgeL.x + this.edgeR.x) / 2 ? 1 : -1;
    // The edge facing the target lunges; the other one is dragged.
    stepCh(this.edgeL, tL, dir > 0 ? TRAIL : LEAD, dt);
    stepCh(this.edgeR, tR, dir > 0 ? LEAD : TRAIL, dt);

    const inHold =
      this.absorbedAt !== null && nowMs - this.absorbedAt < ABSORB_HOLD;
    const fabScaleTarget =
      this.phase === 'absorbed'
        ? this.pending || inHold
          ? FAB_INFLATE
          : 1
        : this.phase === 'toFab'
          ? FAB_ANTICIPATE
          : 1;
    stepCh(this.fabScale, fabScaleTarget, FAB_SCALE, dt);
    stepCh(this.fabX, 0, FAB_X, dt);

    this.blobO = approach(this.blobO, this.phase === 'absorbed' ? 0 : 1, 26, dt);
    this.echoO = approach(this.echoO, this.phase === 'free' ? 0 : 1, 9, dt);

    const left = Math.min(this.edgeL.x, this.edgeR.x);
    const width = Math.max(12, Math.abs(this.edgeR.x - this.edgeL.x));
    const vMid = (this.edgeL.v + this.edgeR.v) / 2;
    return {
      left,
      width,
      scaleY: squash(width, this.restWidth),
      skewX: clamp(-vMid * 0.004, -6, 6),
      blobOpacity: this.blobO,
      fabScale: this.fabScale.x,
      fabX: this.fabX.x,
      echoOpacity: this.echoO,
    };
  }

  settled(): boolean {
    if (this.phase !== 'free' && this.pending) return false;
    if (this.phase === 'toFab') return false;
    const oT = this.phase === 'absorbed' ? 0 : 1;
    return (
      settledCh(this.edgeL, this.target.left, 0.5) &&
      settledCh(this.edgeR, this.target.left + this.target.width, 0.5) &&
      settledCh(this.fabScale, this.phase === 'absorbed' && this.pending ? FAB_INFLATE : 1, 0.005) &&
      settledCh(this.fabX, 0, 0.1) &&
      Math.abs(this.blobO - oT) < 0.02 &&
      this.echoO < 0.02
    );
  }

  private placeAt(r: Rect): void {
    this.edgeL = { x: r.left, v: 0 };
    this.edgeR = { x: r.left + r.width, v: 0 };
  }

  private nearTarget(eps: number): boolean {
    return (
      Math.abs(this.edgeL.x - this.target.left) < eps &&
      Math.abs(this.edgeR.x - (this.target.left + this.target.width)) < eps
    );
  }
}

const instance = new NavGoo();

/** The one simulation shared by every <BottomNav /> mount. */
export const getNavGoo = (): NavGoo => instance;

/** Final resting frame for reduced-motion / first paint / resize. */
export function restFrame(pos: BlobPos, fab: Rect): Frame {
  if (pos === 'fab') {
    return {
      ...droplet(fab),
      scaleY: 1,
      skewX: 0,
      blobOpacity: 0,
      fabScale: 1,
      fabX: 0,
      echoOpacity: 0,
    };
  }
  return {
    ...pos,
    scaleY: 1,
    skewX: 0,
    blobOpacity: 1,
    fabScale: 1,
    fabX: 0,
    echoOpacity: 0,
  };
}
