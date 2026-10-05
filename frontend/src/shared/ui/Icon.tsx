import type { CSSProperties } from 'react';

export type IconName =
  | 'bell' | 'sparkle' | 'chevronRight' | 'chevronLeft' | 'chevronDown'
  | 'pencil' | 'check' | 'checkCircle' | 'loader' | 'drop'
  | 'calendar' | 'plus' | 'heart' | 'moon' | 'zap' | 'smile'
  | 'book' | 'alarm' | 'pill' | 'flame' | 'info' | 'x'
  | 'user' | 'chart' | 'grid' | 'arrowL' | 'refresh'
  | 'walk' | 'thermo' | 'glass' | 'stetho'
  | 'home' | 'bookOpen' | 'apple' | 'brain'
  | 'globe' | 'shield' | 'logout' | 'download' | 'trash' | 'search'
  | 'sun' | 'contrast' | 'cog' | 'bellPlain' | 'pen'
  | 'capsule' | 'tablet' | 'video' | 'phone' | 'mapPin' | 'clock' | 'bellRing' | 'note'
  | 'ribbon' | 'flask' | 'tooth' | 'camera' | 'history' | 'filterLines' | 'export'
  | 'flaskLh' | 'heartLine' | 'target' | 'moonReminder'
  | 'hand' | 'eye' | 'scale' | 'bookmark' | 'warning' | 'doctor'
  | 'faceGreat' | 'faceGood' | 'faceOkay' | 'faceLow' | 'faceHard'
  | 'lock' | 'minus' | 'arrowR'
  | 'users' | 'gradCap' | 'todo' | 'box' | 'watch' | 'help' | 'chat' | 'crown' | 'smartphone' | 'modeRing'
  | 'symptom' | 'star' | 'sprout'
  | 'female' | 'male' | 'cake'
  | 'card'
  | 'syringe' | 'implant'
  | 'mic' | 'gut' | 'dropLine' | 'bed' | 'urine' | 'run' | 'scaleSquare'
  | 'grip' | 'pin' | 'chevronUp'
  | 'copy' | 'share' | 'send'
  | 'bottle' | 'sleep' | 'mother'
  | 'ruler';

const PATHS: Record<IconName, string> = {
  bell:         '<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>',
  sparkle:      '<path d="M12 3l1.6 4.6L18 9.2l-4.4 1.6L12 15l-1.6-4.2L6 9.2l4.4-1.6z"/><path d="M19 13l.7 2 .3 .5 2 .7-2 .7-.3 .5-.7 2-.7-2-.3-.5-2-.7 2-.7 .3-.5z"/>',
  chevronRight: '<path d="M9 18l6-6-6-6"/>',
  chevronLeft:  '<path d="M15 18l-6-6 6-6"/>',
  chevronDown:  '<path d="M6 9l6 6 6-6"/>',
  pencil:       '<path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4z"/>',
  check:        '<path d="M20 6L9 17l-5-5"/>',
  checkCircle:  '<circle cx="12" cy="12" r="9"/><path d="M9 12l2 2 4-4"/>',
  loader:       '<path d="M12 3v3M12 18v3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M3 12h3M18 12h3M5.6 18.4l2.1-2.1M16.3 7.7l2.1-2.1"/>',
  drop:         '<path d="M12 2.7s6 6.2 6 10.3a6 6 0 0 1-12 0c0-4.1 6-10.3 6-10.3z"/>',
  calendar:     '<rect x="3" y="4.5" width="18" height="17" rx="3"/><path d="M3 9h18M8 2.5v4M16 2.5v4"/>',
  plus:         '<path d="M12 5v14M5 12h14"/>',
  heart:        '<path d="M19 5.6a4.6 4.6 0 0 0-7-.6l-1 1-1-1a4.6 4.6 0 1 0-6.5 6.5L12 20l8.5-8.5A4.6 4.6 0 0 0 19 5.6z"/>',
  moon:         '<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>',
  zap:          '<path d="M13 2L4.5 13.5H11l-1 8.5L19.5 10H13z"/>',
  smile:        '<circle cx="12" cy="12" r="9"/><path d="M8 14s1.5 2 4 2 4-2 4-2M9 9h.01M15 9h.01"/>',
  book:         '<path d="M3 5.5A2.5 2.5 0 0 1 5.5 3H20v15H5.5A2.5 2.5 0 0 0 3 20.5z"/><path d="M3 5.5V20.5"/>',
  bookOpen:     '<path d="M12 7v14"/><path d="M3 18a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h5a4 4 0 0 1 4 4 4 4 0 0 1 4-4h5a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1h-6a3 3 0 0 0-3 3 3 3 0 0 0-3-3z"/>',
  home:         '<path d="M3 10.7 12 3l9 7.7"/><path d="M5.2 9.5V19a1.5 1.5 0 0 0 1.5 1.5h10.6A1.5 1.5 0 0 0 18.8 19V9.5"/>',
  alarm:        '<circle cx="12" cy="13" r="8"/><path d="M12 9v4l2.5 1.5M5 3L2 6M19 3l3 3"/>',
  pill:         '<path d="M10.5 20.5a5 5 0 0 1-7-7l6-6a5 5 0 0 1 7 7z"/><path d="M8.5 8.5l7 7"/>',
  stetho:       '<path d="M4.5 3v5a4.5 4.5 0 0 0 9 0V3"/><path d="M4.5 3h-1M13.5 3h1M9 17v1a4 4 0 0 0 8 0v-2"/><circle cx="18" cy="14" r="2.4"/>',
  flame:        '<path d="M12 2s4 4 4 8a4 4 0 0 1-8 0c0-1 .4-2 1-2.6C9 8 12 6 12 2z"/><path d="M12 22a6 6 0 0 0 6-6c0-2-1-3.5-2-4.5.2 2.5-1.6 3.8-2.6 4.2.4-1.8-.4-3.7-1.4-4.7-.2 3-3 3.6-3 6.2A3.8 3.8 0 0 0 12 22z"/>',
  info:         '<circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8h.01"/>',
  x:            '<path d="M18 6L6 18M6 6l12 12"/>',
  glass:        '<path d="M6 3h12l-1.4 16a2 2 0 0 1-2 1.8H9.4a2 2 0 0 1-2-1.8z"/><path d="M6.7 9h10.6"/>',
  walk:         '<circle cx="13" cy="4.5" r="1.6"/><path d="M9 21l2.5-6L9.5 12l-1 4M11.5 15l3 1.5 1.5 3.5M11.5 9l3 1 1.5 3"/>',
  thermo:       '<path d="M14 14.8V5a2 2 0 0 0-4 0v9.8a4 4 0 1 0 4 0z"/>',
  user:         '<circle cx="12" cy="8" r="4"/><path d="M4 21c0-4.4 3.6-7 8-7s8 2.6 8 7"/>',
  chart:        '<path d="M3 3v18h18"/><path d="M7 14l3-4 3 3 4-6"/>',
  grid:         '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>',
  arrowL:       '<path d="M19 12H5M12 19l-7-7 7-7"/>',
  refresh:      '<path d="M3 12a9 9 0 0 1 15-6.7L21 8M21 3v5h-5M21 12a9 9 0 0 1-15 6.7L3 16M3 21v-5h5"/>',
  globe:        '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 2.5 3.8 5.6 3.8 9S14.5 18.5 12 21c-2.5-2.5-3.8-5.6-3.8-9S9.5 5.5 12 3z"/>',
  shield:       '<path d="M12 3l7 3v5c0 4.4-3 8.2-7 10-4-1.8-7-5.6-7-10V6z"/><path d="M9.2 12l2 2 3.6-4"/>',
  logout:       '<path d="M15 3h3a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-3"/><path d="M10 17l5-5-5-5M15 12H3"/>',
  download:     '<path d="M12 3v12M7 10l5 5 5-5"/><path d="M4 20h16"/>',
  apple:        '<path d="M12 8c-1.2-1-2.6-1.4-4-1-2 .6-3 2.7-3 5.2 0 3.6 2.3 8 4.6 8 .9 0 1.6-.4 2.4-.4s1.5.4 2.4.4c2.3 0 4.6-4.4 4.6-8 0-2.5-1-4.6-3-5.2-1.4-.4-2.8 0-4 1z"/><path d="M12 8c0-2.2 1.3-4 3.5-4.5"/>',
  brain:        '<path d="M9.5 4a2.5 2.5 0 0 0-2.4 3.2A2.6 2.6 0 0 0 5 9.8c0 .9.4 1.7 1.1 2.2A2.6 2.6 0 0 0 5.4 14c0 1.2.8 2.2 2 2.5A2.5 2.5 0 0 0 12 18V5.9A2 2 0 0 0 9.5 4z"/><path d="M14.5 4a2.5 2.5 0 0 1 2.4 3.2A2.6 2.6 0 0 1 19 9.8c0 .9-.4 1.7-1.1 2.2.4.5.7 1.2.7 2 0 1.2-.8 2.2-2 2.5A2.5 2.5 0 0 1 12 18"/>',
  trash:        '<path d="M4 7h16M9 7V5a1.5 1.5 0 0 1 1.5-1.5h3A1.5 1.5 0 0 1 15 5v2M6 7l1 12.5A1.5 1.5 0 0 0 8.5 21h7a1.5 1.5 0 0 0 1.5-1.5L18 7"/>',
  search:       '<circle cx="11" cy="11" r="7"/><path d="M20 20l-3.6-3.6"/>',
  sun:          '<circle cx="12" cy="12" r="4.2"/><path d="M12 2v2.4M12 19.6V22M4.2 4.2l1.7 1.7M18.1 18.1l1.7 1.7M2 12h2.4M19.6 12H22M4.2 19.8l1.7-1.7M18.1 5.9l1.7-1.7"/>',
  cog:          '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>',
  bellPlain:    '<path d="M6 16V11a6 6 0 0112 0v5l2 2H4z"/><path d="M10 20a2 2 0 004 0"/>',
  pen:          '<path d="M4 20l4-1 10-10-3-3L5 16z"/>',
  /* Care reminders (M3) — copied from the v13 artboards (docs/design/reminders-v13). */
  capsule:      '<rect x="4" y="9" width="16" height="6" rx="3"/><path d="M12 9v6"/>',
  tablet:       '<rect x="3" y="9" width="18" height="7" rx="3.5" transform="rotate(-35 12 12)"/><path d="M9 7l6 10"/>',
  video:        '<rect x="3" y="6" width="13" height="12" rx="2"/><path d="M16 10l5-3v10l-5-3z"/>',
  phone:        '<path d="M5 4h4l2 5-2.5 1.5a11 11 0 005 5L15 13l5 2v4a2 2 0 01-2 2A16 16 0 013 6a2 2 0 012-2z"/>',
  mapPin:       '<path d="M12 21s-6-5.5-6-11a6 6 0 0112 0c0 5.5-6 11-6 11z"/><circle cx="12" cy="10" r="2"/>',
  clock:        '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  bellRing:     '<path d="M6 16V11a6 6 0 0112 0v5l2 2H4z"/><path d="M10 20a2 2 0 004 0"/><path d="M4 7a8 8 0 012-3M20 7a8 8 0 00-2-3"/>',
  note:         '<path d="M7 3h7l5 5v13H7z"/><path d="M14 3v5h5M10 13h6M10 17h6"/>',
  /* Checkups (M4) — copied from the v14 artboards (docs/design/checkups-v14). The
     artboards' shield-check is `shield` and their file/PDF glyph is `note`. */
  ribbon:       '<path d="M12 3a5 5 0 015 5c0 3-5 6-5 6s-5-3-5-6a5 5 0 015-5z"/><path d="M9 12l-3 9 6-3 6 3-3-9"/>',
  flask:        '<path d="M9 3h6M10 3v6l-5 9a2 2 0 002 3h10a2 2 0 002-3l-5-9V3"/><path d="M8 16h8"/>',
  tooth:        '<path d="M7 3c2 0 3 1 5 1s3-1 5-1c3 0 4 3 3 6-1 2-1 5-2 9-1 3-2 3-3 0l-1-4h-4l-1 4c-1 3-2 3-3 0-1-4-1-7-2-9-1-3 0-6 3-6z"/>',
  camera:       '<path d="M4 8h3l2-3h6l2 3h3v11H4z"/><circle cx="12" cy="13" r="3"/>',
  history:      '<path d="M3 12a9 9 0 109-9 9 9 0 00-7 3.5"/><path d="M3 3v5h5M12 7v5l3 2"/>',
  filterLines:  '<path d="M4 6h16M7 12h10M10 18h4"/>',
  export:       '<path d="M12 3v12M7 8l5-5 5 5M5 14v5a2 2 0 002 2h10a2 2 0 002-2v-5"/>',
  /* Fertility / TTC (M5) — copied from the v19 artboards (docs/design/ttc-v19).
     The BBT tile/chart glyph is the existing `thermo`. `heartLine` / `moonReminder`
     are the artboards' slimmer heart and crescent — `heart` / `moon` stay as
     they are for the screens already using them. */
  flaskLh:      '<path d="M9 3h6M10 3v7l-5 8a2 2 0 002 3h10a2 2 0 002-3l-5-8V3"/>',
  heartLine:    '<path d="M12 20s-7-4.5-7-10a4 4 0 017-2.5A4 4 0 0119 10c0 5.5-7 10-7 10z"/>',
  target:       '<circle cx="12" cy="12" r="8"/><circle cx="12" cy="12" r="3"/>',
  // B-N1-06 (nbl_Cycle_Home): symptoms disc (wave in a circle) and the challenges star.
  symptom:      '<circle cx="12" cy="12" r="8"/><path d="M8 12c1.3-2 2.7-2 4 0s2.7 2 4 0"/>',
  star:         '<path d="M12 3.5l2.6 5.3 5.9.9-4.3 4.1 1 5.8L12 16.9l-5.2 2.7 1-5.8-4.3-4.1 5.9-.9z"/>',
  moonReminder: '<path d="M20 14.5A8 8 0 019.5 4 8 8 0 1020 14.5z"/>',
  /* Pregnancy v2 (M7) — copied from the artboards (docs/design/pregnancy-v2).
     Reused rather than re-added: heart → `heartLine`, water → `drop`, bell →
     `bellPlain`, directions → `mapPin`, edit → `pencil`, reviewer avatar →
     `user`. `doctor` is the artboards' stethoscope (weekly checkup); the older
     `stetho` stays for the screens already using it. The five faces are the
     Log mood scale, best (`faceGreat`, mood 5) to hardest (`faceHard`, mood 1). */
  hand:         '<path d="M8 13V5a1.5 1.5 0 013 0v6M11 11V4a1.5 1.5 0 013 0v7M14 11V6a1.5 1.5 0 013 0v7a6 6 0 01-6 6h-1a5 5 0 01-4-2l-3-4a1.5 1.5 0 012.3-2L8 13"/>',
  eye:          '<path d="M4 12c2-4 5-6 8-6s6 2 8 6c-2 4-5 6-8 6s-6-2-8-6z"/><circle cx="12" cy="12" r="2.5"/>',
  scale:        '<path d="M4 8h16l-1.5 12h-13z"/><path d="M8 8a4 4 0 018 0"/>',
  bookmark:     '<path d="M6 3h12v18l-6-4-6 4z"/>',
  warning:      '<path d="M12 3l9 16H3z"/><path d="M12 10v4M12 17v.01"/>',
  doctor:       '<path d="M6 3v6a6 6 0 0012 0V3"/><path d="M12 15v3a3 3 0 006 0v-3"/><circle cx="18" cy="12" r="2"/>',
  faceGreat:    '<circle cx="12" cy="12" r="9"/><path d="M9 10h.01M15 10h.01"/><path d="M8 14c1.5 2.5 6.5 2.5 8 0"/>',
  faceGood:     '<circle cx="12" cy="12" r="9"/><path d="M9 10h.01M15 10h.01"/><path d="M9 15c1 1 5 1 6 0"/>',
  faceOkay:     '<circle cx="12" cy="12" r="9"/><path d="M9 10h.01M15 10h.01"/><path d="M9 15h6"/>',
  faceLow:      '<circle cx="12" cy="12" r="9"/><path d="M9 10h.01M15 10h.01"/><path d="M9 16c1-1 5-1 6 0"/>',
  faceHard:     '<circle cx="12" cy="12" r="9"/><path d="M9 10h.01M15 10h.01"/><path d="M8 17c1.5-2.5 6.5-2.5 8 0"/>',
  /* Half-filled disc — the conventional "match the system appearance" mark. */
  contrast:     '<circle cx="12" cy="12" r="9"/><path d="M12 3a9 9 0 0 1 0 18z" fill="currentColor" stroke="none"/>',
  /* Night & Bloom primitives (B-N1-03): Plus lock pill, stepper minus. */
  lock:         '<rect x="4.5" y="10.5" width="15" height="10.5" rx="2.5"/><path d="M8 10.5V7.5a4 4 0 0 1 8 0v3"/>',
  minus:        '<path d="M5 12h14"/>',
  arrowR:       '<path d="M5 12h14M12 5l7 7-7 7"/>',
  /* Me hub / account (B-N1-10) — copied from nbl_Me_Hub / nbl_Me_Profile. */
  users:        '<circle cx="9" cy="8" r="3"/><circle cx="17" cy="10" r="2.2"/><path d="M3 20c.8-3.5 3.2-5.5 6-5.5s5.2 2 6 5.5M15 15.5c2.6-.3 4.6 1.3 5.3 4.5"/>',
  gradCap:      '<path d="M3 9l9-5 9 5-9 5z"/><path d="M7 11v5c3 2 7 2 10 0v-5"/>',
  todo:         '<rect x="4" y="4" width="16" height="16" rx="4"/><path d="M8 12l3 3 5-6"/>',
  box:          '<path d="M3 7l9-4 9 4v10l-9 4-9-4z"/><path d="M3 7l9 4 9-4M12 11v10"/>',
  watch:        '<rect x="7" y="6" width="10" height="12" rx="3"/><path d="M9 6l1-3h4l1 3M9 18l1 3h4l1-3"/>',
  help:         '<circle cx="12" cy="12" r="9"/><path d="M9.5 9.5a2.5 2.5 0 015 .5c0 2-2.5 2-2.5 4M12 17h.01"/>',
  chat:         '<path d="M4 5h16v11H9l-5 4z"/>',
  crown:        '<path d="M3 8l4 4 5-7 5 7 4-4-2 11H5z"/>',
  smartphone:   '<rect x="7" y="3" width="10" height="18" rx="2.5"/><path d="M11 18h2"/>',
  modeRing:     '<circle cx="12" cy="12" r="8"/><circle cx="12" cy="12" r="3"/>',
  // B-N2-02: onboarding gender cards + birthday / due-date rows (nbl_Onb_Gender, _Health, _Preg)
  female:       '<circle cx="12" cy="9" r="5"/><path d="M12 14v7M9 18h6"/>',
  male:         '<circle cx="10" cy="14" r="5"/><path d="M13.5 10.5L20 4M15 4h5v5"/>',
  cake:         '<path d="M4 20h16v-6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2z"/><path d="M4 16c1.5 1 3 1 4 0s2.5-1 4 0 2.5 1 4 0 2.5-1 4 0"/><path d="M8 12V9M12 12V9M16 12V9"/>',
  // B-N2-03: teen mode (nbl_Me_Mode)
  sprout:       '<path d="M12 21v-8"/><path d="M12 13c0-4-3-6-7-6 0 4 3 6 7 6zM12 11c0-4 3-6 7-6 0 4-3 6-7 6z"/>',
  // B-N2-07: payment method row (nbl_Prem_Checkout)
  card:         '<rect x="3" y="6" width="18" height="13" rx="2.5"/><path d="M3 10.5h18"/>',
  // CB-CONTRA-04b: injection + implant setup tiles (nbl_Contra_Setup)
  syringe:      '<path d="M18 2l4 4M20 4l-9 9M13 7l4 4M4 20l3-3M7 17l-2-2 8-8 4 4-8 8z"/>',
  implant:      '<circle cx="12" cy="12" r="3"/><path d="M12 3v6M12 15v6M5 5l4 4M15 15l4 4"/>',
  // B-N3-03: log sheet v2 category discs + voice tab (nbl_Log_Sheet_Cycle)
  mic:          '<rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5 11a7 7 0 0014 0M12 18v3"/>',
  gut:          '<path d="M9 3v4a4 4 0 004 4h1a5 5 0 010 10H9a5 5 0 01-5-5"/>',
  dropLine:     '<path d="M12 3s5 6 5 10a5 5 0 01-10 0c0-4 5-10 5-10z"/><path d="M9 14c1 1 2 1.5 3 1.5"/>',
  bed:          '<path d="M3 18v-8h18v8M3 14h18M7 10V7h5v3"/>',
  urine:        '<path d="M12 3v6M8 9h8l-1 12H9z"/>',
  run:          '<circle cx="14" cy="4" r="2"/><path d="M8 21l3-6 3 3v4M9 12l2-4 4 2 3 3"/>',
  scaleSquare:  '<rect x="4" y="4" width="16" height="16" rx="4"/><path d="M9 10a3 3 0 016 0"/>',
  // B-N3-04 (nbl_Log_Customize): drag handle, quick-tile pin, move up.
  grip:         '<path d="M9 6h.01M15 6h.01M9 12h.01M15 12h.01M9 18h.01M15 18h.01"/>',
  pin:          '<path d="M9 3h6l-1 6 4 4H6l4-4z"/><path d="M12 13v8"/>',
  chevronUp:    '<path d="M18 15l-6-6-6 6"/>',
  // B-N4-04 (nbl_Hamdam_Invite): copy code, share, send invite.
  copy:         '<rect x="9" y="9" width="11" height="11" rx="2"/><path d="M5 15V6a2 2 0 012-2h9"/>',
  share:        '<circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="M8.6 13.5l6.8 4M15.4 6.5l-6.8 4"/>',
  send:         '<path d="M4 12l16-8-6 16-3-7z"/>',
  // B-N5-04 (nbl_v15_Main): feeding tile/chip, sleep tile, «ثبت وضعیت امروز».
  bottle:       '<path d="M10 3h4v3h-4zM9 6h6l1 3v10a2 2 0 01-2 2h-4a2 2 0 01-2-2V9z"/><path d="M9 13h6"/>',
  sleep:        '<path d="M20 14.5A8 8 0 019.5 4 8 8 0 1020 14.5z"/><path d="M15 3h4l-4 4h4"/>',
  mother:       '<circle cx="12" cy="7" r="3.5"/><path d="M6 21c0-4 2.5-7 6-7s6 3 6 7"/><path d="M9 13.5c-1 2-1 5-1 7.5"/>',
  // B-N5-06: «ثبت اندازه جدید» (nbl_v16_Growth)
  ruler:        '<path d="M3 17L17 3l4 4L7 21z"/><path d="M8 12l2 2M11 9l2 2M14 6l2 2"/>',
};

interface IconProps {
  name: IconName;
  size?: number;
  fill?: string;
  stroke?: string;
  strokeWidth?: number;
  style?: CSSProperties;
  className?: string;
}

export function Icon({
  name,
  size = 24,
  fill = 'none',
  stroke = 'currentColor',
  strokeWidth = 2,
  style,
  className,
}: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      width={size}
      height={size}
      fill={fill}
      stroke={stroke}
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      style={style}
      className={className}
      // Decorative by contract: the meaning lives in the control's label.
      aria-hidden="true"
      focusable="false"
      dangerouslySetInnerHTML={{ __html: PATHS[name] ?? '' }}
    />
  );
}

/** Solid drop shape for phase indicators. */
export function DropSolid({ size = 16, color }: { size?: number; color: string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill={color}>
      <path d="M12 2.5s6.5 6.6 6.5 11A6.5 6.5 0 0 1 5.5 13.5C5.5 9.1 12 2.5 12 2.5z" />
    </svg>
  );
}
