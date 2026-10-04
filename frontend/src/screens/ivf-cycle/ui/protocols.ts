'use client';

import { useTranslations } from 'next-intl';

import type { IvfProtocol } from '@/entities/ivf';

/** Bundled protocol names — the fallback when the catalog is empty or a row has no title. */
const BUNDLED = ['antagonist', 'long_agonist', 'short_agonist', 'mild', 'natural', 'frozen_transfer'] as const;
type BundledProtocol = (typeof BUNDLED)[number];

const isBundled = (code: string): code is BundledProtocol => (BUNDLED as readonly string[]).includes(code);

/** The catalog's codes in order, else the bundled six. */
export function protocolCodes(catalog: readonly IvfProtocol[] | undefined): string[] {
  return catalog?.length ? catalog.map((p) => p.code) : [...BUNDLED];
}

/** Catalog title → bundled name → the raw code. */
export function useProtocolLabel(catalog: readonly IvfProtocol[] | undefined) {
  const t = useTranslations('ivf.cycle.protocols');
  return (code: string) => {
    const title = catalog?.find((p) => p.code === code)?.title;
    if (title) return title;
    return isBundled(code) ? t(code) : code;
  };
}
