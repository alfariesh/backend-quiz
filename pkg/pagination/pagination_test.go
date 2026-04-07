package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewParams_Defaults(t *testing.T) {
	p := NewParams(0, 0)
	assert.Equal(t, DefaultLimit, p.Limit)
	assert.Equal(t, 0, p.Offset)
}

func TestNewParams_NegativeLimit(t *testing.T) {
	p := NewParams(-5, 0)
	assert.Equal(t, DefaultLimit, p.Limit)
}

func TestNewParams_ExceedsMax(t *testing.T) {
	p := NewParams(500, 0)
	assert.Equal(t, DefaultLimit, p.Limit)
}

func TestNewParams_ValidValues(t *testing.T) {
	p := NewParams(50, 10)
	assert.Equal(t, 50, p.Limit)
	assert.Equal(t, 10, p.Offset)
}

func TestNewParams_NegativeOffset(t *testing.T) {
	p := NewParams(10, -5)
	assert.Equal(t, 0, p.Offset)
}

func TestNewResponse_HasMore(t *testing.T) {
	data := []string{"a", "b", "c"}
	resp := NewResponse(data, 10, 3, 0)

	assert.Len(t, resp.Data, 3)
	assert.Equal(t, 10, resp.Total)
	assert.True(t, resp.HasMore)
}

func TestNewResponse_NoMore(t *testing.T) {
	data := []string{"a", "b", "c"}
	resp := NewResponse(data, 3, 10, 0)

	assert.Len(t, resp.Data, 3)
	assert.Equal(t, 3, resp.Total)
	assert.False(t, resp.HasMore)
}

func TestNewResponse_NilData(t *testing.T) {
	resp := NewResponse[string](nil, 0, 10, 0)

	assert.NotNil(t, resp.Data)
	assert.Len(t, resp.Data, 0)
}

func TestNewResponse_LastPage(t *testing.T) {
	data := []string{"a", "b"}
	resp := NewResponse(data, 12, 5, 10)

	assert.False(t, resp.HasMore) // 10+5 >= 12
}
