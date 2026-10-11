'use client';

import { useTranslations } from 'next-intl';
import { usePathname } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';

import { type Ad, type AdSlot as Slot, sendAdEvent, useAds } from '@/entities/banner';
import { Link } from '@/shared/i18n';

import { capReached, pathAllowsAds, recordShown } from '../lib/frequency';

interface Props {
  slot: Slot;
  /** The effective app mode — the only targeting signal (never health data). `teen` renders nothing. */
  mode: string | null;
}

/** Impressions already counted in this page session, so a re-render or scroll-back never double counts. */
const counted = new Set<number>();

function AdCard({ ad, slot, onFail }: { ad: Ad; slot: Slot; onFail: () => void }) {
  const t = useTranslations('banners');
  const ref = useRef<HTMLDivElement>(null);

  // One impression when at least half the card is on screen.
  useEffect(() => {
    const el = ref.current;
    if (!el || counted.has(ad.id)) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting) && !counted.has(ad.id)) {
          counted.add(ad.id);
          recordShown(ad.id);
          sendAdEvent(ad.id, 'impression');
          io.disconnect();
        }
      },
      { threshold: 0.5 },
    );
    io.observe(el);
    return () => io.disconnect();
  }, [ad.id]);

  const body = (
    <>
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img src={ad.imageUrl} alt={ad.title ?? ''} draggable={false} className="ad-img" onError={onFail} />
      <span className="ad-label">{t('adLabel')}</span>
      {ad.ctaLabel && <span className="ad-cta">{ad.ctaLabel}</span>}
    </>
  );
  const onClick = () => sendAdEvent(ad.id, 'click');
  const aria = [t('adLabel'), ad.title, ad.ctaLabel].filter(Boolean).join(' · ');

  return (
    <div ref={ref} className={`ad-card ad-${slot}`}>
      {!ad.linkUrl ? (
        <div className="ad-link">{body}</div>
      ) : ad.linkType === 'external' ? (
        <a className="ad-link" href={ad.linkUrl} target="_blank" rel="noopener sponsored" aria-label={aria} onClick={onClick}>
          {body}
        </a>
      ) : (
        <Link className="ad-link" href={ad.linkUrl} aria-label={aria} onClick={onClick}>
          {body}
        </Link>
      )}
    </div>
  );
}

/**
 * A sponsored home slot (B-N10-02). Always labelled «تبلیغ» (not editable by the advertiser), external links open
 * with `rel="noopener sponsored"`, a client-side frequency cap hides an ad after a few views a day, and the slot
 * renders nothing for teen mode or on teen / loss / emergency / consent / record / labs / doctor-chat paths.
 */
export function AdSlot({ slot, mode }: Props) {
  const t = useTranslations('banners');
  const pathname = usePathname() ?? '';
  const ads = useAds(slot, mode);
  const [failed, setFailed] = useState<ReadonlySet<number>>(() => new Set());
  // Cap is read once on mount of the list so scrolling doesn't make cards vanish under the finger.
  const [visible, setVisible] = useState<Ad[] | null>(null);

  useEffect(() => {
    if (ads.length && visible === null) setVisible(ads.filter((a) => !capReached(a.id)));
  }, [ads, visible]);

  if (mode === 'teen' || !pathAllowsAds(pathname)) return null;
  const shown = (visible ?? []).filter((a) => !failed.has(a.id));
  if (shown.length === 0) return null;

  return (
    <div className="sec">
      <div className={`ad-row ad-row-${slot}`} role="region" aria-label={t('adRegion')}>
        {shown.map((ad) => (
          <AdCard key={ad.id} ad={ad} slot={slot} onFail={() => setFailed((p) => new Set(p).add(ad.id))} />
        ))}
      </div>
    </div>
  );
}
