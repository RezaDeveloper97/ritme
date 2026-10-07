package media

import (
	"strconv"
	"strings"
)

// byteRange is the part of a file a GET answers: [Start, End] inclusive. Partial = 206.
type byteRange struct {
	Start, End int64
	Partial    bool
}

// parseRange reads a Range header for a file of size bytes (RFC 9110 §14). Only one "bytes=" range is honoured;
// a missing, malformed, multi-range or foreign-unit header is ignored (the whole file, 200), as the RFC allows.
// ok=false = unsatisfiable (416).
func parseRange(header string, size int64) (byteRange, bool) {
	full := byteRange{Start: 0, End: size - 1}
	spec, found := strings.CutPrefix(strings.TrimSpace(header), "bytes=")
	if !found || strings.Contains(spec, ",") {
		return full, true
	}
	first, last, found := strings.Cut(strings.TrimSpace(spec), "-")
	if !found {
		return full, true
	}
	if first == "" { // suffix: the last N bytes
		n, err := strconv.ParseInt(last, 10, 64)
		if err != nil || n < 0 {
			return full, true
		}
		if n == 0 || size == 0 {
			return byteRange{}, false
		}
		return byteRange{Start: max(size-n, 0), End: size - 1, Partial: true}, true
	}
	start, err := strconv.ParseInt(first, 10, 64)
	if err != nil || start < 0 {
		return full, true
	}
	end := size - 1
	if last != "" {
		e, err := strconv.ParseInt(last, 10, 64)
		if err != nil || e < start {
			return full, true
		}
		end = min(e, size-1)
	}
	if start >= size {
		return byteRange{}, false
	}
	return byteRange{Start: start, End: end, Partial: true}, true
}
