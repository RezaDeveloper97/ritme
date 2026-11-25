'use client';

import { useSearchParams } from 'next/navigation';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState, type ReactNode } from 'react';

import {
  KindThumb,
  formatClock,
  lessonBlock,
  useCourseOutline,
  useLesson,
  useLessonMeta,
  usePlayback,
  useLessonDownload,
  useSaveLessonProgress,
  type CourseOutline,
  type LessonDetail,
  type StudentLesson,
} from '@/entities/course';
import { pilot } from '@/shared/pilot';
import { Link, useDirection, useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { useOfflineBlobUrl } from '@/shared/pwa/offline';
import { Card, EmptyState, HeaderButton, Icon, PrimaryButton, ScreenHeader, Skeleton, SkeletonGroup } from '@/shared/ui';

import { useLessonMedia } from '../model/use-media';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/lrn.css';

type Media = ReturnType<typeof useLessonMedia>;

/**
 * `/learn/[course]/[lesson]` — Learn_Video / Learn_Audio / Learn_PDF. The
 * media comes from a 30-min signed URL (B-N8-02, Range-capable) played by the
 * native element; position is restored and saved (throttled). `?audio=1`
 * plays a video lesson in the audio layout («فقط صدا»). 403s render the
 * locked / expired states.
 */
export function LearnLessonPage({ courseId, lessonId }: { courseId: number; lessonId: number }) {
  const t = useTranslations('learn');
  const router = useRouter();
  const params = useSearchParams();
  const q = useLesson(lessonId);
  const outline = useCourseOutline(courseId);
  const [audioOnly, setAudioOnly] = useState(params.get('audio') === '1');
  const back = () => router.back(`/learn/${courseId}`);

  if (q.isPending) {
    return (
      <Plain title="" onBack={back} backLabel={t('common.back')}>
        <SkeletonGroup label={t('common.loading')} className="lrn-body">
          <Skeleton shape="card" />
          <Skeleton width="medium" />
          <Skeleton shape="block" />
        </SkeletonGroup>
      </Plain>
    );
  }
  if (q.isError) {
    const block = lessonBlock(q.error);
    return (
      <Plain title="" onBack={back} backLabel={t('common.back')}>
        <Card className="lrn-body">
          <EmptyState
            icon={block ? 'lock' : 'warning'}
            className={block === 'expired' ? 'nb-tone-danger' : undefined}
            title={t(block ? `lesson.blocked.${block}.title` : 'common.loadError')}
            body={block ? t(`lesson.blocked.${block}.body`) : undefined}
            action={
              block ? (
                <PrimaryButton onClick={back}>{t('lesson.backToCourse')}</PrimaryButton>
              ) : (
                <PrimaryButton icon="refresh" loading={q.isFetching} onClick={() => void q.refetch()}>
                  {t('common.retry')}
                </PrimaryButton>
              )
            }
          />
        </Card>
      </Plain>
    );
  }

  const d = q.data;
  const kind = d.lesson.kind;
  if (kind === 'pdf') return <PdfLesson detail={d} onBack={back} />;
  if (kind === 'audio' || audioOnly) {
    return <AudioLesson detail={d} outline={outline.data} onBack={back} onVideo={kind === 'video' ? () => setAudioOnly(false) : undefined} />;
  }
  return <VideoLesson detail={d} outline={outline.data} onBack={back} onAudio={() => setAudioOnly(true)} />;
}

function Plain({ title, onBack, backLabel, children }: { title: string; onBack: () => void; backLabel: string; children: ReactNode }) {
  return (
    <div className="view lrn-view">
      <div className="scroll">
        <ScreenHeader title={title} onBack={onBack} backLabel={backLabel} />
        {children}
      </div>
    </div>
  );
}

/**
 * The signed URL of a ready lesson, plus the «not ready» / failure states. A
 * downloaded lesson (B-N8-04) plays from its decrypted local copy instead.
 */
function useLessonUrl(lesson: StudentLesson) {
  const local = useOfflineBlobUrl(lesson.id);
  const ready = lesson.media?.status === 'ready';
  const pb = usePlayback(lesson.id, ready && local.checked && !local.url);
  return { ready: ready || !!local.url, pb, url: local.url ?? pb.data?.url ?? null, offline: !!local.url };
}

/** «دانلود» pill (B-N8-04): download for offline / progress + cancel / remove. */
function DownloadPill({ detail }: { detail: LessonDetail }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const { lesson, course } = detail;
  const dl = useLessonDownload({ lessonId: lesson.id, courseId: course.id, courseTitle: course.title, title: lesson.title, kind: lesson.kind });
  if (dl.state === 'unsupported' || dl.state === 'checking' || lesson.media?.status !== 'ready') return null;
  return (
    <>
      {dl.state === 'downloading' ? (
        <button type="button" className="lrn-pill" aria-busy onClick={dl.cancel} aria-label={t('download.cancel')}>
          <Icon name="x" size={16} />
          {dl.percent != null ? t('download.progress', { percent: formatNumber(dl.percent, locale) }) : t('download.downloading')}
        </button>
      ) : dl.state === 'saved' ? (
        <button type="button" className="lrn-pill is-saved" onClick={() => void dl.remove()} aria-label={t('download.remove')}>
          <Icon name="check" size={16} />
          {t('download.saved')}
        </button>
      ) : (
        <button type="button" className="lrn-pill" onClick={() => void dl.start()}>
          <Icon name="download" size={16} />
          {t('download.action')}
        </button>
      )}
      {dl.error ? (
        <p className="lrn-dl-error" role="alert">
          {t(`download.errors.${dl.error}`)}
        </p>
      ) : null}
    </>
  );
}

function MediaUnavailable({ ready, failed, onRetry }: { ready: boolean; failed: boolean; onRetry: () => void }) {
  const t = useTranslations('learn');
  if (!ready) return <p className="lrn-media-msg">{t('lesson.notReady')}</p>;
  if (failed) {
    return (
      <div className="lrn-media-msg">
        <p>{t('lesson.mediaError')}</p>
        <button type="button" className="lrn-pill" onClick={onRetry}>
          <Icon name="refresh" size={16} />
          {t('common.retry')}
        </button>
      </div>
    );
  }
  return <span className="lrn-media-msg" aria-busy>{t('common.loading')}</span>;
}

function lessonPosition(d: LessonDetail) {
  return { resumeAt: d.lesson.progress?.positionSeconds ?? 0, completed: d.lesson.progress?.completed ?? false };
}

function Scrubber({ media, tone }: { media: Media; tone: 'video' | 'audio' }) {
  const t = useTranslations('learn');
  return (
    <input
      type="range"
      className={`lrn-scrub is-${tone}`}
      min={0}
      max={Math.max(1, Math.floor(media.duration))}
      step={1}
      value={Math.floor(media.time)}
      onChange={(e) => media.seek(Number(e.currentTarget.value))}
      aria-label={t('player.seek')}
      {...pilot('lesson.seek', 'text')}
      aria-valuetext={formatClock(media.time)}
    />
  );
}

function PlayControls({ media, big }: { media: Media; big?: boolean }) {
  const t = useTranslations('learn');
  const rtl = useDirection() === 'rtl';
  return (
    <div className={big ? 'lrn-ctrls is-big' : 'lrn-ctrls'}>
      <button type="button" className="lrn-ctrl" onClick={() => media.skip(-15)} {...pilot('lesson.skip-back')} aria-label={t('player.back15')}>
        <Icon name={rtl ? 'forward' : 'rewind'} size={28} />
      </button>
      <button type="button" className="lrn-ctrl is-main" onClick={media.toggle} {...pilot('lesson.play')} aria-label={t(media.playing ? 'player.pause' : 'player.play')}>
        <Icon name={media.playing ? 'pause' : 'play'} size={30} fill={media.playing ? 'none' : 'currentColor'} strokeWidth={media.playing ? 3 : 2} />
      </button>
      <button type="button" className="lrn-ctrl" onClick={() => media.skip(15)} {...pilot('lesson.skip-fwd')} aria-label={t('player.fwd15')}>
        <Icon name={rtl ? 'rewind' : 'forward'} size={28} />
      </button>
    </div>
  );
}

function speedLabel(rate: number) {
  return `${rate}x`;
}

/** Next lesson card + PDF files of the course (from the outline). */
function useCourseExtras(outline: CourseOutline | undefined, lessonId: number, nextId: number | null) {
  const all = outline ? [...outline.chapters.flatMap((c) => c.lessons), ...outline.lessons] : [];
  return {
    next: nextId ? (all.find((l) => l.id === nextId) ?? null) : null,
    files: all.filter((l) => l.kind === 'pdf' && l.id !== lessonId),
  };
}

function NextCard({ lesson, courseId }: { lesson: StudentLesson; courseId: number }) {
  const t = useTranslations('learn');
  return (
    <Link href={`/learn/${courseId}/${lesson.id}`} className="lrn-next">
      <KindThumb kind={lesson.kind} size="sm" />
      <span className="lrn-row-main">
        <span className="lrn-row-meta">{t('lesson.next')}</span>
        <span className="lrn-row-title">{lesson.title}</span>
      </span>
      <Icon name="chevronLeft" size={20} className="lrn-chev" />
    </Link>
  );
}

function FileRow({ lesson, courseId }: { lesson: StudentLesson; courseId: number }) {
  const meta = useLessonMeta();
  return (
    <Link href={`/learn/${courseId}/${lesson.id}`} className="lrn-file">
      <Icon name="fileDoc" size={22} className="lrn-file-ic" />
      <span className="lrn-row-main">
        <span className="lrn-row-title">{lesson.title}</span>
        <span className="lrn-row-meta">{meta(lesson)}</span>
      </span>
    </Link>
  );
}

const NOTE_KEY = (id: number) => `ritme_learn_note_${id}`;

/** «یادداشت‌های من» — kept on this device only (never sent anywhere). */
function NotesBox({ lessonId }: { lessonId: number }) {
  const t = useTranslations('learn');
  const [text, setText] = useState('');
  useEffect(() => {
    try {
      setText(window.localStorage.getItem(NOTE_KEY(lessonId)) ?? '');
    } catch {
      /* storage blocked: start empty */
    }
  }, [lessonId]);
  const onChange = (v: string) => {
    setText(v);
    try {
      window.localStorage.setItem(NOTE_KEY(lessonId), v);
    } catch {
      /* storage blocked: keep in memory */
    }
  };
  return (
    <label className="lrn-notes">
      <span className="lrn-notes-hint">{t('lesson.notesHint')}</span>
      <textarea {...pilot('lesson.notes', 'text')} value={text} maxLength={4000} rows={5} onChange={(e) => onChange(e.currentTarget.value)} placeholder={t('lesson.notesPlaceholder')} />
    </label>
  );
}

type VTab = 'about' | 'notes' | 'files';

function VideoLesson({ detail, outline, onBack, onAudio }: { detail: LessonDetail; outline?: CourseOutline; onBack: () => void; onAudio: () => void }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const { lesson, course } = detail;
  const { ready, pb, url } = useLessonUrl(lesson);
  const media = useLessonMedia({ lessonId: lesson.id, ...lessonPosition(detail), onUrlError: () => void pb.refetch() });
  const [tab, setTab] = useState<VTab>('about');
  const extras = useCourseExtras(outline, lesson.id, detail.nextLessonId);
  const chapterIdx = outline?.chapters.findIndex((c) => c.id === lesson.chapterId) ?? -1;
  const done = lesson.progress?.completed ?? false;

  const fullscreen = () => {
    const el = media.ref.current as (HTMLVideoElement & { webkitEnterFullscreen?: () => void }) | null;
    if (!el) return;
    if (el.requestFullscreen) void el.requestFullscreen().catch(() => el.webkitEnterFullscreen?.());
    else el.webkitEnterFullscreen?.();
  };

  return (
    <div className="view lrn-view">
      <div className="scroll">
        <div className="lrn-stage">
          {url ? (
            <video
              ref={(el) => {
                media.ref.current = el;
              }}
              className="lrn-video"
              src={url}
              preload="metadata"
              playsInline
              controlsList="nodownload"
              onContextMenu={(e) => e.preventDefault()}
              onClick={media.toggle}
              {...media.handlers}
            />
          ) : (
            <MediaUnavailable ready={ready} failed={pb.isError} onRetry={() => void pb.refetch()} />
          )}
          <div className="lrn-stage-top">
            <HeaderButton label={t('common.back')} icon={rtl ? 'arrowR' : 'arrowL'} onClick={onBack} variant="soft" className="lrn-stage-btn" />
          </div>
          {url ? (
            <>
              <PlayControls media={media} />
              <div className="lrn-stage-bar">
                <span className="lrn-time">{formatClock(media.time)}</span>
                <Scrubber media={media} tone="video" />
                <span className="lrn-time">{formatClock(media.duration)}</span>
                <button type="button" className="lrn-speed" onClick={media.cycleRate} {...pilot('lesson.speed')} aria-label={t('player.speed')}>
                  {speedLabel(media.rate)}
                </button>
                <button type="button" className="lrn-stage-icon" onClick={fullscreen} aria-label={t('player.fullscreen')}>
                  <Icon name="expand" size={20} />
                </button>
              </div>
            </>
          ) : null}
        </div>

        <div className="lrn-body">
          <p className="lrn-kicker">
            {chapterIdx >= 0
              ? t('lesson.numberChapter', {
                  number: formatNumber(lesson.number, locale),
                  total: formatNumber(course.lessonsCount, locale),
                  chapter: formatNumber(chapterIdx + 1, locale),
                })
              : t('lesson.number', { number: formatNumber(lesson.number, locale), total: formatNumber(course.lessonsCount, locale) })}
          </p>
          <h1 className="lrn-lesson-title">{lesson.title}</h1>
          <p className="lrn-lesson-sub">
            {course.title} · {course.instructor.name}
          </p>
          <div className="lrn-pills">
            <button type="button" className="lrn-pill" onClick={onAudio}>
              <Icon name="music" size={16} />
              {t('player.audioOnly')}
            </button>
            <DownloadPill detail={detail} />
            <button type="button" className="lrn-pill" onClick={() => setTab('notes')}>
              <Icon name="note" size={16} />
              {t('player.note')}
            </button>
          </div>
          <div className="lrn-tabs" role="tablist" aria-label={t('lesson.tabsLabel')}>
            {(['about', 'notes', 'files'] as const).map((v) => (
              <button key={v} type="button" role="tab" aria-selected={tab === v} className="lrn-tab" onClick={() => setTab(v)} {...pilot(`lesson.tab:${v}`)}>
                {t(`lesson.tabs.${v}`)}
              </button>
            ))}
          </div>
          {tab === 'about' ? (
            <p className="lrn-desc">{lesson.description ?? t('lesson.noDescription')}</p>
          ) : tab === 'notes' ? (
            <NotesBox lessonId={lesson.id} />
          ) : extras.files.length ? (
            extras.files.map((f) => <FileRow key={f.id} lesson={f} courseId={course.id} />)
          ) : (
            <EmptyState icon="fileDoc" title={t('course.noFiles')} />
          )}
          {extras.next ? <NextCard lesson={extras.next} courseId={course.id} /> : null}
        </div>
        <div className="page-tail" />
      </div>
      <div className="lrn-footer">
        <PrimaryButton icon="check" block loading={media.saving} disabled={done || !url} onClick={media.finish} {...pilot('lesson.finish')}>
          {t(done ? 'lesson.doneAlready' : 'lesson.markDone')}
        </PrimaryButton>
      </div>
    </div>
  );
}

function AudioLesson({ detail, outline, onBack, onVideo }: { detail: LessonDetail; outline?: CourseOutline; onBack: () => void; onVideo?: () => void }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const { lesson, course } = detail;
  const { ready, pb, url } = useLessonUrl(lesson);
  const media = useLessonMedia({ lessonId: lesson.id, ...lessonPosition(detail), onUrlError: () => void pb.refetch() });
  const [transcript, setTranscript] = useState(false);
  const extras = useCourseExtras(outline, lesson.id, detail.nextLessonId);

  // Lock-screen / notification controls while it plays in the background.
  useEffect(() => {
    if (typeof navigator === 'undefined' || !('mediaSession' in navigator) || typeof MediaMetadata === 'undefined') return;
    navigator.mediaSession.metadata = new MediaMetadata({ title: lesson.title, artist: course.instructor.name, album: course.title });
  }, [lesson.title, course.instructor.name, course.title]);

  return (
    <div className="view lrn-view">
      <SkyGlow />
      <div className="scroll">
        <ScreenHeader
          title={course.title}
          onBack={onBack}
          backLabel={t('common.back')}
          action={onVideo ? <HeaderButton label={t('player.video')} icon="video" onClick={onVideo} /> : undefined}
          className="lrn-audio-hdr"
        />
        {url ? (
          <audio
            ref={(el) => {
              media.ref.current = el;
            }}
            src={url}
            preload="metadata"
            controlsList="nodownload"
            {...media.handlers}
          />
        ) : null}
        <div className="lrn-body lrn-audio">
          <div className="lrn-art" aria-hidden>
            <span className="lrn-art-ring">
              <Icon name="moon" size={44} />
            </span>
          </div>
          <h1 className="lrn-audio-title">{lesson.title}</h1>
          <p className="lrn-lesson-sub">
            {t('lesson.session', {
              name: course.instructor.name,
              number: formatNumber(lesson.number, locale),
              total: formatNumber(course.lessonsCount, locale),
            })}
          </p>
          {url ? (
            <>
              <Scrubber media={media} tone="audio" />
              <div className="lrn-audio-times">
                <span className="lrn-time">{formatClock(media.time)}</span>
                <span className="lrn-time">-{formatClock(Math.max(0, media.duration - media.time))}</span>
              </div>
              <PlayControls media={media} big />
            </>
          ) : (
            <MediaUnavailable ready={ready} failed={pb.isError} onRetry={() => void pb.refetch()} />
          )}
          <div className="lrn-pills is-center">
            <button type="button" className="lrn-pill" onClick={media.cycleRate} {...pilot('lesson.speed')} aria-label={t('player.speed')}>
              <Icon name="clock" size={16} />
              {speedLabel(media.rate)}
            </button>
            <button type="button" className="lrn-pill" aria-pressed={media.sleepMin > 0} onClick={media.cycleSleep} {...pilot('lesson.sleep')}>
              <Icon name="moon" size={16} />
              {media.sleepMin ? t('player.sleepIn', { minutes: formatNumber(media.sleepMin, locale) }) : t('player.sleep')}
            </button>
            <button type="button" className="lrn-pill" aria-expanded={transcript} onClick={() => setTranscript((v) => !v)}>
              <Icon name="filterLines" size={16} />
              {t('player.transcript')}
            </button>
            <DownloadPill detail={detail} />
          </div>
          {transcript ? (
            <Card className="lrn-transcript">
              <p>{lesson.description ?? t('lesson.noTranscript')}</p>
            </Card>
          ) : null}
          <p className="lrn-note">{t('player.background')}</p>
          {extras.next ? <NextCard lesson={extras.next} courseId={course.id} /> : null}
        </div>
        <div className="page-tail" />
      </div>
    </div>
  );
}

function SkyGlow() {
  return <span className="lrn-glow" aria-hidden />;
}

function PdfLesson({ detail, onBack }: { detail: LessonDetail; onBack: () => void }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const { lesson, course } = detail;
  const { ready, pb, url, offline } = useLessonUrl(lesson);
  const save = useSaveLessonProgress();
  const done = (lesson.progress?.completed ?? false) || save.isSuccess;
  return (
    <div className="view lrn-view">
      <div className="scroll">
        <ScreenHeader
          title={lesson.title}
          subtitle={lesson.pageCount ? t('pdf.pages', { pages: formatNumber(lesson.pageCount, locale) }) : course.title}
          onBack={onBack}
          backLabel={t('common.back')}
        />
        <div className="lrn-body">
          {url ? (
            <>
              <iframe className="lrn-pdf" src={url} title={lesson.title} />
              <div className="lrn-pills">
                {offline ? null : (
                  <a className="lrn-pill lrn-pdf-open" href={url} target="_blank" rel="noopener noreferrer">
                    <Icon name="export" size={16} />
                    {t('pdf.openTab')}
                  </a>
                )}
                <DownloadPill detail={detail} />
              </div>
            </>
          ) : (
            <MediaUnavailable ready={ready} failed={pb.isError} onRetry={() => void pb.refetch()} />
          )}
          {lesson.description ? <p className="lrn-desc">{lesson.description}</p> : null}
        </div>
        <div className="page-tail" />
      </div>
      <div className="lrn-footer">
        <PrimaryButton
          icon="check"
          block
          loading={save.isPending}
          disabled={done || !url}
          {...pilot('lesson.finish')}
          onClick={() => save.mutate({ lessonId: lesson.id, positionSeconds: 0, percent: 100, completed: true })}
        >
          {t(done ? 'lesson.doneAlready' : 'pdf.markRead')}
        </PrimaryButton>
      </div>
    </div>
  );
}
