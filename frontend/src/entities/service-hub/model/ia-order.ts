import type { ServiceSection } from './types';

/**
 * CB-NAV-03 (IA_Nav «خدمات», nbd_Nav_Services): whatever order the admin catalog gives the hub sections, the
 * shop is always the last content section — in its own dashed frame with the «separate from your health data»
 * note — and the emergency card closes the screen. Every other section keeps the catalog order.
 */
export function iaOrderedSections(sections: readonly ServiceSection[]): ServiceSection[] {
  const rest = sections.filter((s) => s.code !== 'shop' && s.code !== 'emergency');
  const shop = sections.filter((s) => s.code === 'shop');
  const emergency = sections.filter((s) => s.code === 'emergency');
  return [...rest, ...shop, ...emergency];
}
