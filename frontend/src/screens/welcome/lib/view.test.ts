import { describe, expect, it } from 'vitest';

import { resolveWelcomeView } from './view';

describe('resolveWelcomeView', () => {
  it('shows the slides to a first-time visitor and the card afterwards', () => {
    expect(resolveWelcomeView('', false)).toEqual({ kind: 'intro', slide: 0 });
    expect(resolveWelcomeView('', true)).toEqual({ kind: 'card' });
  });

  it('honours ?step=welcome and a valid ?slide=N', () => {
    expect(resolveWelcomeView('?step=welcome', false)).toEqual({ kind: 'card' });
    expect(resolveWelcomeView('?slide=3', true)).toEqual({ kind: 'intro', slide: 2 });
  });

  it('ignores an out-of-range or malformed slide', () => {
    expect(resolveWelcomeView('?slide=9', false)).toEqual({ kind: 'intro', slide: 0 });
    expect(resolveWelcomeView('?slide=x', true)).toEqual({ kind: 'card' });
  });
});
