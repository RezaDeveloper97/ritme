// Public API of entities/instructor.
export {
  instructorSchema,
  instructorStatusSchema,
  meSchema,
  APPLY_LIMITS,
} from './model/schema';
export type { Instructor, InstructorStatus, Me, ApplyInput } from './model/schema';
export { accessOf, pathForAccess, initialOf } from './model/access';
export type { InstructorAccess } from './model/access';
export { instructorKeys, fetchMe, applyInstructor } from './api/instructor-api';
export { useInstructorMe } from './api/queries';
