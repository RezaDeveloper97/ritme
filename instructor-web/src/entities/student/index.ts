// Public API of entities/student.
export { grantSchema, groupSchema, groupDetailSchema, GROUP_NAME_MAX } from './model/schema';
export type { Grant, GrantStatus, Group, GroupDetail, GroupInput } from './model/schema';
export { phoneLabel, initialOf, targetOf } from './model/display';
export {
  studentKeys,
  fetchStudents,
  fetchGroups,
  fetchGroup,
  createGroup,
  updateGroup,
  deleteGroup,
  revokeGrant,
} from './api/student-api';
export { useStudents, useGroups, useGroup, useRevokeGrant } from './api/queries';
export { GrantRow } from './ui/GrantRow';
export { GrantStatusPill, progressText } from './ui/GrantStatus';
export { RevokeButton } from './ui/RevokeButton';
