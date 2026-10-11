package validation_test

import (
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func TestParseQuery_PHPParseStrAndNormalisation(t *testing.T) {
	q := validation.ParseQuery("a=%20x%20&empty=&b[]=1&b[]=2&c[k]=v&log.date=d&a=last&blank=%E2%80%8B")
	assert.Equal(t, `{"a":"last","empty":null,"b":["1","2"],"c":{"k":"v"},"log_date":"d","blank":null}`, canonical(t, phpval.Packed(q)))
}

func TestNormalize_TrimStringsAndEmptyToNull(t *testing.T) {
	v, err := phpval.Decode([]byte("{\"a\":\"  x \",\"b\":\"\",\"c\":[\"  \",1,true],\"password\":\"  keep  \",\"d\":{\"e\":\"\u00a0y\ufeff\"}}"))
	require.NoError(t, err)
	assert.Equal(t, `{"a":"x","b":null,"c":[null,1,true],"password":"  keep  ","d":{"e":"y"}}`, canonical(t, validation.Normalize(v)))
}

func TestInput_AllMergesBodyAndQuery(t *testing.T) {
	app := fiber.New()
	app.All("/", func(c fiber.Ctx) error { return c.JSON(validation.Input(c)) })
	call := func(method, target, ct, body string) string {
		req := httptest.NewRequest(method, target, strings.NewReader(body))
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		resp, err := app.Test(req)
		require.NoError(t, err)
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return string(b)
	}
	assert.JSONEq(t, `{"a":"body","n":5,"q":"x"}`, call("POST", "/?a=query&q=x", "application/json", `{"a":" body ","n":5}`))
	assert.JSONEq(t, `{"q":"x"}`, call("POST", "/?q=x", "application/json", `{not json`), "invalid JSON counts as empty")
	assert.JSONEq(t, `{"0":1,"1":2}`, call("POST", "/", "application/json", `[1,2]`))
	assert.JSONEq(t, `{"f":"v","list":["a","b"]}`, call("POST", "/", "application/x-www-form-urlencoded", `f=+v+&list[]=a&list[]=b`))
	assert.JSONEq(t, `{"q":"1"}`, call("GET", "/?q=1", "", ""))
	assert.JSONEq(t, `[]`, call("POST", "/", "", `{"a":1}`), "no content type: nothing parsed")
}

// Multipart fields come out of a Go map: indexed names must still rebuild as an ordered list, every time.
func TestInput_MultipartIndexedFieldsAreAList(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error { return c.JSON(validation.Input(c)) })
	for range 20 {
		var buf strings.Builder
		w := multipart.NewWriter(&buf)
		for _, k := range []string{"m[10]", "m[2]", "m[1]", "m[0]", "m[3]", "m[4]", "m[5]", "m[6]", "m[7]", "m[8]", "m[9]"} {
			require.NoError(t, w.WriteField(k, strings.Trim(k, "m[]")))
		}
		require.NoError(t, w.Close())
		req := httptest.NewRequest(fiber.MethodPost, "/", strings.NewReader(buf.String()))
		req.Header.Set("Content-Type", w.FormDataContentType())
		resp, err := app.Test(req)
		require.NoError(t, err)
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"m":["0","1","2","3","4","5","6","7","8","9","10"]}`, string(b))
	}
}
