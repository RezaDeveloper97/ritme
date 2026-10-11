import type { MetadataRoute } from 'next';

/**
 * CB-DIR-11: the public directory pages (`/<locale>/places*`) are meant to be crawled; everything that is private by
 * design (doctor code / shared report, emergency card, a booking confirmation) is kept out. The app itself sits behind
 * the sign-in guard, so crawlers only ever see the public routes anyway.
 */
export default function robots(): MetadataRoute.Robots {
  return {
    rules: [{ userAgent: '*', allow: '/', disallow: ['/*/d', '/*/shared', '/*/emergency', '/*/places/booked'] }],
  };
}
