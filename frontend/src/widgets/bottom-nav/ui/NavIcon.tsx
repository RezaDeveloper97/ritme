import { Art3d, type Art3dPath } from '@/shared/ui';

import type { NavIconName } from '../model/nav-items';
import './nbnav.css';

/** The 3D picture of each tab (public/art3d). The active tab is full colour; the others are muted in CSS. */
const ART: Record<NavIconName, Art3dPath> = {
  home: 'icons/home',
  calendar: 'icons/calendar',
  fertility: 'ttc',
  treatment: 'icons/syringe',
  pregnancy: 'pregnancy',
  child: 'icons/child',
  symptoms: 'icons/symptom',
  services: 'icons/grid',
  analysis: 'icons/chart',
  me: 'icons/user',
  learn: 'icons/bookOpen',
  actions: 'icons/gift',
};

export function NavIcon({ name }: { name: NavIconName }) {
  return <Art3d name={ART[name]} size={26} className="nbnav-ic" />;
}
