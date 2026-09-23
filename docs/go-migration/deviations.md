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
| D-08 | Errors | 404/405 without `Accept: application/json` → HTML page | always the JSON body (both clients send the header) | proposed |
| D-09 | Dates | `Carbon::parse` accepts relative strings (`+1 day`) and military zone letters | `civildate.ParseLenient` rejects them (clients only send `Y-m-d`) | proposed |
| D-10 | Health | `/up` returns Laravel's HTML health page | plain `OK` (not part of the `/api` contract) | proposed |
| D-11 | Messages | `premium` persona `/messages/daily` → 500 on some dates (`OverrideType::LOW_ENERGY` doesn't exist, `CycleMessageEngine.php:102`) | preserved (golden locks the 500) unless the user asks for a fix | proposed: preserve |
| D-12 | Admin | changing own role / deactivating / deleting self silently ignored; Persian flash messages | 422 `cannot_modify_self`; English messages + `error_code` (admin-web translates) | proposed |
| D-13 | Storage | `/storage/*` via Apache: missing file → HTML 403, directory → 403, dot-files served, multi-range/If-Range honoured | missing/dir/dot-file → JSON 404; multi-range and If-Range → full file | proposed |
| D-14 | Admin articles | body stored raw, sanitized only on read | sanitized on write as well (read path unchanged) | proposed |
| D-15 | Admin languages | creating a language directly as default yields zero translation files | copies from the previous default language | proposed |
| D-16 | Admin banners | internal `link_url` accepts any string (incl. `javascript:`) | **kept as Laravel**; proposal: require a leading `/` | proposed |
