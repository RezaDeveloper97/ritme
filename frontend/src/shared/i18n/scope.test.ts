import { describe, expect, it } from 'vitest';

import { pickNamespaces, scopeNamespace } from './scope';

const bundle = () => ({
  common: { close: 'بستن', open: 'باز' },
  home: {
    title: 'خانه',
    nb: { hero: { a: '1', b: '2' }, log: { c: '3' }, other: { d: '4' } },
    skip: { text: 'x' },
  },
  menopause: { home: { x: 'x' }, report: { y: 'y' } },
});

describe('pickNamespaces', () => {
  it('copies whole namespaces', () => {
    expect(pickNamespaces(bundle(), ['common'])).toEqual({ common: bundle().common });
  });

  it('copies only the named subtree of a dotted scope', () => {
    expect(pickNamespaces(bundle(), ['home.nb.hero', 'menopause.home'])).toEqual({
      home: { nb: { hero: { a: '1', b: '2' } } },
      menopause: { home: { x: 'x' } },
    });
  });

  it('merges sibling scopes of one namespace', () => {
    expect(pickNamespaces(bundle(), ['home.title', 'home.nb.log', 'home.nb.hero'])).toEqual({
      home: { title: 'خانه', nb: { log: { c: '3' }, hero: { a: '1', b: '2' } } },
    });
  });

  it('lets the broader entry win, whatever the order', () => {
    const want = { home: bundle().home };
    expect(pickNamespaces(bundle(), ['home', 'home.nb.log'])).toEqual(want);
    expect(pickNamespaces(bundle(), ['home.nb.log', 'home'])).toEqual(want);
    const nb = { home: { nb: bundle().home.nb } };
    expect(pickNamespaces(bundle(), ['home.nb.log', 'home.nb'])).toEqual(nb);
  });

  it('skips a namespace or path that is not in the bundle, as the namespace-only version did', () => {
    expect(pickNamespaces(bundle(), ['nope' as 'common', 'home.missing.deep', 'home.title.deeper'])).toEqual({});
  });

  it('never writes into the source bundle', () => {
    const source = bundle();
    pickNamespaces(source, ['home.nb', 'home.nb.log', 'home.title', 'menopause.home']);
    expect(source).toEqual(bundle());
  });

  it('a subtree that was not asked for stays out', () => {
    const picked = pickNamespaces(bundle(), ['home.nb.hero']) as { home: { nb: Record<string, unknown> } };
    expect(Object.keys(picked.home.nb)).toEqual(['hero']);
    expect(picked).not.toHaveProperty('common');
  });
});

describe('scopeNamespace', () => {
  it('names the namespace of a whole or dotted scope', () => {
    expect(scopeNamespace('common')).toBe('common');
    expect(scopeNamespace('home.nb.hero')).toBe('home');
  });
});
