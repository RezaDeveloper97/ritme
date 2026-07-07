/** Fixed home-page slots a banner can occupy (mirrors the backend enum). */
export type BannerPosition = 'home_top' | 'home_middle' | 'home_bottom';

/** How a banner's link opens: an in-app route vs. an absolute external URL. */
export type BannerLinkType = 'internal' | 'external';

/** A single promotional banner as rendered by the slideshow. */
export interface Banner {
  id: number;
  /** Optional caption / alt text, already localized by the API. */
  title: string | null;
  imageUrl: string;
  position: BannerPosition;
  /** `null` when the banner is purely decorative (no link). */
  linkUrl: string | null;
  linkType: BannerLinkType | null;
}

/** Active banners grouped by slot — the shape returned by `useBanners`. */
export type BannersByPosition = Record<BannerPosition, Banner[]>;
