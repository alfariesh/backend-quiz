package pagination

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type Params struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func NewParams(limit, offset int) Params {
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	if offset < 0 {
		offset = 0
	}
	return Params{Limit: limit, Offset: offset}
}

type Response[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Limit      int `json:"limit"`
	Offset     int `json:"offset"`
	HasMore    bool `json:"has_more"`
}

func NewResponse[T any](data []T, total, limit, offset int) Response[T] {
	if data == nil {
		data = []T{}
	}
	return Response[T]{
		Data:    data,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}
}
