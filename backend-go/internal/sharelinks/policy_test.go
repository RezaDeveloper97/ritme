package sharelinks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func TestSharingPolicy(t *testing.T) {
	assert.True(t, SharingPolicy{}.DoctorSharing(), "an adult")
	assert.False(t, SharingPolicy{Teen: true}.DoctorSharing(), "a teen without a parent link")
	assert.True(t, SharingPolicy{Teen: true, ParentLinked: true}.DoctorSharing())
	b, err := jsonx.Marshal(SharingPolicy{Teen: true}.JSON(), 0)
	require.NoError(t, err)
	assert.JSONEq(t, `{"teen":true,"parent_linked":false,"doctor_sharing":false,"emergency_card":true}`, string(b))
}

func TestStripLosses(t *testing.T) {
	val, err := phpval.Decode([]byte(`{"record":{"sections":[
		{"key":"basics","empty":false,"data":{}},
		{"key":"pregnancies","editable":false,"empty":false,"data":{"pregnancies_count":3,"births_count":1,"items":[
			{"id":null,"source":"tracked","outcome":"birth","date":"2024-06-10","baby_count":1,"editable":false},
			{"id":null,"source":"tracked","outcome":"ended","date":null,"baby_count":null,"editable":false},
			{"id":null,"source":"manual","outcome":"ended","date":"2019-01-01","baby_count":null,"editable":false}]}}]}}`))
	require.NoError(t, err)
	m := val.(phpval.Map)
	stripLosses(m)
	b, err := jsonx.Marshal(m, 0)
	require.NoError(t, err)
	assert.NotContains(t, string(b), `"source":"tracked","outcome":"ended"`)
	assert.Contains(t, string(b), `"source":"manual","outcome":"ended"`, "her own typed entries stay")
	assert.Contains(t, string(b), `"pregnancies_count":2`)

	only, err := phpval.Decode([]byte(`{"record":{"sections":[{"key":"pregnancies","empty":false,"data":{"pregnancies_count":1,"births_count":0,"items":[{"source":"tracked","outcome":"ended"}]}}]}}`))
	require.NoError(t, err)
	stripLosses(only.(phpval.Map))
	b, err = jsonx.Marshal(only, 0)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"empty":true`)
	assert.Contains(t, string(b), `"items":[]`)

	stripLosses(phpval.NewMap()) // no record: a no-op
}
