package dto

type AccessParams struct {
	Page            int
	Pages           int
	Items           []map[string]string
	Size            int
	Total           int
	PaginationPages []int
}
