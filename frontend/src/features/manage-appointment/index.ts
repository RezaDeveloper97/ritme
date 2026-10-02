// Public API of the `manage-appointment` feature (CLAUDE.md §3.3).
export {
  useCancelAppointment,
  useCreateAppointment,
  useDeleteAppointment,
  useTogglePrepItem,
  useUpdateAppointment,
  type TogglePrepItemVars,
  type UpdateAppointmentVars,
  type ForUser,
} from './api/mutations';
export {
  setPrepItemDone,
  toAppointmentBody,
  withForUser,
  type AppointmentInput,
  type AppointmentPatch,
} from './model/body';
// B-N4-06: an owner's record opened by a companion (`?for=` → `for_user_id`).
export { delegatedAppointmentKeys, fetchAppointmentFor, useAppointmentFor } from './api/queries';
