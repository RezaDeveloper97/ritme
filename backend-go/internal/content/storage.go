package content

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/httpx"
)

// fileRoot serves the public disk (STORAGE_PATH/app/public), what Apache serves
// through the `public/storage` symlink in the Laravel image.
type fileRoot struct{ dir string }

// cleanStoragePath validates the path below /storage/: every segment must be a plain
// name — no "", ".", "..", dot-files, backslashes or NUL — so nothing can leave the
// root (os.Root enforces the same for symlinks).
func cleanStoragePath(p string) (string, bool) {
	if p == "" || strings.ContainsAny(p, "\\\x00") {
		return "", false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || strings.HasPrefix(seg, ".") {
			return "", false
		}
	}
	return p, true
}

// Storage is GET /storage/* : a regular file of the public disk, with Apache's default
// validators (Last-Modified, ETag "<size>-<mtime µs>" in hex, Accept-Ranges) and
// conditional 304s. Missing files, directories and anything outside the root are 404;
// there is no directory listing.
func (h *Handlers) Storage(c fiber.Ctx) error {
	if h.fileRoot.dir == "" {
		return httpx.NotFound()
	}
	// Fiber leaves the path percent-encoded. Encoded separators are refused (Apache's
	// AllowEncodedSlashes Off answers 404 too); the rest is decoded and every segment
	// must be a plain name, so "..", "%2e%2e", dot-files and NUL never reach the disk.
	raw := strings.TrimPrefix(string(c.Request().URI().PathOriginal()), "/storage/")
	if lower := strings.ToLower(raw); strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") {
		return httpx.NotFound()
	}
	rel, err := url.PathUnescape(raw)
	if err != nil {
		return httpx.NotFound()
	}
	name, ok := cleanStoragePath(rel)
	if !ok {
		return httpx.NotFound()
	}
	root, err := os.OpenRoot(h.fileRoot.dir)
	if err != nil {
		return httpx.NotFound()
	}
	f, err := root.Open(name)
	_ = root.Close()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) || isEscape(err) {
			return httpx.NotFound()
		}
		return fmt.Errorf("content: storage open: %w", err)
	}
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		_ = f.Close()
		return httpx.NotFound()
	}

	mod := st.ModTime().UTC()
	etag := `"` + strconv.FormatInt(st.Size(), 16) + "-" + strconv.FormatInt(mod.UnixMicro(), 16) + `"`
	c.Set(fiber.HeaderLastModified, mod.Format(time.RFC1123[:len(time.RFC1123)-3]+"GMT"))
	c.Set(fiber.HeaderETag, etag)
	c.Set(fiber.HeaderAcceptRanges, "bytes")
	if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
		c.Set(fiber.HeaderContentType, ct)
	}
	if notModified(c, etag, mod) {
		_ = f.Close()
		return c.SendStatus(fiber.StatusNotModified)
	}
	size := st.Size()
	start, length := int64(0), size
	c.Status(fiber.StatusOK)
	if rng := c.Get(fiber.HeaderRange); rng != "" && c.Get(fiber.HeaderIfRange) == "" {
		s, n, ok := parseRange(rng, size)
		switch {
		case !ok:
			_ = f.Close()
			c.Set(fiber.HeaderContentRange, "bytes */"+strconv.FormatInt(size, 10))
			return c.SendStatus(fiber.StatusRequestedRangeNotSatisfiable)
		case n >= 0:
			start, length = s, n
			c.Status(fiber.StatusPartialContent)
			c.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, start+length-1, size))
		}
	}
	if c.Method() == fiber.MethodHead {
		_ = f.Close()
		c.Response().SkipBody = true
		c.Response().Header.SetContentLength(int(length))
		return nil
	}
	if start > 0 {
		if _, err := f.Seek(start, io.SeekStart); err != nil {
			_ = f.Close()
			return fmt.Errorf("content: storage seek: %w", err)
		}
	}
	return c.SendStream(io.LimitReader(f, length), int(length))
}

// parseRange handles a single "bytes=a-b" / "bytes=a-" / "bytes=-n" range. n=-1 means
// "ignore the header and send the whole file" (multiple ranges, other units, syntax
// errors); ok=false means unsatisfiable (416).
func parseRange(h string, size int64) (start, n int64, ok bool) {
	spec, found := strings.CutPrefix(h, "bytes=")
	if !found || strings.Contains(spec, ",") {
		return 0, -1, true
	}
	a, b, found := strings.Cut(strings.TrimSpace(spec), "-")
	if !found {
		return 0, -1, true
	}
	if a == "" { // suffix: the last b bytes
		last, err := strconv.ParseInt(b, 10, 64)
		if err != nil || last < 0 {
			return 0, -1, true
		}
		if last == 0 || size == 0 {
			return 0, 0, false
		}
		if last > size {
			last = size
		}
		return size - last, last, true
	}
	first, err := strconv.ParseInt(a, 10, 64)
	if err != nil || first < 0 {
		return 0, -1, true
	}
	if first >= size {
		return 0, 0, false
	}
	end := size - 1
	if b != "" {
		e, err := strconv.ParseInt(b, 10, 64)
		if err != nil || e < first {
			return 0, -1, true
		}
		if e < end {
			end = e
		}
	}
	return first, end - first + 1, true
}

func isEscape(err error) bool {
	var pe *fs.PathError
	return errors.As(err, &pe) && strings.Contains(pe.Err.Error(), "escapes")
}

// notModified evaluates If-None-Match (preferred) / If-Modified-Since like Apache.
func notModified(c fiber.Ctx, etag string, mod time.Time) bool {
	if inm := c.Get(fiber.HeaderIfNoneMatch); inm != "" {
		for _, t := range strings.Split(inm, ",") {
			t = strings.TrimSpace(t)
			if t == "*" || strings.TrimPrefix(t, "W/") == etag {
				return true
			}
		}
		return false
	}
	if ims := c.Get(fiber.HeaderIfModifiedSince); ims != "" {
		if t, err := time.Parse(time.RFC1123, ims); err == nil {
			return !mod.Truncate(time.Second).After(t)
		}
	}
	return false
}
