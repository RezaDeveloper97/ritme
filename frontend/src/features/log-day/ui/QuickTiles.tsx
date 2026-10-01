'use client';

import { useTranslations } from 'next-intl';

import type { LogCategory, LogDayValues } from '@/entities/health-log';
import { TileButton } from '@/shared/ui';

import { categoryEntries, paramEntries, parseTileKey, type LabelContext } from '../model/draft';
import { tileLook } from '../model/presentation';

interface QuickTilesProps {
  tiles: readonly string[];
  categories: readonly LogCategory[];
  values: LogDayValues;
  labels: LabelContext;
  openCategory: string | null;
  onOpen: (category: string) => void;
}

/**
 * «ثبت سریع» (nbl_Log_Sheet_Cycle): up to 8 pinned tiles from `/logs/preferences`, 4 per row. A tile
 * opens its accordion section below; its caption shows what is logged, or «باز است» while open.
 */
export function QuickTiles({ tiles, categories, values, labels, openCategory, onOpen }: QuickTilesProps) {
  const t = useTranslations('logSheet');
  if (!tiles.length) return null;
  return (
    <section className="lday-quick" aria-labelledby="lday-quick-title">
      <div className="lday-sec-head">
        <h3 id="lday-quick-title" className="lday-sec-title">
          {t('quick.title')}
        </h3>
        <span className="lday-sec-hint">{t('quick.hint')}</span>
      </div>
      <div className="lday-tiles">
        {tiles.map((key) => {
          const { category: code, param: paramCode } = parseTileKey(key);
          const category = categories.find((c) => c.code === code);
          if (!category) return null;
          const param = paramCode ? category.params.find((p) => p.code === paramCode) : undefined;
          if (paramCode && !param) return null;
          const tileKey = `tiles.${code}` as 'tiles.pain';
          const label = param ? param.label : t.has(tileKey) ? t(tileKey) : category.label;
          const logged = param
            ? paramEntries(code, param, values[code]?.[param.code], labels)
            : categoryEntries(category, values, labels);
          const open = openCategory === code;
          const look = tileLook(key);
          return (
            <TileButton
              key={key}
              layout="card"
              icon={look.icon}
              tone={look.tone}
              label={label}
              sub={open ? t('quick.open') : param ? logged[0]?.summary : logged[0]?.label}
              pressed={open}
              aria-controls={`lday-cat-${code}`}
              onClick={() => onOpen(code)}
              className="lday-tile"
            />
          );
        })}
      </div>
    </section>
  );
}
