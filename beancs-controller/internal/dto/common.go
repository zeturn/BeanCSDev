package dto

type ErrorResponse struct {
	Error string `json:"error"`
}

type ListResponse[T any] struct {
	Data    []T   `json:"data"`
	Total   int64 `json:"total"`
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
}
