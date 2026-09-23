'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';

import { Link } from '@/shared/i18n';
import { Icon, type IconName } from '@/shared/ui';


interface Choice {
  key: 'medication' | 'appointment' | 'consult';
  href: string;
  icon: IconName;
  tone: 'brand' | 'amber' | 'teal';
}

/** «مشاوره» opens the same form as a visit, pre-set to a phone consultation. */
const CHOICES: readonly Choice[] = [
  { key: 'medication', href: '/reminders/medication/new', icon: 'tablet', tone: 'brand' },
  { key: 'appointment', href: '/reminders/appointment/new?kind=in_person', icon: 'doctor', tone: 'amber' },
  { key: 'consult', href: '/reminders/appointment/new?kind=phone', icon: 'video', tone: 'teal' },
];

/**
 * AddChooser (`v13_AddChooser`) — «چه چیزی را یادآوری کنم؟». Sheet content
 * only: the grip, title and close button are `AppSheet`'s (CLAUDE.md §4.1).
 * Following a row navigates to its form, which closes the sheet.
 */
export function AddChooserSheet() {
  const t = useTranslations('care');

  return (
    <div className="rad-list">
      {CHOICES.map((choice) => (
        <Link key={choice.key} href={choice.href} className="rad-row">
          <span className={clsx('rad-tile', `is-${choice.tone}`)} aria-hidden>
            <Icon name={choice.icon} size={22} strokeWidth={1.8} />
          </span>
          <span className="rad-body">
            <span className="rad-title">{t(`chooser.${choice.key}.title`)}</span>
            <span className="rad-desc">{t(`chooser.${choice.key}.description`)}</span>
          </span>
          <Icon name="chevronLeft" size={18} strokeWidth={2} className="rad-chev" />
        </Link>
      ))}
      <p className="rad-note">{t('chooser.disclaimer')}</p>
    </div>
  );
}
