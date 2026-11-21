'use client';

import { useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { Icon, SecondaryButton } from '@/shared/ui';
import { isMapEnabled, type MapBbox, type MapPoint, NeshanMap } from '@/shared/ui/map';

import { foldDigits } from '../model/join';
import '@/shared/ui/styles/cjn.css';

const round6 = (n: number): number => Math.round(n * 1e6) / 1e6;
const parseCoord = (raw: string, limit: number): number | null => {
  const n = Number(foldDigits(raw).replace('٫', '.'));
  return raw.trim() !== '' && Number.isFinite(n) && Math.abs(n) <= limit ? n : null;
};

/**
 * The venue pin (nbl_Dir_JoinForm «مکان روی نقشه»). With a Neshan key the map opens on the current pin and the
 * «pin here» pill sets it to the centre of the view; without one (dev, blocked SDK) it falls back to the device
 * location button and typed coordinates. The device location is read only on tap and goes to the API as the pin
 * the owner confirms — never anywhere else.
 */
export function PinPicker({ value, onChange }: { value: MapPoint | null; onChange: (p: MapPoint | null) => void }) {
  const t = useTranslations('cityJoin.place');
  const [mapOk, setMapOk] = useState(isMapEnabled());
  const [locating, setLocating] = useState(false);
  const [locateFailed, setLocateFailed] = useState(false);
  const [lat, setLat] = useState(value ? String(value.lat) : '');
  const [lng, setLng] = useState(value ? String(value.lng) : '');
  const pins = useMemo(() => (value ? [{ id: 'pin', label: t('pinSet'), lat: value.lat, lng: value.lng }] : []), [value, t]);

  const set = (p: MapPoint) => {
    const point = { lat: round6(p.lat), lng: round6(p.lng) };
    setLat(String(point.lat));
    setLng(String(point.lng));
    setLocateFailed(false);
    onChange(point);
  };
  const onBbox = (b: MapBbox) => set({ lat: (b.south + b.north) / 2, lng: (b.west + b.east) / 2 });
  const typed = (nextLat: string, nextLng: string) => {
    setLat(nextLat);
    setLng(nextLng);
    const la = parseCoord(nextLat, 90);
    const ln = parseCoord(nextLng, 180);
    if (la !== null && ln !== null) onChange({ lat: la, lng: ln });
    else if (nextLat.trim() === '' && nextLng.trim() === '') onChange(null);
  };
  const locate = () => {
    if (!navigator.geolocation) {
      setLocateFailed(true);
      return;
    }
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        setLocating(false);
        set({ lat: pos.coords.latitude, lng: pos.coords.longitude });
      },
      () => {
        setLocating(false);
        setLocateFailed(true);
      },
      { enableHighAccuracy: true, timeout: 10_000 },
    );
  };

  return (
    <div className="cjn-pin">
      <div className="cjn-head">
        <div>
          <h3 className="cjn-label">{t('mapTitle')}</h3>
        </div>
        <span className="cjn-chip" data-set={value !== null}>
          <Icon name={value ? 'check' : 'mapPin'} size={14} />
          {value ? t('pinSet') : t('pinNone')}
        </span>
      </div>

      {mapOk ? (
        <div className="cjn-map">
          <NeshanMap
            ariaLabel={t('mapLabel')}
            pins={pins}
            center={value ?? undefined}
            zoom={value ? 16 : 12}
            searchAreaLabel={t('setPin')}
            onSearchArea={onBbox}
            myLocationLabel={t('myLocation')}
            onLocate={set}
            onLocateError={() => setLocateFailed(true)}
            onUnavailable={() => setMapOk(false)}
            className="cjn-map-fill"
          />
          <p className="cjn-hint">{t('mapDrag')}</p>
        </div>
      ) : null}

      <SecondaryButton icon="mapPin" loading={locating} onClick={locate}>
        {t('locate')}
      </SecondaryButton>
      {locateFailed ? (
        <p role="alert" className="cjn-err">
          {t('locateError')}
        </p>
      ) : null}

      <details className="cjn-details" open={!mapOk}>
        <summary>{t('coords')}</summary>
        <div className="cjn-row">
          <label className="fld-label">
            <span className="fld-label-t">{t('lat')}</span>
            <span className="field">
              <input type="text" inputMode="decimal" dir="ltr" value={lat} onChange={(e) => typed(e.target.value, lng)} />
            </span>
          </label>
          <label className="fld-label">
            <span className="fld-label-t">{t('lng')}</span>
            <span className="field">
              <input type="text" inputMode="decimal" dir="ltr" value={lng} onChange={(e) => typed(lat, e.target.value)} />
            </span>
          </label>
        </div>
      </details>
    </div>
  );
}
