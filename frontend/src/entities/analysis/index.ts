// Public API of the `analysis` entity (B-N3-08). Import only from here (CLAUDE.md §3.3).
// B-N3-09…12 add their report parsers/hooks to api/ and export them here.

export {
  ANALYSIS_RANGES,
  DEFAULT_RANGE,
  HUB_RANGES,
  isAnalysisRange,
  type AnalysisPhrase,
  type AnalysisRange,
  type AnalysisRangeKey,
  type AnalysisSection,
  type AnalysisSummary,
  type Correlation,
  type CorrelationGroup,
  type CyclePhase,
  type CycleStatus,
  type HubCycle,
  type HubCycleBar,
  type HubRangeKey,
  type HubRecentCycle,
  type HubSymptoms,
  type HubVitals,
  type HubWeight,
  type BloodPressureStats,
  type BloodSugarStats,
  type MoodByPhase,
  type MoodShare,
  type PatternRelation,
  type PhaseMoods,
  type Regularity,
  type SymptomCount,
  type SymptomHighlight,
  type TopFinding,
  type TopFindingKind,
  type VitalsReadiness,
  type WeightPoint,
} from './model/types';
export { barShares, lowestIndex, shareOfMax, stripCells, type StripCell, type StripDay } from './model/layout';

export { analysisKeys } from './api/keys';
export { analysisRangeSchema, analysisSummarySchema, correlationSchema, phraseSchema, sectionSchema } from './api/schema';
export { fetchAnalysisSummary, useAnalysisSummary } from './api/queries';

// B-N3-09 — detail reports (cycle, period, symptoms, correlations, body).
export {
  CORRELATION_KEYS,
  REPORT_DEFAULT_RANGE,
  REPORT_RANGES,
  type BodyReport,
  type CorrelationKey,
  type CorrelationsReport,
  type CycleExclusion,
  type CycleLayout,
  type CycleReport,
  type FlowLevel,
  type PeriodReport,
  type PeriodStatus,
  type ReportCycle,
  type ReportName,
  type ReportPeriod,
  type SymptomPatternItem,
  type SymptomsReport,
} from './model/reports';
export {
  bodyReportSchema,
  correlationsReportSchema,
  cycleReportSchema,
  periodReportSchema,
  symptomsReportSchema,
} from './api/reports';
export {
  useBodyReport,
  useCorrelationsReport,
  useCycleReport,
  usePeriodReport,
  useSymptomsReport,
} from './api/report-queries';

// B-N3-10 — monthly report (An_Monthly).
export {
  MONTHLY_CALENDARS,
  MONTHLY_METRIC_KEYS,
  isMonthlyCalendar,
  type BloodPressureValue,
  type MonthlyCalendar,
  type MonthlyMetric,
  type MonthlyMetricKey,
  type MonthlyMonth,
  type MonthlyReport,
} from './model/monthly';
export { fetchMonthlyReport, monthlyReportSchema, useMonthlyReport } from './api/monthly';

// T-M10-26 — per-marker lab trend (An_Labs, §4.1).
export { trendByMarker, type LabTrendDirection, type LabTrendList, type LabTrendMarker } from './model/lab-trend';
export { fetchLabTrendList, labTrendListSchema, useLabTrendList } from './api/lab-trend';
