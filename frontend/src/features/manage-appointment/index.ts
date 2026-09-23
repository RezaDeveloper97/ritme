// Public API of the `manage-appointment` feature (CLAUDE.md §3.3).
export {
  useCancelAppointment,
  useCreateAppointment,
  useDeleteAppointment,
  useTogglePrepItem,
  useUpdateAppointment,
  type TogglePrepItemVars,
  type UpdateAppointmentVars,
} from './api/mutations';
export {
  setPrepItemDone,
  toAppointmentBody,
  type AppointmentInput,
  type AppointmentPatch,
} from './model/body';
