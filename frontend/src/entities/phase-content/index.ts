// Public API of the `phase-content` entity. Import only from here (§3.3).
export { PHASE_SECTION_KEYS } from './model/types';
export type { PhaseContent, PhaseSectionKey } from './model/types';
export {
  phaseContentKeys,
  fetchPhaseContent,
  usePhaseContent,
} from './api/queries';
