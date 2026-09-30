'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { AppSheet } from '@/shared/sheet';
import { SegmentedTabs } from '@/shared/ui';

import { PASSCODE_MAX, PASSCODE_MIN } from '../model/passcode';
import { getLockController } from '../model/store';
import { PasscodePad } from './PasscodePad';

type Mode = 'create' | 'verify';

interface PasscodeSheetProps {
  open: boolean;
  /** `create`: choose + confirm a new passcode, then the lock is on. `verify`: prove the current one. */
  mode: Mode;
  onClose: () => void;
  /** `create`: the lock is now on. `verify`: the passcode matched. */
  onDone: () => void;
}

const LENGTHS = [String(PASSCODE_MIN), String(PASSCODE_MAX)] as const;

/**
 * The passcode sheet of the Privacy screen: set a new 4- or 6-digit passcode
 * (entered twice), or verify the current one before the lock is turned off.
 * The passcode is hashed on the device (PBKDF2) and never sent anywhere.
 */
export function PasscodeSheet({ open, mode, onClose, onDone }: PasscodeSheetProps) {
  const t = useTranslations('common.appLock');
  const [length, setLength] = useState<(typeof LENGTHS)[number]>(LENGTHS[0]);
  const [first, setFirst] = useState<string | null>(null);
  const [value, setValue] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const reset = () => {
    setFirst(null);
    setValue('');
    setError(null);
    setBusy(false);
  };
  const close = () => {
    reset();
    onClose();
  };

  const verifyLength = getLockController()?.getSnapshot().length ?? PASSCODE_MIN;

  const complete = async (code: string) => {
    const c = getLockController();
    if (!c || busy) return;
    if (mode === 'verify') {
      setBusy(true);
      const ok = await c.verifyPasscode(code);
      setBusy(false);
      setValue('');
      if (ok) {
        reset();
        onDone();
      } else setError(t('wrong'));
      return;
    }
    if (first === null) {
      setFirst(code);
      setValue('');
      return;
    }
    if (code !== first) {
      setFirst(null);
      setValue('');
      setError(t('mismatch'));
      return;
    }
    setBusy(true);
    await c.enable({ passcode: code });
    reset();
    onDone();
  };

  const heading =
    mode === 'verify' ? t('verifyTitle') : first === null ? t('createTitle') : t('confirmTitle');

  return (
    <AppSheet open={open} onClose={close} size="half" title={t('sheetTitle')} className="lk-sheet">
      <div className="lk-sheet-body">
        <h2 id="lk-sheet-h" className="lk-sheet-h">
          {heading}
        </h2>
        {mode === 'create' && first === null ? (
          <SegmentedTabs
            label={t('lengthLabel')}
            tabs={LENGTHS.map((l) => ({ value: l, label: t('digits', { count: Number(l) }) }))}
            value={length}
            onChange={(v) => {
              setLength(v);
              setValue('');
            }}
            className="lk-length"
          />
        ) : null}
        <PasscodePad
          labelledBy="lk-sheet-h"
          length={mode === 'verify' ? verifyLength : first?.length ?? Number(length)}
          value={value}
          onChange={(v) => {
            setValue(v);
            if (v) setError(null);
          }}
          onComplete={(v) => void complete(v)}
          disabled={busy}
          error={error}
        />
        <p className="lk-sheet-note">{t('localNote')}</p>
      </div>
    </AppSheet>
  );
}
