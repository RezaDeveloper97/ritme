package partners

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

var now = time.Date(2026, 10, 11, 9, 0, 0, 0, civildate.Tehran)

func TestLangFilesLoad(t *testing.T) {
	assert.Equal(t, "Your request was sent", T("messages.requested", "en"))
	assert.Equal(t, "درخواستت ثبت شد", T("messages.requested", "fa"))
	assert.NotEmpty(t, attributes("fa"))
	assert.NotEmpty(t, attributes("en"))
}

func TestCanMove(t *testing.T) {
	ok := [][2]string{{StatusRequested, StatusAccepted}, {StatusAccepted, StatusSampled}, {StatusSampled, StatusResultReady},
		{StatusRequested, StatusRejected}, {StatusAccepted, StatusRejected}, {StatusRequested, StatusSampled}}
	for _, m := range ok {
		assert.True(t, CanMove(m[0], m[1]), m[0]+"→"+m[1])
	}
	bad := [][2]string{{StatusSampled, StatusRejected}, {StatusResultReady, StatusSampled}, {StatusAccepted, StatusAccepted},
		{StatusCancelled, StatusAccepted}, {StatusRejected, StatusAccepted}, {StatusRequested, StatusCancelled}, {StatusResultReady, StatusRejected}}
	for _, m := range bad {
		assert.False(t, CanMove(m[0], m[1]), m[0]+"→"+m[1])
	}
	assert.True(t, CanCancel(StatusRequested))
	assert.True(t, CanCancel(StatusAccepted))
	assert.False(t, CanCancel(StatusSampled))
}

// The partner payload is an allow-list: adding a field is a privacy decision that must change this test.
func TestSubmissionAllowList(t *testing.T) {
	var names []string
	rt := reflect.TypeOf(Submission{})
	for i := 0; i < rt.NumField(); i++ {
		names = append(names, rt.Field(i).Name)
	}
	assert.Equal(t, []string{"Reference", "ServiceCode", "FullName", "Phone", "City", "Address", "PreferredDate", "Window", "Note"}, names)
}

func TestPickText(t *testing.T) {
	raw := `{"fa":"آزمایش","en":"Lab"}`
	assert.Equal(t, "Lab", PickText(raw, "en", "fa"))
	assert.Equal(t, "آزمایش", PickText(raw, "ar", "fa"))
	assert.Equal(t, "plain", PickText("plain", "en", "fa"))
	assert.Equal(t, "x", PickText(`{"de":"x"}`, "en", "fa"))
}

func body(t *testing.T, js string) phpval.Map {
	t.Helper()
	v, err := phpval.Decode([]byte(js))
	require.NoError(t, err)
	return v.(phpval.Map)
}

func TestParseInput(t *testing.T) {
	good := `{"service_id":1,"full_name":" Sara  K ","phone":"۰۹۱۲-۰۰۰ ۰۰۰۰","city":"Tehran","address":"Valiasr","preferred_date":"2026-10-14","preferred_window":"noon","consent_share":true,"consent_import":false,"user_id":5}`
	in, err := parseInput(body(t, good), "en", now)
	require.NoError(t, err)
	assert.Equal(t, "Sara K", in.Payload.FullName)
	assert.Equal(t, "09120000000", in.Payload.Phone)
	assert.Equal(t, WindowNoon, in.Window)
	assert.True(t, in.ConsentShare)
	assert.False(t, in.ConsentImport)

	for name, js := range map[string]string{
		"no consent":    `{"service_id":1,"full_name":"a","phone":"09120000000","city":"c","address":"a","preferred_date":"2026-10-14","preferred_window":"noon"}`,
		"false consent": `{"service_id":1,"full_name":"a","phone":"09120000000","city":"c","address":"a","preferred_date":"2026-10-14","preferred_window":"noon","consent_share":false}`,
		"bad phone":     `{"service_id":1,"full_name":"a","phone":"call","city":"c","address":"a","preferred_date":"2026-10-14","preferred_window":"noon","consent_share":true}`,
		"past day":      `{"service_id":1,"full_name":"a","phone":"09120000000","city":"c","address":"a","preferred_date":"2026-10-10","preferred_window":"noon","consent_share":true}`,
		"far day":       `{"service_id":1,"full_name":"a","phone":"09120000000","city":"c","address":"a","preferred_date":"2027-03-01","preferred_window":"noon","consent_share":true}`,
		"bad window":    `{"service_id":1,"full_name":"a","phone":"09120000000","city":"c","address":"a","preferred_date":"2026-10-14","preferred_window":"night","consent_share":true}`,
	} {
		_, err := parseInput(body(t, js), "en", now)
		assert.Error(t, err, name)
	}
}
