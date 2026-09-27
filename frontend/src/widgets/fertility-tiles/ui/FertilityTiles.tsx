'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { formatBbt, useFertilityToday } from '@/entities/fertility';
import { Link, type Locale } from '@/shared/i18n';
import { Icon, type IconName } from '@/shared/ui';

type Tone = 'lh' | 'bbt' | 'intercourse';

/**
 * Icon disc tone (`v19_Main` / `nb2_Main`): LH amber, BBT teal, intercourse rose —
 * the `--fert-*` tokens via `.fert-tone-*`; `.fert-disc` draws 13 % fill / 33 % border.
 */
const TONE: Record<Tone, string> = {
  lh: 'fert-tone-amber',
  bbt: 'fert-tone-teal',
  intercourse: 'fert-tone-rose',
};

const ICON: Record<Tone, IconName> = { lh: 'flask', bbt: 'thermo', intercourse: 'heart' };

interface Tile {
  tone: Tone;
  href: string;
  label: string;
  value: string;
  aria: string;
}

function TileLink({ tile }: { tile: Tile }) {
  return (
    <Link
      href={tile.href}
      aria-label={`${tile.aria} — ${tile.value}`}
      className="flex min-h-11 min-w-11 flex-col items-center gap-2 rounded-2xl bg-(--surface) px-2 py-3 text-center focus-visible:outline-2 focus-visible:outline-(--brand)"
    >
      <span
        aria-hidden
        className={clsx('fert-disc size-12 rounded-full', TONE[tile.tone])}
      >
        <Icon name={ICON[tile.tone]} size={22} />
      </span>
      <span className="text-sm font-bold text-(--ink)">{tile.label}</span>
      <span className="text-xs text-(--muted)">{tile.value}</span>
    </Link>
  );
}

function TilesSkeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-busy="true" aria-label={label} className="grid grid-cols-3 gap-3">
      {[0, 1, 2].map((i) => (
        <div key={i} className="h-[116px] animate-pulse rounded-2xl bg-(--surface)" />
      ))}
    </div>
  );
}

/**
 * TTC quick tiles (LH / BBT / intercourse) for the cycle home, read from
 * `/fertility/today`. The home mounts it only for `pregnancyIntention === 'trying'`.
 * Renders nothing on error so the rest of the home is unaffected.
 */
export function FertilityTiles() {
  const t = useTranslations('fertility');
  const locale = useLocale() as Locale;
  const { data, isPending, isError } = useFertilityToday();

  if (isError) return null;

  return (
    <section aria-label={t('tiles.label')} className="mx-4 mt-4.5">
      {isPending || !data ? (
        <TilesSkeleton label={t('tiles.label')} />
      ) : (
        <div className="grid grid-cols-3 gap-3">
          {(
            [
              {
                tone: 'lh',
                href: '/fertility/log?focus=lh',
                label: t('tiles.lh'),
                value: data.lh.value
                  ? (data.lh.label ?? t(`log.lh.options.${data.lh.value}`))
                  : t('tiles.notLogged'),
                aria: t('tiles.openLh'),
              },
              {
                tone: 'bbt',
                href: '/fertility/bbt',
                label: t('tiles.bbt'),
                value:
                  data.bbt != null
                    ? t('tiles.bbtValue', { value: formatBbt(data.bbt, locale) })
                    : t('tiles.notLogged'),
                aria: t('tiles.openBbt'),
              },
              {
                tone: 'intercourse',
                href: '/fertility/log?focus=intercourse',
                label: t('tiles.intercourse'),
                value: data.intercourse.value
                  ? (data.intercourse.label ?? t(`log.intercourse.options.${data.intercourse.value}`))
                  : t('tiles.notLogged'),
                aria: t('tiles.openIntercourse'),
              },
            ] satisfies Tile[]
          ).map((tile) => (
            <TileLink key={tile.tone} tile={tile} />
          ))}
        </div>
      )}
    </section>
  );
}
