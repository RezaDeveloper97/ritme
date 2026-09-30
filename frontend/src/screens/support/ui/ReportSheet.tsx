'use client';

import { useTranslations } from 'next-intl';
import { useId, useRef, useState } from 'react';

import { AppSheet } from '@/shared/sheet';
import { Icon, PrimaryButton, SecondaryButton } from '@/shared/ui';

import { useSendReport } from '../api/support';
import { isReportValid, REPORT_MAX, SCREENSHOT_MAX_BYTES } from '../model/support';

function readDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result));
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}

/**
 * «گزارش مشکل» (B-N1-12): a message and an optional screenshot, sent to
 * POST /support/reports. The server re-encodes the image and keeps it on
 * private storage; nothing goes to a third party.
 */
export function ReportSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useTranslations('me.support.reportSheet');
  const send = useSendReport();
  const [message, setMessage] = useState('');
  const [shot, setShot] = useState<{ name: string; url: string } | null>(null);
  const [shotError, setShotError] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);
  const ids = useId();

  const close = () => {
    if (send.isPending) return;
    setMessage('');
    setShot(null);
    setShotError(false);
    send.reset();
    onClose();
  };

  const pick = async (file: File | undefined) => {
    setShotError(false);
    if (!file) return;
    if (file.size > SCREENSHOT_MAX_BYTES) {
      setShotError(true);
      return;
    }
    setShot({ name: file.name, url: await readDataUrl(file) });
  };

  const submit = () => {
    if (!isReportValid(message) || send.isPending) return;
    send.mutate({ message, screenshot: shot?.url });
  };

  if (send.isSuccess) {
    return (
      <AppSheet open={open} onClose={close} size="half" title={t('title')}>
        <div className="rep-done">
          <Icon name="checkCircle" size={40} className="rep-done-icon" />
          <p className="rep-done-title">{t('sent')}</p>
          <p className="rep-done-body">{t('sentBody')}</p>
          <PrimaryButton onClick={close}>{t('done')}</PrimaryButton>
        </div>
      </AppSheet>
    );
  }

  return (
    <AppSheet
      open={open}
      onClose={close}
      size="full"
      title={t('title')}
      footer={
        <PrimaryButton loading={send.isPending} disabled={!isReportValid(message)} onClick={submit}>
          {t('send')}
        </PrimaryButton>
      }
    >
      <div className="rep-body">
        <label htmlFor={`${ids}-msg`} className="rep-label">
          {t('label')}
        </label>
        <textarea
          id={`${ids}-msg`}
          className="rep-text"
          rows={6}
          maxLength={REPORT_MAX}
          value={message}
          placeholder={t('placeholder')}
          aria-describedby={`${ids}-hint`}
          onChange={(e) => setMessage(e.target.value)}
        />
        <p id={`${ids}-hint`} className="rep-hint">
          {t('hint')}
        </p>

        <input
          ref={fileRef}
          type="file"
          accept="image/png,image/jpeg,image/webp"
          className="rep-file"
          tabIndex={-1}
          aria-hidden
          onChange={(e) => {
            void pick(e.target.files?.[0]);
            e.target.value = '';
          }}
        />
        {shot ? (
          <div className="rep-shot">
            {/* A local preview of the user's own file (data URL), never a remote image. */}
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={shot.url} alt="" className="rep-shot-img" />
            <span className="rep-shot-name">{shot.name}</span>
            <button type="button" className="rep-shot-remove" onClick={() => setShot(null)} aria-label={t('remove')}>
              <Icon name="x" size={18} />
            </button>
          </div>
        ) : null}
        <SecondaryButton icon="camera" onClick={() => fileRef.current?.click()}>
          {shot ? t('replace') : t('attach')}
        </SecondaryButton>
        {shotError ? (
          <p className="rep-error" role="alert">
            {t('tooLarge')}
          </p>
        ) : null}
        {send.isError ? (
          <p className="rep-error" role="alert">
            {t('error')}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}
