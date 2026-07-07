'use client';

import { useTranslations } from 'next-intl';

import {
  useAlertAction,
  useMarkAllAlertsRead,
  usePregnancyAlerts,
  type AlertLevel,
  type PregnancyAlert,
} from '@/entities/pregnancy';
import { Icon, type IconName } from '@/shared/ui';

type T = ReturnType<typeof useTranslations>;

const LEVEL_STYLE: Record<AlertLevel, { icon: IconName; color: string; soft: string }> = {
  info: { icon: 'info', color: '#2E9BE9', soft: '#E3F1FD' },
  warning: { icon: 'flame', color: '#E9662E', soft: '#FDEBE2' },
  emergency: { icon: 'shield', color: '#E5484D', soft: '#FCE7E7' },
};

function AlertRow({ alert, t, onRead, onDismiss }: { alert: PregnancyAlert; t: T; onRead: () => void; onDismiss: () => void }) {
  const style = LEVEL_STYLE[alert.alertLevel];
  return (
    <div
      className="card"
      style={{
        padding: '12px 13px',
        textAlign: 'start',
        borderInlineStart: `3px solid ${style.color}`,
        opacity: alert.isRead ? 0.7 : 1,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 9 }}>
        <span className="dot" style={{ width: 28, height: 28, background: style.soft, color: style.color }}>
          <Icon name={style.icon} size={15} />
        </span>
        <span style={{ flex: 1, fontSize: 13.5, fontWeight: 800, color: 'var(--ink)' }}>{alert.title}</span>
        {!alert.isRead && (
          <span style={{ fontSize: 10.5, fontWeight: 700, color: style.color, background: style.soft, borderRadius: 20, padding: '2px 8px' }}>
            {t('alerts.unreadBadge')}
          </span>
        )}
      </div>
      <p style={{ margin: '8px 0 0', fontSize: 12.5, lineHeight: 1.8, color: 'var(--muted)' }}>{alert.message}</p>
      {alert.recommendedActions.length > 0 && (
        <ul style={{ margin: '8px 0 0', paddingInlineStart: 18, fontSize: 12, color: 'var(--ink)', lineHeight: 1.9 }}>
          {alert.recommendedActions.map((a, i) => (
            <li key={i}>{a}</li>
          ))}
        </ul>
      )}
      <div style={{ display: 'flex', gap: 8, marginTop: 10 }}>
        {!alert.isRead && (
          <button type="button" className="btn btn-soft" onClick={onRead} style={{ height: 32, fontSize: 12, padding: '0 12px', width: 'auto' }}>
            <Icon name="check" size={14} />
          </button>
        )}
        <button type="button" className="btn btn-ghost" onClick={onDismiss} style={{ height: 32, fontSize: 12, padding: '0 12px', width: 'auto' }}>
          {t('alerts.dismiss')}
        </button>
      </div>
    </div>
  );
}

/** The alerts section on the tracker: a list with per-item read/dismiss and a
 *  "mark all read" shortcut. Alert copy is already localized by the backend. */
export function AlertsCard({ t }: { t: T }) {
  const alertsQuery = usePregnancyAlerts();
  const action = useAlertAction();
  const markAll = useMarkAllAlertsRead();

  const alerts = alertsQuery.data?.alerts ?? [];
  const hasUnread = (alertsQuery.data?.counts.unread ?? 0) > 0;

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', margin: '2px 2px 10px' }}>
        <span style={{ fontSize: 15, fontWeight: 800, color: 'var(--ink)' }}>{t('alerts.title')}</span>
        {hasUnread && (
          <button type="button" onClick={() => markAll.mutate()} style={{ background: 'none', border: 0, color: 'var(--brand)', fontSize: 12.5, fontWeight: 700, cursor: 'pointer', fontFamily: 'inherit' }}>
            {t('alerts.markAllRead')}
          </button>
        )}
      </div>

      {alerts.length === 0 ? (
        <div className="card" style={{ padding: '16px 14px', textAlign: 'center', color: 'var(--muted)', fontSize: 13, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8 }}>
          <Icon name="checkCircle" size={16} style={{ color: 'var(--green)' }} />
          {t('alerts.empty')}
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 9 }}>
          {alerts.map((alert) => (
            <AlertRow
              key={alert.id}
              alert={alert}
              t={t}
              onRead={() => action.mutate({ id: alert.id, action: 'read' })}
              onDismiss={() => action.mutate({ id: alert.id, action: 'dismiss' })}
            />
          ))}
        </div>
      )}
    </div>
  );
}
