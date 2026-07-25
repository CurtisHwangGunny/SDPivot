package handler

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCommaSeparatedTagIDs(t *testing.T) {
	assert.Nil(t, parseCommaSeparatedTagIDs(""))
	assert.Equal(t, []string{"a", "b"}, parseCommaSeparatedTagIDs("a,b"))
	assert.Equal(t, []string{"a", "b"}, parseCommaSeparatedTagIDs(" a , b "))
	assert.Equal(t, []string{"a"}, parseCommaSeparatedTagIDs("a,__untagged__,,"))
}

func TestParseDimensionTagFilters(t *testing.T) {
	filters, err := parseDimensionTagFilters(`[
		{"dimension_id":"lifecycle","tag_ids":["active"]},
		{"dimension_id":"department","tag_ids":["legal"," legal ",""]},
		{"dimension_id":"department","tag_ids":["finance"]}
	]`)
	require.NoError(t, err)
	assert.Equal(t, []types.DimensionTagFilter{
		{DimensionID: "department", TagIDs: []string{"legal", "finance"}},
		{DimensionID: "lifecycle", TagIDs: []string{"active"}},
	}, filters)

	filters, err = parseDimensionTagFilters(`{"lifecycle":["active"]," department ":["legal"],"department":["finance"]}`)
	require.NoError(t, err)
	assert.Equal(t, []types.DimensionTagFilter{
		{DimensionID: "department", TagIDs: []string{"legal", "finance"}},
		{DimensionID: "lifecycle", TagIDs: []string{"active"}},
	}, filters)
}

func TestParseDimensionTagFiltersRejectsInvalidInput(t *testing.T) {
	_, err := parseDimensionTagFilters(`[{"dimension_id":"","tag_ids":["active"]}]`)
	require.Error(t, err)

	_, err = parseDimensionTagFilters(`not-json`)
	require.Error(t, err)
}
