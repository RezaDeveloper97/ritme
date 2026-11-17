// The autopilot's vocabulary (docs/autopilot.md, frontend/CLAUDE.md §11.6): how controls are named, the
// script contract, the catalog shape and the agent manifest. Pure and UI-free, so any component can import
// it to mark a control without pulling in the engine (`@/shared/autopilot`).
export {
  EMPTY_CATALOG,
  findCatalogTarget,
  matchesAny,
  type CatalogMatch,
  type CatalogRoute,
  type CatalogSheet,
  type CatalogTarget,
  type PilotCatalog,
  type PilotSurface,
  type PilotValueType,
} from './catalog';
export {
  buildAutopilotManifest,
  MANIFEST_GUIDE,
  type AutopilotManifest,
  type BuildManifestInput,
  type ManifestOption,
  type ManifestTarget,
} from './manifest';
export {
  parseAutopilotScript,
  PILOT_ACTIONS,
  SCRIPT_VERSION,
  targetOf,
  type ParseResult,
  type PilotAction,
  type PilotScript,
  type PilotStep,
  type PilotValue,
} from './script';
export {
  describeTarget,
  isPattern,
  isTargetId,
  matchTargetId,
  parseTargetId,
  pilot,
  PILOT_ATTR,
  PILOT_KIND_ATTR,
  PILOT_OPTION_ATTR,
  pilotOption,
  type ParsedTargetId,
  type PilotAttrs,
  type PilotKind,
  type PilotOptionAttrs,
  type TargetRef,
} from './target';
