# Laravel → Go deviations

Every intentional difference between the Laravel and Go output goes here **before** it is implemented, with the
user's OK. The contract harness allow-list must reference the entry id.

| Id | Area | Laravel behaviour | Go behaviour | Status |
|---|---|---|---|---|
| D-01 | Reminders | `GET /reminders/enums` with a 3rd locale → 500 (`$recurrenceLabels[$r][$locale]`) | fall back to English labels | proposed |
| D-02 | Typed int route params | non-numeric `{id}` → 500 `Server Error` | 404 | proposed |
| D-03 | Admin | `UserController::destroy` doesn't revoke tokens | admin delete revokes all tokens | proposed |
| D-04 | Languages | admin-added `lang/<code>/*.php` written into the image, lost on deploy | stored as JSON on the storage volume | proposed |
| D-05 | Messages | `MessageManager::extractSymptoms` / `PatternLayer` read non-existent columns (only `low_energy`, `insufficient_data`, `ttc_tracking`, `chronic_fatigue` can fire) | **preserved** in M2 (fix later as a product change) | decided: preserve |
| D-06 | Admin sessions | PHP-serialized redis sessions | new Go sessions → admins log in once again | decided |
| D-07 | Android | `POST /api/diagnostics/crash-reports` → 404 | still 404 (not in M2 scope) | decided |
