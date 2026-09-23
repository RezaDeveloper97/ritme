// Public API of the `manage-medication` feature (CLAUDE.md §3.3).
export {
  useCreateMedication,
  useDeleteMedication,
  useToggleMedicationActive,
  useUpdateMedication,
  type ToggleMedicationVars,
  type UpdateMedicationVars,
} from './api/mutations';
export { toMedicationBody, type MedicationInput, type MedicationPatch } from './model/body';
