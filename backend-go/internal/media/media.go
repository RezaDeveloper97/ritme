// Package media is the lesson media pipeline of the courses domain (bloom B-N8-02, deviations.md D-81): resumable
// uploads of video / audio / pdf by approved instructors and protected, range-capable playback for their students.
//
// Why not internal/files (CB-CORE-05): that storage seals every blob with AES-256-GCM in one piece and reads it whole
// into memory — right for ≤ 10 MB health documents, wrong for a 2 GB lesson video that a player reads by byte
// ranges. Course media is an instructor's published teaching material, not a user's health data, so it is stored
// unencrypted on its own volume (MEDIA_STORAGE_PATH, default STORAGE_PATH/app/private/media; directories 0700,
// files 0600) and protected by access checks plus short-lived HMAC URLs instead. The URL scheme follows files.Signer.
//
// Upload (tus-style, on /api/instructor/v1/media, approved instructors only, owner-scoped):
//
//	POST   /media {lesson_id, size, mime?}   → 201 {media}: an upload of `size` bytes for one of the caller's lessons
//	GET    /media/{id}                      → {media} with offset_bytes — the resume point after an interruption
//	PATCH  /media/{id}  Upload-Offset: n    → the next chunk (application/offset+octet-stream, ≤ MaxChunkBytes);
//	                                          409 media_offset_mismatch {offset} when n is not the stored offset
//	DELETE /media/{id}                      → cancel / remove
//
// A chunk is applied whole or not at all (the body is read completely before it is written), so an interrupted
// request leaves the offset where it was and the client resumes from GET's offset_bytes. The first chunk is
// sniffed (magic bytes, never the client's name or Content-Type) against the lesson kind; the last one finalizes
// the upload: processing → virus-scan hook (Scanner; no-op today) → ready, and the lesson's media_id / media_status
// follow. Unfinished uploads expire after UploadTTL and are swept with their files.
//
// Playback: an authenticated call mints a URL (GET /api/v1/learning/lessons/{id}/playback for a student — the same
// access rules as the lesson itself; GET /api/instructor/v1/media/{id}/playback for its owner):
//
//	/api/v1/media/{id}/stream?u=<viewer>&expires=<unix>&signature=<base64url>
//	signature = HMAC-SHA256(MEDIA_URL_KEY, "ritme-media-url:v1\n<id>\n<viewer>\n<expires>")
//
// The stream answers 200 / 206 (one byte range) / 416; a wrong link is 403 link_invalid, a past expiry 410
// link_expired (the player then mints a fresh URL and resumes at its position). The token lives in the query string,
// which the Go access log never writes (it logs the path only); the nginx locations turn their access log off.
//
// HLS later (not built): a processing step after the scan can transcode a ready video into an HLS ladder
// (ffmpeg, `<id>/hls/*.m3u8|*.ts`) and the stream route serves the playlist and segments under the same signed
// prefix (the signature then covers the directory, not one file). The status machine already has `processing`.
package media

import (
	"errors"
	"slices"
	"time"
)

// Lesson / media kinds (= learning lesson kinds).
const (
	KindVideo = "video"
	KindAudio = "audio"
	KindPDF   = "pdf"
)

// Media statuses (learning_media.status) and the lesson's extra «none».
const (
	StatusUploading  = "uploading"
	StatusProcessing = "processing"
	StatusReady      = "ready"
	StatusFailed     = "failed"
	LessonNone       = "none"
)

const (
	mib = int64(1) << 20
	gib = int64(1) << 30
)

// Limits (product decisions, D-81).
const (
	// MaxChunkBytes bounds one PATCH body (inside the API's 25 MB body limit and nginx's client_max_body_size).
	MaxChunkBytes = 16 * mib
	// ChunkSize is the chunk size the client is told to use.
	ChunkSize = 8 * mib
	// UploadTTL is how long an unfinished upload may be resumed.
	UploadTTL = 24 * time.Hour
	// MaxActiveUploads bounds the unfinished uploads of one instructor.
	MaxActiveUploads = 5
	// QuotaBytes bounds what one instructor stores (uploads in flight included).
	QuotaBytes = 100 * gib
)

// MaxBytes is the largest upload per kind.
var MaxBytes = map[string]int64{KindVideo: 2 * gib, KindAudio: 500 * mib, KindPDF: 100 * mib}

// Mimes are the accepted (sniffed) content types per kind.
var Mimes = map[string][]string{
	KindVideo: {"video/mp4", "video/quicktime", "video/webm"},
	KindAudio: {"audio/mpeg", "audio/mp4", "audio/aac", "audio/ogg", "audio/wav", "audio/flac", "audio/webm"},
	KindPDF:   {"application/pdf"},
}

// AllMimes lists every accepted content type (the declared `mime` of POST /media).
func AllMimes() []string {
	var out []string
	for _, k := range []string{KindVideo, KindAudio, KindPDF} {
		out = append(out, Mimes[k]...)
	}
	return out
}

// Accepts reports whether kind accepts mime.
func Accepts(kind, mime string) bool { return slices.Contains(Mimes[kind], mime) }

// Errors mapped by the handlers.
var (
	ErrDisabled       = errors.New("media: storage unavailable")
	ErrNotFound       = errors.New("media: not found")
	ErrLessonNotFound = errors.New("media: lesson not found")
	ErrNotReady       = errors.New("media: no ready media")
	ErrTooLarge       = errors.New("media: upload too large")
	ErrChunkTooLarge  = errors.New("media: chunk too large")
	ErrBeyondSize     = errors.New("media: chunk past the declared size")
	ErrType           = errors.New("media: content type not accepted")
	ErrMimeMismatch   = errors.New("media: declared type not accepted for the lesson kind")
	ErrTooManyUploads = errors.New("media: too many unfinished uploads")
	ErrQuota          = errors.New("media: storage quota exceeded")
	ErrBusy           = errors.New("media: another chunk is being written")
	ErrNotUploading   = errors.New("media: upload already finished")
	ErrUploadExpired  = errors.New("media: upload expired")
	ErrRejected       = errors.New("media: rejected by the scanner")
	ErrLinkInvalid    = errors.New("media: invalid link")
	ErrLinkExpired    = errors.New("media: link expired")
)

// OffsetMismatchError is a chunk sent for another offset than the stored one (tus 409).
type OffsetMismatchError struct{ Offset uint64 }

func (e *OffsetMismatchError) Error() string { return "media: offset mismatch" }
