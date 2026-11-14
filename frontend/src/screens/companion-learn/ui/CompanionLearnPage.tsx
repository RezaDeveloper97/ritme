'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCompanionHome } from '@/entities/companion';
import { useRouter } from '@/shared/i18n';
import { AppSheet, openSheet } from '@/shared/sheet';
import { ScreenHeader, SkyLayer } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { LEARN_TOPICS, type LearnTopic, type LearnTopicId, LearnTile, matchArticle } from '@/widgets/companion-support';

/**
 * `/companion/learn` — «یادگیری» (companion v2 compLearn): four short topics and «افسانه‌های رایج». A topic
 * opens a matching companion article when the server has one, else a half sheet with short, non-medical,
 * non-definitive copy.
 */
export function CompanionLearnPage() {
  const t = useTranslations('companionHome.learn');
  const router = useRouter();
  const home = useCompanionHome();
  const [topic, setTopic] = useState<LearnTopicId | null>(null);
  const articles = home.data?.articles ?? [];

  const open = (tp: LearnTopic) => {
    const article = matchArticle(tp, articles);
    if (article) openSheet('article', article.slug);
    else setTopic(tp.id);
  };

  return (
    <div className="view cmh-page">
      <SkyLayer />
      <ScreenHeader title={t('title')} onBack={() => router.replace('/companion')} backLabel={t('back')} />
      <div className="scroll cmh-scroll">
        <div className="cmh-body">
          <div className="cmh2-learn-intro">
            <h2 className="cmh2-title is-start">{t('heading')}</h2>
            <p className="cmh2-sub is-start">{t('sub')}</p>
          </div>
          <div className="cmh2-tiles">
            {LEARN_TOPICS.map((tp) => (
              <LearnTile key={tp.id} topic={tp} label="long" onOpen={() => open(tp)} tall />
            ))}
          </div>
          <section className="nb-card cmh2-myth" aria-labelledby="cmh2-myth-title">
            <h2 id="cmh2-myth-title" className="cmh2-myth-title">
              {t('myths.title')}
            </h2>
            <p className="cmh2-myth-body">{t('myths.body')}</p>
          </section>
        </div>
      </div>
      <AppSheet
        open={topic != null}
        onClose={() => setTopic(null)}
        size="half"
        title={topic ? t(`topics.${topic}.long`) : undefined}
      >
        {topic ? (
          <div className="cmh2-sheet-copy">
            <p>{t(`topics.${topic}.body`)}</p>
            <p className="cmh2-sheet-note">{t('disclaimer')}</p>
          </div>
        ) : null}
      </AppSheet>
      <BottomNav mode="companion" />
    </div>
  );
}
