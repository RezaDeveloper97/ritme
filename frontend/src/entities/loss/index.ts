// Public API of the `loss` entity (CB-LOSS-02 on the CB-LOSS-01 API). Import only from here (CLAUDE.md §3.3).

export {
  LOSS_CATALOG_GROUPS,
  LOSS_MOODS,
  LOSS_NEXT_STEPS,
  LOSS_TYPES,
  type LossAppointment,
  type LossCatalogGroup,
  type LossCatalogItem,
  type LossEvent,
  type LossFollowup,
  type LossFollowupInput,
  type LossHotline,
  type LossMood,
  type LossNextStep,
  type LossNote,
  type LossState,
  type LossType,
  type RecordLossInput,
} from './model/types';
export {
  crisisHotlines,
  emergencyNumber,
  lifeModeOfNextStep,
  shouldRecordLoss,
  splitWallClock,
  visitAt,
  warningSignParts,
} from './model/loss';
export { lossKeys } from './api/keys';
export { lossCatalogSchema, lossNoteSchema, lossStateSchema } from './api/schema';
export {
  fetchLossState,
  lossFieldError,
  lossNoteProblem,
  type LossNoteProblem,
  useLogLossMood,
  useLossCatalog,
  useLossNote,
  useLossState,
  useRecordLoss,
  useSaveLossFollowup,
  useSaveLossNextStep,
  useSaveLossNote,
} from './api/queries';
