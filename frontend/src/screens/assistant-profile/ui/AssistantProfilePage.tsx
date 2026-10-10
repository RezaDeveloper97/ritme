'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import {
  BodySkeleton,
  ConsentCard,
  LoadError,
  PROFILE_KEYS,
  scalarOf,
  useAssistantError,
  useAssistantProfile,
  useSaveProfile,
} from '@/entities/assistant';
import { useRouter } from '@/shared/i18n';
import { Card, InfoNote, PrimaryButton, ScreenHeader, SkyLayer, Switch } from '@/shared/ui';

const GROUPS = [
  { id: 'basic', keys: ['age', 'life_mode', 'basics'] },
  { id: 'ritme', keys: ['cycle', 'pregnancies', 'conditions', 'medications', 'allergies'] },
] as const;

/**
 * `/assistant/profile` — «پروفایل سلامت» (nbl_/nbd_v18_Profile): one toggle per shareable item. Nothing is used (or sent
 * to the model) until the `assistant_profile` consent is accepted; pregnancy losses are never shared (backend rule).
 */
export function AssistantProfilePage() {
  const t = useTranslations('assistant');
  const router = useRouter();
  const errText = useAssistantError();
  const q = useAssistantProfile();
  const save = useSaveProfile();
  const [shared, setShared] = useState<string[] | null>(null);
  const [saved, setSaved] = useState(false);
  const server = q.data?.shared;
  useEffect(() => {
    if (server && shared === null) setShared(server);
  }, [server, shared]);

  const toggle = (key: string, on: boolean) => {
    setSaved(false);
    setShared((cur) => (on ? [...(cur ?? []), key] : (cur ?? []).filter((k) => k !== key)));
  };

  let body;
  if (q.isPending || (q.isSuccess && shared === null)) body = <BodySkeleton rows={3} />;
  else if (q.isError || !q.data || shared === null) body = <div className="as-body"><LoadError onRetry={() => void q.refetch()} busy={q.isFetching} /></div>;
  else {
    const dirty = JSON.stringify([...shared].sort()) !== JSON.stringify([...q.data.shared].sort());
    body = (
      <div className="as-body">
        {!q.data.consent.granted ? <ConsentCard code="assistant_profile" /> : null}
        {GROUPS.map((g) => (
          <Card key={g.id} className="as-switches">
            <h3 className="as-h3">{t(`profile.group.${g.id}`)}</h3>
            {g.keys
              .filter((k) => (PROFILE_KEYS as readonly string[]).includes(k))
              .map((k) => {
                const item = q.data.items.find((i) => i.key === k);
                const hint = item && scalarOf(item) !== null ? String(scalarOf(item)) : t(`profile.hint.${k}` as 'profile.hint.age');
                return (
                  <div key={k} className="as-switch">
                    <span className="as-row-main">
                      <strong id={`as-sw-${k}`}>{t(`profile.keys.${k}` as 'profile.keys.age')}</strong>
                      <span>{hint}</span>
                    </span>
                    <Switch checked={shared.includes(k)} onCheckedChange={(on) => toggle(k, on)} labelledBy={`as-sw-${k}`} />
                  </div>
                );
              })}
          </Card>
        ))}
        <InfoNote icon="shield">{t('profile.note')}</InfoNote>
        {save.isError ? <p className="as-error" role="alert">{errText(save.error)}</p> : null}
        {saved ? <p className="as-ok" role="status">{t('profile.saved')}</p> : null}
        <PrimaryButton block loading={save.isPending} disabled={!dirty} onClick={() => save.mutate(shared, { onSuccess: () => setSaved(true) })}>
          {t('profile.save')}
        </PrimaryButton>
      </div>
    );
  }

  return (
    <div className="view as-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('profile.title')} subtitle={t('profile.subtitle')} onBack={() => router.push('/assistant')} backLabel={t('common.back')} />
        {body}
      </div>
    </div>
  );
}
