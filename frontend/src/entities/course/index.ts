// Public API of the `course` entity (B-N8-03 on /api/v1/learning). Import only from here (CLAUDE.md §3.3).

export {
  LESSON_KINDS,
  type ContinueCard,
  type CourseAccess,
  type CourseChapter,
  type CourseFilter,
  type CourseOutline,
  type InstructorRef,
  type LessonBlock,
  type LessonDetail,
  type LessonKind,
  type LessonProgress,
  type MyCourses,
  type Playback,
  type StudentCourse,
  type StudentLesson,
  type UnlockedGrant,
} from './model/types';
export {
  KIND_LOOK,
  filterCourses,
  formatClock,
  groupByInstructor,
  minutesOf,
  newCourses,
  percentAt,
  type InstructorGroup,
} from './model/format';
export { courseKeys } from './api/keys';
export {
  lessonBlock,
  useCourseOutline,
  useLesson,
  useMyCourses,
  usePlayback,
  useSaveLessonProgress,
  useUnlockedGrant,
  type ProgressInput,
} from './api/queries';
export { CourseCard, KindThumb, useAccessLabel, useCourseMeta, useLessonMeta } from './ui/CourseBits';
