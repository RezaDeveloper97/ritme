-- phase_contents (App\Models\PhaseContent).

-- name: GetPhaseContent :one
-- PhaseContent::getByPhase($phase): where('phase', $phase)->first().
SELECT id, phase, symptom_prediction, vaginal_discharge, fertility, hormonal_changes,
       sex_tips, nutrition, exercise, skin_care, sleep
FROM `phase_contents`
WHERE phase = ?
LIMIT 1;
