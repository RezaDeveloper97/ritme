// Public API of the `manage-medication` feature (CLAUDE.md §3.3).
export {
  useCreateMedication,
  useDeleteMedication,
  useToggleMedicationActive,
  useUpdateMedication,
  type ToggleMedicationVars,
  type UpdateMedicationVars,
  type ForUser,
} from './api/mutations';
export { toMedicationBody, withForUser, type MedicationInput, type MedicationPatch } from './model/body';
// B-N4-06: an owner's record opened by a companion (`?for=` → `for_user_id`).
export { delegatedMedicationKeys, fetchMedicationFor, useMedicationFor } from './api/queries';
