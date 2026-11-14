// Plain module (no 'use client'): the server route `/analysis` calls analysisTabOf(), and a server component
// cannot call a function exported from a client module.
export type AnalysisTab = 'analysis' | 'history';

/** `?tab=history` opens the history tab; anything else the analysis hub. */
export function analysisTabOf(tab: string | undefined): AnalysisTab {
  return tab === 'history' ? 'history' : 'analysis';
}
