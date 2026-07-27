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
  info: { icon: 'info', color: 'var(--blue)', soft: 'var(--blue-soft)' },
  warning: { icon: 'flame', color: 'var(--orange)', soft: 'var(--orange-soft)' },
  emergency: { icon: 'shield', color: 'var(--danger)', soft: 'var(--danger-soft)' },
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
      <div className="alert-head">
        <span className="dot alert-dot" style={{ background: style.soft, color: style.color }}>
          <Icon name={style.icon} size={15} />
        </span>
        <span className="alert-title">{alert.title}</span>
        {!alert.isRead && (
          <span className="alert-badge" style={{ color: style.color, background: style.soft }}>
            {t('alerts.unreadBadge')}
          </span>
        )}
      </div>
      <p className="alert-msg">{alert.message}</p>
      {alert.recommendedActions.length > 0 && (
        <ul className="alert-actions">
          {alert.recommendedActions.map((a, i) => (
            <li key={i}>{a}</li>
          ))}
        </ul>
      )}
      <div className="alert-btns">
        {!alert.isRead && (
          <button type="button" className="btn btn-soft alert-btn" onClick={onRead}>
            <Icon name="check" size={14} />
          </button>
        )}
        <button type="button" className="btn btn-ghost alert-btn" onClick={onDismiss}>
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
      <div className="alerts-head">
        <span className="alerts-title">{t('alerts.title')}</span>
        {hasUnread && (
          <button type="button" className="alerts-markall" onClick={() => markAll.mutate()}>
            {t('alerts.markAllRead')}
          </button>
        )}
      </div>

      {alerts.length === 0 ? (
        <div className="card alerts-empty">
          <Icon name="checkCircle" size={16} className="alerts-empty-icon" />
          {t('alerts.empty')}
        </div>
      ) : (
        <div className="alerts-list">
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
