/** Query keys of the student learning reads (B-N8-03). Progress writes invalidate `all`. */
export const courseKeys = {
  all: ['learning'] as const,
  mine: () => [...courseKeys.all, 'mine'] as const,
  course: (id: number) => [...courseKeys.all, 'course', id] as const,
  lesson: (id: number) => [...courseKeys.all, 'lesson', id] as const,
  playback: (id: number) => [...courseKeys.all, 'playback', id] as const,
  unlocked: (grant: number) => [...courseKeys.all, 'unlocked', grant] as const,
};
