// Public API of the `care-reminder` entity (M3). Import only from here (CLAUDE.md §3.3).

// ── Model: types & enums ───────────────────────────────────────
export {
  ALL_WEEKDAYS,
  APPOINTMENT_KINDS,
  APPOINTMENT_STATUSES,
  APPOINTMENT_TOPICS,
  MEDICATION_DURATIONS,
  MEDICATION_FORMS,
  MEDICATION_UNITS,
  REMIND_BEFORE,
  type Appointment,
  type AppointmentKind,
  type AppointmentScope,
  type AppointmentStatus,
  type AppointmentTopic,
  type CareEnums,
  type CareOption,
  type CareToday,
  type Medication,
  type MedicationDuration,
  type MedicationForm,
  type MedicationUnit,
  type NextAppointment,
  type PrepItem,
  type RemindBefore,
  type TodayDose,
  type Weekday,
} from './model/types';
export { applyIntake, type IntakeChange } from './model/intake';
export { medicationSchedule, type MedicationSchedule } from './model/schedule';

// ── API: keys, parsers, reads ──────────────────────────────────
export { careKeys, type MedicationFilters } from './api/keys';
export { appointmentSchema, careTodaySchema, medicationSchema } from './api/schema';
export {
  fetchAppointment,
  fetchAppointments,
  fetchCareEnums,
  fetchCareToday,
  fetchMedication,
  fetchMedications,
  useAppointment,
  useAppointments,
  useCareEnums,
  useCareToday,
  useMedication,
  useMedications,
} from './api/queries';
