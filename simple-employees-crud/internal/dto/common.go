// Package dto contains Data Transfer Objects (DTOs) used exclusively at the
// HTTP boundary.  DTOs carry validation tags interpreted by Gin's built-in
// validator (go-playground/validator/v10).  Domain entities are never exposed
// directly to callers; they are always mapped to/from a DTO.
package dto

// PaginationRequest is embedded in list-query DTOs to provide uniform
// page/limit/sort parameters across all collection endpoints.
type PaginationRequest struct {
	// Page is 1-based.  Defaults to 1.
	Page int `form:"page"    json:"page"`
	// Limit is the maximum number of items per page.  Defaults to 20, max 100.
	Limit int `form:"limit"   json:"limit"`
	// SortBy is the column to sort on (validated in the repository layer).
	SortBy string `form:"sort_by"    json:"sort_by"`
	// SortOrder is "asc" or "desc".  Defaults to "asc".
	SortOrder string `form:"sort_order" json:"sort_order"`
}

// Normalise applies sensible defaults and clamps values to acceptable ranges.
func (p *PaginationRequest) Normalise() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.SortOrder != "desc" {
		p.SortOrder = "asc"
	}
}

// Offset returns the SQL OFFSET value derived from Page and Limit.
func (p *PaginationRequest) Offset() int {
	return (p.Page - 1) * p.Limit
}

// SearchRequest extends PaginationRequest with a full-text search query.
type SearchRequest struct {
	PaginationRequest
	// Query is the search string passed to PostgreSQL plainto_tsquery.
	Query string `form:"q" json:"q" binding:"required,min=1,max=200"`
}
