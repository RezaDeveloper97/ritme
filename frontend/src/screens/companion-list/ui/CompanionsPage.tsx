'use client';

import { useTranslations } from 'next-intl';

import { CompanionCard, companionName, FamilyStrip, familySpouse, useCompanions } from '@/entities/companion';
import { useUserProfile } from '@/entities/user';
import { useRouter } from '@/shared/i18n';
import { EmptyState, PrimaryButton, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';
import { ParentCodeCard } from '@/widgets/linked-teen-card';

/**
 * «همدم‌ها» (`/companions`, B-N4-04, nbl_Hamdam_List / nbd_Hamdam_List): the
 * owner's invited and active companions, each with the sections it may see,
 * the «خانواده شما» strip when there is a spouse, and «افزودن همدم».
 */
export function CompanionsPage() {
  const t = useTranslations('companions');
  const router = useRouter();
  const query = useCompanions();
  const profile = useUserProfile();
  const companions = query.data ?? [];
  const spouse = familySpouse(companions);

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('list.loading')} className="cmp-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="users"
        title={t('list.errorTitle')}
        body={t('list.errorBody')}
        action={
          <PrimaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('list.retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (companions.length === 0) {
    body = <EmptyState icon="users" title={t('list.emptyTitle')} body={t('list.emptyBody')} className="cmp-empty" />;
  } else {
    const spouseName = spouse ? (companionName(spouse) ?? t('unnamed')) : '';
    body = (
      <>
        <ul className="cmp-list">
          {companions.map((c) => (
            <li key={c.id}>
              <CompanionCard companion={c} onOpen={() => router.push(`/companions/${c.id}`)} />
            </li>
          ))}
        </ul>
        {spouse ? (
          <FamilyStrip
            selfName={profile.data?.name ?? null}
            companionName={spouseName}
            note={spouse.status === 'invited' ? t('family.pending', { name: spouseName }) : t('family.childrenSoon')}
          />
        ) : null}
      </>
    );
  }

  return (
    <div className="view cmp-page">
      <SkyLayer />
      <div className="scroll cmp-scroll is-form">
        <ScreenHeader title={t('list.title')} onBack={() => router.push('/profile')} backLabel={t('back')} />
        <div className="cmp-body">
          <div className="cmp-intro">
            <h2 className="cmp-heading">{t('list.heading')}</h2>
            <p className="cmp-lead">{t('list.lead')}</p>
          </div>
          {body}
          {/* CB-TEEN-03: where a mother accepts her teen's sharing code. */}
          <ParentCodeCard />
        </div>
        <div className="cmp-footer">
          <PrimaryButton icon="plus" onClick={() => router.push('/companions/new')}>
            {t('list.add')}
          </PrimaryButton>
        </div>
      </div>
    </div>
  );
}
