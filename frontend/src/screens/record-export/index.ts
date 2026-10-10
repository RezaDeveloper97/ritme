// Public API of the `record-export` screen (bloom B-N6-04): doctor report builder, preview and the public share view.
export { RecordExportPage } from './ui/RecordExportPage';
export { RecordPreviewPage } from './ui/RecordPreviewPage';
export { SharedReportPage } from './ui/SharedReportPage';
// CB-MENO-11: the menopause doctor report on the builder's PDF + share link (`/menopause/report`).
export { MenopauseReportPage, type MenopauseReportOrigin } from './ui/MenopauseReportPage';
// CB-REC-05: the public doctor code page `/d` (24h summary by code) on the same read-only paper.
export { DoctorCodePage } from './ui/DoctorCodePage';
