// Public API of the `ivf` entity (CB-IVF-02 on the CB-IVF-01 API). Import only from here (CLAUDE.md §3.3).

export {
  IVF_ROLES,
  IVF_ROUTES,
  IVF_STAGES,
  type IvfAppointmentKind,
  type IvfCompanion,
  type IvfCycle,
  type IvfDose,
  type IvfDoseDay,
  type IvfDoseInput,
  type IvfHome,
  type IvfNextAppointment,
  type IvfRole,
  type IvfRoute,
  type IvfStage,
  type IvfStageInfo,
  type IvfStepStatus,
  type IvfTimelineStep,
} from './model/types';
export { ivfKeys } from './api/keys';
export { ivfHomeSchema, ivfStagesSchema } from './api/schema';
export {
  fetchIvfHome,
  useIvfHome,
  useIvfStages,
  useLogIvfDose,
  useSetIvfCompanionNotify,
  useStartIvfCycle,
  useUnlogIvfDose,
} from './api/queries';

// CB-IVF-03 — the injection schedule (`/ivf/meds`): meds CRUD, doses with a site, catalog sites/presets/guidance.
export {
  IVF_INJECTED_ROUTES,
  IVF_MAX_TIMES,
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
  useUnlogIvfScheduleDose,
  useUpdateIvfMed,
} from './api/meds-queries';
