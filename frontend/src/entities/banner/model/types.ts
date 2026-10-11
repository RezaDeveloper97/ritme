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

/** Sponsored home slots (B-N10-02): `ad_slider` 342x170, `ad_square` 166x166, `ad_strip` 342x76. */
export type AdSlot = 'ad_slider' | 'ad_square' | 'ad_strip';

/** One sponsored item. The advertiser name is admin-only; the «تبلیغ» label is rendered by the client. */
export interface Ad {
  id: number;
  title: string | null;
  imageUrl: string;
  slot: AdSlot;
  ctaLabel: string | null;
  linkUrl: string | null;
  linkType: BannerLinkType | null;
}
