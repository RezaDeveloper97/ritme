// Public API of the analysis charts (B-N3-09): pure SVG on Night & Bloom tokens, no data fetching and no
// i18n — callers pass localized labels. Every chart is a role="img" with an accessible name plus a visually
// hidden data table. Plots run left → right (day 1 / oldest first) in both directions.
export { ChartFigure, type DataTable } from './ui/ChartFigure';
export { ColumnChart, type Column } from './ui/ColumnChart';
export { PhaseBar, type PhasePart } from './ui/PhaseBar';
export { CycleDots, type CycleDotRow } from './ui/CycleDots';
export { HeatGrid, type HeatLevel, type HeatRow } from './ui/HeatGrid';
export { SeriesBars, type BarSeries } from './ui/SeriesBars';
export { StripRows, type StripRow } from './ui/StripRows';
export { TrendLine } from './ui/TrendLine';
export { ChartCard, HBarList, StatTiles, type HBar, type StatTile } from './ui/Blocks';
export { ReportFrame, type ReportFrameProps } from './ui/ReportFrame';
