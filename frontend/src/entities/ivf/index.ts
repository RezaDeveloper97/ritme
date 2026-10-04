// Public API of the `ivf` entity (CB-IVF-02 on the CB-IVF-01 API). Import only from here (CLAUDE.md §3.3).

export {
  IVF_ROLES,
  IVF_ROUTES,
  IVF_STAGES,
  type IvfAppointmentKind,
  type IvfCompanion,
  type IvfCycle,
  type IvfCyclePatch,
  type IvfCycleStartInput,
  type IvfDose,
  type IvfDoseDay,
  type IvfDoseInput,
  type IvfHome,
  type IvfNextAppointment,
  type IvfProtocol,
  type IvfRole,
  type IvfRoute,
  type IvfStage,
  type IvfStageInfo,
  type IvfStepStatus,
  type IvfTimelineStep,
} from './model/types';
export { ivfKeys } from './api/keys';
export {
  ivfHomeSchema,
  ivfProtocolsSchema,
  ivfStagesSchema,
  toIvfCyclePatchBody,
  toIvfCycleStartBody,
} from './api/schema';
export {
  fetchIvfHome,
  useIvfHome,
  useIvfProtocols,
  useIvfStages,
  useLogIvfDose,
  useSetIvfCompanionNotify,
  useStartIvfCycle,
  useUnlogIvfDose,
  useUpdateIvfCycle,
} from './api/queries';

// CB-IVF-03 — the injection schedule (`/ivf/meds`): meds CRUD, doses with a site, catalog sites/presets/guidance.
export {
  IVF_INJECTED_ROUTES,
  IVF_MAX_TIMES,
  IVF_STIMULATION_ROLES,
  IVF_STOCK_UNITS,
  type IvfGuidance,
  type IvfInjectionSite,
  type IvfInventory,
  type IvfMed,
  type IvfMedInput,
  type IvfMedPreset,
  type IvfMedsView,
  type IvfSites,
  type IvfSiteUse,
  type IvfStockUnit,
  type IvfTrigger,
} from './model/meds';
export {
  ivfGuidanceSchema,
  ivfInjectionSitesSchema,
  ivfMedPresetsSchema,
  ivfMedsViewSchema,
  toIvfMedBody,
} from './api/meds-schema';
export {
  fetchIvfMeds,
  useAddIvfMed,
  useDeleteIvfMed,
  useIvfGuidance,
  useIvfInjectionSites,
  useIvfMedPresets,
  useIvfMeds,
  useLogIvfScheduleDose,
  useSetIvfMedActive,
  useUnlogIvfScheduleDose,
  useUpdateIvfMed,
} from './api/meds-queries';
// Scan log (CB-IVF-04).
export {
  IVF_E2_UNITS,
  IVF_FOLLICLE_BINS,
  IVF_OVARIES,
  IVF_SCAN_LIMITS,
  type IvfE2Unit,
  type IvfFollicleBin,
  type IvfGrowthPoint,
  type IvfOvary,
  type IvfOvaryCounts,
  type IvfScan,
  type IvfScanInput,
  type IvfScansView,
} from './model/scan';
export {
  ivfScanErrorMessage,
  ivfScansKey,
  ivfScansSchema,
  useDeleteIvfScan,
  useIvfScans,
  useSaveIvfScan,
} from './api/scans';

// CB-IVF-05 — the two-week wait (`/ivf/tww`): daily mood, luteal support, danger signs, the cycle outcome.
export {
  IVF_OUTCOMES,
  IVF_TWW_MOODS,
  type IvfDangerSign,
  type IvfLutealMed,
  type IvfNextStep,
  type IvfOutcome,
  type IvfOutcomeResult,
  type IvfTww,
  type IvfTwwMood,
} from './model/tww';
export { ivfDangerSignsSchema, ivfOutcomeSchema, ivfTwwSchema } from './api/tww-schema';
export {
  fetchIvfTww,
  ivfTwwKeys,
  useIvfDangerSigns,
  useIvfTww,
  useRecordIvfOutcome,
  useSetIvfTwwMood,
} from './api/tww';

// CB-IVF-06b — the first 422 field message of any IVF write (the cycle setup / editor reuse the scan helper).
export { ivfScanErrorMessage as ivfValidationMessage } from './api/scans';
