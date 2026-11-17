'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useMemo, useState } from 'react';

import { CENTRE_TYPES, type CentreType, type InsuranceCentre, useCentres } from '@/entities/insurance';
import { useRouter } from '@/shared/i18n';
import { pilot, pilotOption } from '@/shared/pilot';
import {
  ChipGroup,
  EmptyState,
  Icon,
  type IconName,
  IconCircle,
  InfoNote,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SearchField,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';
import { isMapEnabled, type MapPin, NeshanMap } from '@/shared/ui/map';

const TYPE_ICON: Record<string, IconName> = {
  gyn: 'female',
  lab: 'flask',
  imaging: 'image',
  hospital: 'bed',
  pharmacy: 'pill',
  pediatric: 'sprout',
  other: 'mapPin',
};

/** Typed text reaches the API after a short pause (one request per pause, not per key). */
function useDebounced(value: string, ms: number): string {
  const [out, setOut] = useState(value);
  useEffect(() => {
    const id = window.setTimeout(() => setOut(value), ms);
    return () => window.clearTimeout(id);
  }, [value, ms]);
  return out;
}

function knownType(type: string): CentreType {
  return (CENTRE_TYPES as readonly string[]).includes(type) ? (type as CentreType) : 'other';
}

function CentreCard({
  centre,
  selected,
  onLocate,
}: {
  centre: InsuranceCentre;
  selected: boolean;
  onLocate: () => void;
}) {
  const t = useTranslations('insurance.centers');
  const type = knownType(centre.type);
  const meta = [t(`types.${type}`), centre.services, centre.city].filter(Boolean).join(' · ');
  const hasPoint = centre.lat !== null && centre.lng !== null;
  return (
    <li id={`ins-centre-${centre.id}`} className={selected ? 'ins-centre is-selected' : 'ins-centre'}>
      <IconCircle icon={TYPE_ICON[type] ?? 'mapPin'} tone="data" size="lg" />
      <div className="ins-centre-body">
        <h3 className="ins-centre-name">{centre.name}</h3>
        <p className="ins-centre-meta">{meta}</p>
        {centre.insurerName ? <p className="ins-centre-meta">{t('insurer', { name: centre.insurerName })}</p> : null}
        <StatusPill tone={centre.isDirect ? 'data' : 'period'} className="ins-centre-pill">
          {centre.isDirect ? t('direct') : t('reimbursed')}
        </StatusPill>
      </div>
      <div className="ins-centre-actions">
        {centre.phone ? (
          <a className="ins-centre-btn" href={`tel:${centre.phone}`} aria-label={t('call', { name: centre.name })}>
            <Icon name="phone" size={18} />
          </a>
        ) : null}
        {hasPoint ? (
          <button type="button" className="ins-centre-btn" aria-label={t('locate', { name: centre.name })} onClick={onLocate}>
            <Icon name="mapPin" size={18} />
          </button>
        ) : null}
      </div>
    </li>
  );
}

/**
 * Contracted centres (CB-INS-06, nbl_Ins_Centers) at `/insurance/centers`: search, type chips, a Neshan map when a
 * key exists (the list alone otherwise) and the centre cards with the direct / reimbursed badge. A back-header
 * screen, no bottom nav. Nothing here reads or sends her own location.
 */
export function InsuranceCentersPage() {
  const t = useTranslations('insurance');
  const tc = useTranslations('insurance.centers');
  const router = useRouter();
  const [text, setText] = useState('');
  const [type, setType] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [mapFailed, setMapFailed] = useState(false);
  const q = useDebounced(text.trim(), 300);
  const query = useCentres(type, q);
  const centres = useMemo(() => query.data ?? [], [query.data]);

  const pins = useMemo<MapPin[]>(
    () =>
      centres.flatMap((c) =>
        c.lat !== null && c.lng !== null ? [{ id: String(c.id), label: c.name, lat: c.lat, lng: c.lng }] : [],
      ),
    [centres],
  );
  const showMap = isMapEnabled() && !mapFailed && pins.length > 0;

  const select = (id: string | null) => {
    setSelectedId(id);
    if (!id) return;
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    document.getElementById(`ins-centre-${id}`)?.scrollIntoView({ block: 'nearest', behavior: reduce ? 'auto' : 'smooth' });
  };

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={tc('loading')} className="ins-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="mapPin"
        title={tc('error')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (centres.length === 0) {
    body = <EmptyState icon="mapPin" title={tc('empty')} body={tc('emptyHint')} />;
  } else {
    body = (
      <>
        {showMap ? (
          <div className="ins-map">
            <NeshanMap
              ariaLabel={tc('mapLabel')}
              pins={pins}
              selectedId={selectedId}
              onSelect={select}
              onUnavailable={() => setMapFailed(true)}
              className="ins-map-view"
            />
          </div>
        ) : null}
        <ul className="nb-card ins-centres" aria-label={tc('listLabel')}>
          {centres.map((c) => (
            <CentreCard
              key={c.id}
              centre={c}
              selected={selectedId === String(c.id)}
              onLocate={() => select(selectedId === String(c.id) ? null : String(c.id))}
            />
          ))}
        </ul>
        <InfoNote icon="info">{tc('note')}</InfoNote>
      </>
    );
  }

  return (
    <div className="view ins-page">
      <SkyLayer />
      <div className="scroll ins-scroll">
        <ScreenHeader title={tc('title')} onBack={() => router.push('/insurance')} backLabel={t('back')} />
        <SearchField
          value={text}
          onValueChange={setText}
          label={tc('searchLabel')}
          clearLabel={tc('clear')}
          placeholder={tc('searchPlaceholder')}
          maxLength={80}
          {...pilot('ins-centers.search', 'text')}
        />
        <ChipGroup label={tc('typesLabel')} pilot={pilot('ins-centers.type', 'choice')}>
          {CENTRE_TYPES.filter((x) => x !== 'other').map((x) => (
            <PillChip key={x} pressed={type === x} onPressedChange={(on) => setType(on ? x : '')} {...pilotOption(x)}>
              {tc(`types.${x}`)}
            </PillChip>
          ))}
        </ChipGroup>
        {body}
      </div>
    </div>
  );
}
