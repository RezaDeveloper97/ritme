package form

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// ImageRule is `[required|nullable]|image|mimes:jpeg,jpg,png,webp|max:<KB>[|dimensions:min_width=…,min_height=…]`
// for one multipart file field. The file's bytes decide its type (never the client's
// file name or Content-Type), so SVG, HTML or PHP renamed to .jpg are refused.
type ImageRule struct {
	Field     string
	Required  bool
	MaxKB     int
	MinWidth  int
	MinHeight int
}

// mimesList is the :values of the mimes message.
const mimesList = "jpeg, jpg, png, webp"

// Check validates the upload and, when it passes, stores the inspected image in *out
// (left nil when no file was sent and none is required).
func (r ImageRule) Check(c fiber.Ctx, out **media.Image) Check {
	return func(in phpval.Map, add Add) error {
		fh, err := c.FormFile(r.Field)
		if err != nil || fh == nil {
			switch {
			case Has(in, r.Field): // a text value where a file belongs
				add(r.Field, Msg(c, "validation.image", r.Field))
			case r.Required:
				add(r.Field, Msg(c, "validation.required", r.Field))
			}
			return nil
		}
		data, err := media.ReadUpload(fh, int64(r.MaxKB)*1024)
		if errors.Is(err, media.ErrTooLarge) {
			add(r.Field, Msg(c, "validation.max.file", r.Field, "max", strconv.Itoa(r.MaxKB)))
			return nil
		}
		if err != nil {
			add(r.Field, Msg(c, "validation.uploaded", r.Field))
			return nil //nolint:nilerr // an unreadable upload is a validation failure, not a server error
		}
		img, err := media.Inspect(data)
		switch {
		case errors.Is(err, media.ErrWrongType):
			add(r.Field, Msg(c, "validation.mimes", r.Field, "values", mimesList))
			return nil
		case err != nil:
			add(r.Field, Msg(c, "validation.image", r.Field))
			return nil //nolint:nilerr // not an image: a validation failure
		}
		if img.Width < r.MinWidth || img.Height < r.MinHeight {
			add(r.Field, Msg(c, "validation.dimensions", r.Field))
			return nil
		}
		*out = &img
		return nil
	}
}
