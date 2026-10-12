'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useSaveSupplyReminder, useShopChecklist, useSupplyReminder } from '@/entities/shop';
import { useLifeStage } from '@/entities/user';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, Icon, ProgressBar, Switch } from '@/shared/ui';

/**
 * CB-SHOP-09: the layette-checklist card of the baby home — progress bar + «X از Y آماده», links to
 * `/shop/checklist`. Renders nothing when the list is unavailable (teen mode, no template yet).
 */
export function ChecklistCard() {
  const t = useTranslations('shop.home.checklist');
  const locale = useLocale() as Locale;
  const q = useShopChecklist(null);
  if (!q.data || q.data.summary.total === 0) return null;
  const { ready, total } = q.data.summary;
  return (
    <Link href="/shop/checklist" className="shp-checklist">
      <Card className="shp-checklist-card">
        <Icon name="todo" size={22} />
        <div className="shp-checklist-main">
          <p className="shp-checklist-title">{t('title')}</p>
          <ProgressBar
            value={ready}
            max={total}
            label={t('progress')}
            valueLabel={t('count', { ready: formatNumber(ready, locale), total: formatNumber(total, locale) })}
          />
        </div>
        <Icon name="forward" size={18} />
      </Card>
    </Link>
  );
}

/**
 * CB-SHOP-09: the discreet personal-supplies reminder (opt-in, off by default). Visible copy never names the reason:
 * it is a plain "look at your personal supplies" nudge before the predicted date, kept on the server. Nothing reaches
 * a seller. Renders nothing for a teen-mode account (404), and (2026-10-12) while the life mode is pregnancy,
 * postpartum or menopause — the server only schedules this reminder for cycle / TTC / no-mode accounts
 * (`shopPeriodPredictor`), so a toggle there would be a no-op.
 */
export function SupplyReminderCard() {
  const t = useTranslations('shop.home.supply');
  const q = useSupplyReminder();
  const save = useSaveSupplyReminder();
  const life = useLifeStage();
  const mode = life.data?.mode;
  if (!q.data || life.isPending || mode === 'pregnancy' || mode === 'postpartum' || mode === 'menopause') return null;
  const on = q.data.enabled;
  return (
    <Card className="shp-supply">
      <Icon name="bell" size={22} />
      <div className="shp-supply-main">
        <p id="shp-supply-title" className="shp-supply-title">
          {t('title')}
        </p>
        <p className="shp-supply-body">{t('body')}</p>
      </div>
      <Switch
        checked={on}
        disabled={save.isPending}
        labelledBy="shp-supply-title"
        onCheckedChange={(enabled) => save.mutate({ enabled })}
      />
    </Card>
  );
}
