'use client';

import { useCallback, useEffect, useRef, useState, type SyntheticEvent } from 'react';

import { percentAt, useSaveLessonProgress } from '@/entities/course';

/** Seconds between two silent progress saves while playing. */
export const PROGRESS_EVERY_MS = 15_000;
export const SPEEDS = [1, 1.25, 1.5, 2, 0.75] as const;
export const SLEEP_MINUTES = [0, 15, 30, 60] as const;

/** Next value in a cycle of options (speed / sleep timer buttons). */
export function nextInCycle<T>(list: readonly T[], current: T): T {
  const i = list.indexOf(current);
  return list[(i + 1) % list.length] as T;
}

/** Whether a silent save is due (`lastAt` = time of the previous save, ms). */
export function saveDue(now: number, lastAt: number): boolean {
  return now - lastAt >= PROGRESS_EVERY_MS;
}

interface Options {
  lessonId: number;
  /** Saved position to resume from (ignored when completed or near the end). */
  resumeAt: number;
  completed: boolean;
  /** The signed URL lapsed or failed: mint a new one. */
  onUrlError: () => void;
}

/**
 * Shared state of the native `<video>` / `<audio>` element: play/pause, time,
 * speed, sleep timer, resume from the saved position and throttled
 * `PUT /progress` (every 15 s while playing, on pause, on hide, on leave; the
 * end marks the lesson done).
 */
export function useLessonMedia({ lessonId, resumeAt, completed, onUrlError }: Options) {
  const ref = useRef<HTMLMediaElement | null>(null);
  const save = useSaveLessonProgress();
  const saveRef = useRef(save.mutate);
  saveRef.current = save.mutate;
  const lastSave = useRef(0);
  const resumeRef = useRef(completed ? 0 : resumeAt);
  const [playing, setPlaying] = useState(false);
  const [time, setTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [rate, setRate] = useState<number>(1);
  const [sleepMin, setSleepMin] = useState<number>(0);

  const report = useCallback(
    (extra?: { completed?: boolean; silent?: boolean }) => {
      const el = ref.current;
      if (!el || !Number.isFinite(el.currentTime)) return;
      lastSave.current = Date.now();
      const d = el.duration;
      saveRef.current({
        lessonId,
        positionSeconds: el.currentTime,
        percent: extra?.completed ? 100 : percentAt(el.currentTime, d),
        completed: extra?.completed,
        silent: extra?.silent ?? true,
      });
    },
    [lessonId],
  );

  // Save once more when the screen is hidden or left.
  useEffect(() => {
    // The element at leave time is the one to save (not a snapshot from mount).
    const holder = ref;
    const onHide = () => {
      if (document.visibilityState === 'hidden' && ref.current && !ref.current.paused) report();
    };
    document.addEventListener('visibilitychange', onHide);
    return () => {
      document.removeEventListener('visibilitychange', onHide);
      const el = holder.current;
      if (el && el.currentTime > 0 && !el.ended) report({ silent: false });
    };
  }, [report]);

  // Sleep timer: pause after N minutes.
  useEffect(() => {
    if (!sleepMin) return;
    const id = window.setTimeout(() => {
      ref.current?.pause();
      setSleepMin(0);
    }, sleepMin * 60_000);
    return () => window.clearTimeout(id);
  }, [sleepMin]);

  const handlers = {
    onLoadedMetadata: (e: SyntheticEvent<HTMLMediaElement>) => {
      const el = e.currentTarget;
      setDuration(Number.isFinite(el.duration) ? el.duration : 0);
      el.playbackRate = rate;
      const at = resumeRef.current;
      if (at > 0 && (!Number.isFinite(el.duration) || at < el.duration - 5)) el.currentTime = at;
    },
    onTimeUpdate: (e: SyntheticEvent<HTMLMediaElement>) => {
      const el = e.currentTarget;
      setTime(el.currentTime);
      resumeRef.current = el.currentTime;
      if (!el.paused && saveDue(Date.now(), lastSave.current)) report();
    },
    onPlay: () => setPlaying(true),
    onPause: (e: SyntheticEvent<HTMLMediaElement>) => {
      setPlaying(false);
      if (!e.currentTarget.ended) report();
    },
    onEnded: () => {
      setPlaying(false);
      report({ completed: true, silent: false });
    },
    onError: () => {
      setPlaying(false);
      onUrlError();
    },
  };

  const toggle = () => {
    const el = ref.current;
    if (!el) return;
    if (el.paused) void el.play().catch(() => setPlaying(false));
    else el.pause();
  };
  const seek = (to: number) => {
    const el = ref.current;
    if (!el) return;
    el.currentTime = Math.max(0, Math.min(Number.isFinite(el.duration) ? el.duration : to, to));
    setTime(el.currentTime);
  };
  const skip = (by: number) => seek((ref.current?.currentTime ?? 0) + by);
  const cycleRate = () => {
    const next = nextInCycle(SPEEDS, rate as (typeof SPEEDS)[number]);
    setRate(next);
    if (ref.current) ref.current.playbackRate = next;
  };
  const cycleSleep = () => setSleepMin((m) => nextInCycle(SLEEP_MINUTES, m as (typeof SLEEP_MINUTES)[number]));
  const finish = () => report({ completed: true, silent: false });

  return { ref, handlers, playing, time, duration, rate, sleepMin, toggle, seek, skip, cycleRate, cycleSleep, finish, saving: save.isPending };
}
