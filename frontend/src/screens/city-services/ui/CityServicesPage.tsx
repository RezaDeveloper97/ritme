'use client';

import { useState } from 'react';

import { type BrowseQuery, CityServicesHome } from './CityServicesHome';
import { DirectoryList } from './DirectoryList';

/**
 * CB-DIR-06: `/city-services` is the «مادر و کودک» home (`CityServicesHome`); tapping a category, a search or «همه»
 * opens the list with that filter. CB-DIR-07 replaced bloom's chip-only list (B-N10-05) with `DirectoryList`
 * (search, filter sheet, richer cards, map link); the map itself is `/city-services/map`.
 */
export function CityServicesPage() {
  const [browse, setBrowse] = useState<BrowseQuery | null>(null);
  return browse ? <DirectoryList initial={browse} onHome={() => setBrowse(null)} /> : <CityServicesHome onBrowse={setBrowse} />;
}
