package emergency

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

var now = time.Date(2026, 10, 6, 9, 0, 0, 0, civildate.Tehran)

func TestLangFilesLoad(t *testing.T) {
	assert.Equal(t, "Emergency card saved", T("messages.saved", "en"))
	assert.Equal(t, "کارت اضطراری ذخیره شد", T("messages.saved", "fa"))
	assert.NotEmpty(t, attributes("fa"))
}

func TestMaskAndDigits(t *testing.T) {
	assert.Equal(t, "•••• 4821", MaskLast4("4821"))
	assert.Equal(t, "09120000000", asciiDigits(" ۰۹۱۲-۰۰۰ ۰۰۰۰ "))
	assert.Equal(t, "+989120000000", asciiDigits("+٩٨٩١٢٠٠٠٠٠٠٠"))
}

func body(t *testing.T, js string) phpval.Map {
	t.Helper()
	v, err := phpval.Decode([]byte(js))
	require.NoError(t, err)
	return v.(phpval.Map)
}

func TestParseInput(t *testing.T) {
	in, err := parseInput(body(t, `{"show_on_lock_screen":true,"emergency_contact":{"name":"  Ali  ","relation":null,"phone":"۰۹۱۲۰۰۰۰۰۰۰"},"insurance":null,"x":1}`), "en", now)
	require.NoError(t, err)
	assert.True(t, in.SetLockScreen && in.LockScreen)
	assert.False(t, in.SetPregnancy, "absent key keeps the value")
	require.NotNil(t, in.Contact)
	assert.Equal(t, Contact{Name: "Ali", Phone: "09120000000"}, *in.Contact)
	assert.True(t, in.SetInsurance)
	assert.Nil(t, in.Insurance, "null clears")

	in, err = parseInput(body(t, `{"emergency_contact":null,"insurance":{"last4":"۴۸۲۱"}}`), "en", now)
	require.NoError(t, err)
	assert.True(t, in.SetContact)
	assert.Nil(t, in.Contact)
	assert.Equal(t, &Insurance{Last4: "4821"}, in.Insurance)

	for name, js := range map[string]string{
		"bool":         `{"show_pregnancy":"maybe"}`,
		"no name":      `{"emergency_contact":{"phone":"09120000000"}}`,
		"bad phone":    `{"emergency_contact":{"name":"A","phone":"12"}}`,
		"letters":      `{"emergency_contact":{"name":"A","phone":"call me"}}`,
		"full number":  `{"insurance":{"last4":"6037991234564821"}}`,
		"not digits":   `{"insurance":{"last4":"12a4"}}`,
		"long label":   `{"insurance":{"label":"` + strings.Repeat("x", 61) + `","last4":"1234"}}`,
		"contact list": `{"emergency_contact":["Ali"]}`,
	} {
		_, err := parseInput(body(t, js), "en", now)
		var fe *httpx.FailError
		require.ErrorAs(t, err, &fe, name)
		assert.Equal(t, 422, fe.Status, name)
	}
}

func TestOwnerJSON_InsuranceOnlyForOwner(t *testing.T) {
	c := Card{Allergies: true, Public: jsonx.Obj("name", "Sara")}
	c.Row.InsuranceLast4.String, c.Row.InsuranceLast4.Valid = "4821", true
	out := c.OwnerJSON()
	card, _ := out.Get("card")
	ins, _ := card.(*jsonx.OrderedMap).Get("insurance")
	masked, _ := ins.(*jsonx.OrderedMap).Get("masked")
	assert.Equal(t, "•••• 4821", masked)
	_, inPublic := c.Public.Get("insurance")
	assert.False(t, inPublic, "the public card never carries insurance")
}
