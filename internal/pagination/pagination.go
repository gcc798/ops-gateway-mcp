package pagination

import (
	"fmt"
	"net/url"
	"strconv"
)

type Query struct {
	Page     int `json:"page,omitempty"`
	PageSize int `json:"page_size,omitempty"`
}

func (q *Query) Validate() error {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.Page < 1 || q.Page > 1000000 || q.PageSize < 1 || q.PageSize > 100 {
		return fmt.Errorf("page must be 1–1000000; page_size must be 1–100")
	}
	return nil
}
func (q Query) Offset() int { return (q.Page - 1) * q.PageSize }
func Parse(v url.Values) (Query, error) {
	q := Query{}
	for key, dst := range map[string]*int{"page": &q.Page, "page_size": &q.PageSize} {
		if value, ok := v[key]; ok {
			if len(value) != 1 {
				return q, fmt.Errorf("invalid %s", key)
			}
			n, err := strconv.Atoi(value[0])
			if err != nil || n < 1 {
				return q, fmt.Errorf("invalid %s", key)
			}
			*dst = n
		}
	}
	err := q.Validate()
	return q, err
}

type Result[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
