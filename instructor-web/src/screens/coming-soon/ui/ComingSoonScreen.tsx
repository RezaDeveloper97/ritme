import { useTranslations } from 'next-intl';

import { Icon, type IconName } from '@/shared/ui';

type Section = 'content' | 'groups' | 'students';

const ICONS: Record<Section, IconName> = { content: 'content', groups: 'groups', students: 'students' };

/** Placeholder of a panel tab whose screens come with B-N8-06 / B-N8-07. */
export function ComingSoonScreen({ section }: { section: Section }) {
  const t = useTranslations('comingSoon');
  return (
    <div className="page">
      <h1 className="page__title">{t(`${section}.title`)}</h1>
      <section className="empty-card">
        <span className="empty-card__icon">
          <Icon name={ICONS[section]} size={26} />
        </span>
        <h2 className="empty-card__title">{t('title')}</h2>
        <p className="empty-card__text">{t(`${section}.text`)}</p>
      </section>
    </div>
  );
}
