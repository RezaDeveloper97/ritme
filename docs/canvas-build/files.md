# Encrypted file storage (CB-CORE-05)

Generic storage for every document kind the canvas needs, generalised from bloom's lab-sheet storage (B-N6-06).
Code: `backend-go/internal/files` (Go only, deviations.md **D-59**). Consumers: CB-REC (record documents),
insurance (claim documents), CB-DIR (place photos / licences), CB-SHOP (product images).

## Layers

| Layer | File | What |
|---|---|---|
| `Vault` | `vault.go` | AES-256-GCM blobs on disk, per purpose and owner. No DB. Labs use it through `internal/labs/files`. |
| `Signer` | `signer.go` | HMAC download links, short-lived, bound to one file of one owner. |
| `Service` | `service.go` | The `files` table, type / size / quota rules, photo re-encoding, owner / signed / public reads. |
| `Handlers` | `handlers.go` | `/api/v1/files/*` (routes in `internal/http/routes_files.go`). |

## Purposes

| Purpose | Visibility | Who writes | Accepts | Max per file | Quota per owner |
|---|---|---|---|---|---|
| `record_document` | private | user (`POST /files`) | photo, PDF | 10 MB | 300 files / 500 MB |
| `claim_document` | private | user | photo, PDF | 10 MB | 200 files / 300 MB |
| `place_licence` | private | Go code only for now (CB-DIR opens user uploads with place-ownership checks) | photo, PDF | 10 MB | 20 files / 100 MB |
| `place_photo` | **public** | Go code only for now (same) | photo | 5 MB | 30 files / 60 MB |
| `product_image` | **public** | Go code only (admin, owner 0 = Ritme team) | photo | 5 MB | 20 000 files / 10 GB (shared by the team) |
| `lab` | private | `internal/labs` only | photo, PDF | (labs' own limits) | (labs' own limits) |

- Users can upload only `record_document` and `claim_document` (`UserUpload`); every other purpose is 422 on
  `POST /files` and written through the Go API by the task that owns it.
- The type is sniffed from the bytes (`ai.SniffDocument`); the client's file name and Content-Type are ignored. A PDF
  must also start with `%PDF-` (after an optional BOM / whitespace) and carry `%%EOF` in its last KB — stricter than
  the shared sniffer, so polyglots with junk before the header are refused. Photos: at most 20 MP; PNGs with a bit
  depth above 8 are refused before decoding.
- Photos (JPEG / PNG / WebP) are decoded and **re-encoded to WebP** (≤ 2400 px, q85; EXIF / GPS and anything
  appended after the image dropped; at most 2 re-encodes at once → 503 `upload_busy` + `Retry-After`). PDFs are
  stored as sent. A public purpose accepts photos only, so a public file is always a re-encoded image.
- `size_bytes` and `sha256` describe the stored plaintext (after re-encoding).
- Quotas are checked inside the upload transaction: the user row is locked first (`SELECT … FOR UPDATE`, which
  serialises one user's uploads), then the owner's `(user_id, purpose)` usage is read with a locking read, so
  concurrent uploads cannot overshoot. Platform uploads (owner 0, no row to lock) retry a deadlock up to 3 times.
  Deleting a file frees its quota.
- Limits live in `files.Registry` (code, not data); a consumer that needs other limits changes the registry.
- `lab` is *blob only*: lab sheets keep their metadata in `lab_files` and never appear in `files` or `/files/*`.

## Storage at rest

```
STORAGE_PATH/app/private/files/<purpose>/<owner>/<40 hex>.bin      (lab: app/private/labs/<user>/<40 hex>.bin)
"RFF1" (lab: "RLF1") ‖ key id (sha256(key)[:4]) ‖ nonce (12) ‖ AES-256-GCM(ciphertext ‖ tag)
additional data: "file:v1:<purpose>:<owner>:<relative path>"   (lab: "lab-file:v1:<user>:<path>")
```

Private part of the volume only (never `app/public`, which `/storage/*` serves); directories 0700, files 0600,
created with `O_EXCL`. A blob copied to another owner, purpose or name does not decrypt; a tampered blob fails the
tag. Paths are validated before touching the disk (no traversal, no other owner's directory). The lab format is
unchanged from B-N6-06, so existing lab sheets still open.

## Keys (server env only — never in the repo, `.env` files in git, logs or chat)

| Env | Meaning |
|---|---|
| `FILE_KEY` | base64 of 32 random bytes (`openssl rand -base64 32`): seals new files, roots the URL-signing key. |
| `FILE_KEY_PREVIOUS` | comma-separated older keys: decrypt only, after a rotation. |
| `LAB_FILE_KEY` (+ `_PREVIOUS`) | lab sheets' key. **Fallback** for generic files when `FILE_KEY` is unset (startup log Warn "FILE_KEY is not set: generic files sealed with LAB_FILE_KEY"); when `FILE_KEY` is set later, `LAB_FILE_KEY` and its previous keys are still tried for decryption, so files sealed under it keep opening. |

`FILE_KEY` / `FILE_KEY_PREVIOUS` are passed to the Go service by `docker-compose.yml`, `docker-compose.stage.yml`,
`docker-compose.prod.yml` (`${FILE_KEY:-}`) and documented empty in `.env.stage.example`; the contract stack sets a
contract-only key. Set a dedicated `FILE_KEY` on stage / prod so lab sheets and other documents use separate keys.

Policy (`config.Files.Resolve`, same as the private-note / lab key policy): without any key a public development key
is used **only** for `APP_ENV` local / testing / contract; anywhere else (staging, production) the vault is disabled
and every `/files/*` route answers 503 `storage_unavailable` (fail closed). Removal never needs a key.

Rotation: set the new `FILE_KEY`, move the old one into `FILE_KEY_PREVIOUS`, restart. Old files open with the
previous key; new ones are sealed with the new key. Outstanding signed links stop verifying (they last minutes).

## Links

- **Private files** — `GET /api/v1/files/{id}/download?expires=<unix>&signature=<base64url>`, valid 5 minutes
  (`DefaultLinkTTL`, at most `MaxLinkTTL` = 15 min). `signature = HMAC-SHA256(K, "ritme-file-url:v1\n<id>\n<owner>\n
  <purpose>\n<path>\n<expires>")` with `K = HMAC-SHA256(file key, "ritme-file-url-key:v1")` (never the AES key
  itself). The verifier recomputes it from the stored row, so a link opens exactly one blob of one owner: another
  id, another user's file, a changed expiry or a forged signature is 403 `link_invalid`; past expiry 403
  `link_expired` (never 401 — clients wipe the session only on an auth 401). No bearer is needed (the link works in
  `<img src>` / a download); the response is an attachment with `Cache-Control: private, no-store`, `nosniff`, a
  sandbox CSP and `Referrer-Policy: no-referrer`.
- **Public files** — `GET /api/v1/files/public/{id}/{name}`, no auth, cacheable for an hour (`max-age=3600`). Served only when the
  purpose is public in the registry **and** the row is public **and** it is an image **and** `name` is the blob's
  random name (ids cannot be enumerated). A private file is 404 under any id and name.
- Both anonymous routes are IP-throttled (300 / minute). Links are absolute on `APP_URL`.
- `POST /files` and `GET /files/{id}` return `url` (+ `url_expires_at` for private files); clients ask
  `GET /files/{id}` again for a fresh link instead of storing one.

## Go API for consumers

```go
svc := files.NewService(files.Options{DB: db, Vault: vault, BaseURL: cfg.App.URL})
f, err := svc.Put(ctx, userID, files.PurposeRecordDocument, data, now) // ErrPurpose/ErrType/ErrTooLarge/ErrQuota/ErrBusy/ErrDisabled
f, err  = svc.Get(ctx, userID, id)          // owner-scoped (0 = Ritme team), ErrNotFound otherwise
f, b, _ := svc.Open(ctx, userID, id)        // + decrypted bytes (e.g. to hand to the AI extractor)
link, _ := svc.Link(f, now, 0)              // signed (private) or public URL
err     = svc.Delete(ctx, userID, id)
```

A consumer table stores `files.id` (e.g. `record_documents.file_ids`), checks ownership through `Get` with the
request user, and deletes files it owns with `Delete`. Build the vault like `routes_files.go` does (or share one).

## Privacy rules (each has a test in `internal/files`)

- Health files are user-scoped in every query (`user_id` in the SQL), encrypted at rest, never logged (no content,
  name, hash or owner in logs); the storage path never leaves the server.
- User B can neither read, link nor delete user A's file (uniform 404); A's signed link moved onto B's file is 403.
- Public purposes never serve private files; the platform (owner 0) can only own public files.
- Account deletion (`internal/profile`, admin user delete) drops the rows (FK cascade) and every purpose's blobs
  (`files.RemoveUser`, lab sheets included).

## Orphan sweep

`Service.SweepLoop` (started with the listener, every 6 h) removes generic blobs that have no `files` row and are
older than an hour (an upload writes its blob before its row commits): a failed blob removal after a delete, or an
account deleted by a path that did not call `RemoveUser`. Lab sheets keep their own sweep (internal/labs).

## Errors and logs

Vault errors never carry a filesystem path (`*os.PathError` is rewritten to op + cause), since storage paths carry
the owner id.

## Notes / follow-ups (security review of CB-CORE-05)

- **L4** — nginx access logs record the full signed URL (`?expires&signature`). Links are short-lived and single-file,
  but stage/prod nginx should drop the query string for `/api/v1/files/*/download` from access logs.
- **L5** — behind a CDN, the IP throttle of the anonymous routes needs the CDN's `real_ip` configuration, or every
  request shares one IP bucket.
- **L8** — quotas are per purpose; there is no global per-user cap across purposes yet (worst case = sum of the
  user-uploadable purposes, ~800 MB).
- Admin upload / delete endpoints for `product_image` / `place_photo` belong to CB-SHOP / CB-DIR admin tasks;
  CB-DIR re-enables user uploads of `place_photo` / `place_licence` together with place-ownership checks.
