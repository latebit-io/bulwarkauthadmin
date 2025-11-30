package shared

type PageOptions struct {
	page   int
	size   int
	sortBy string
}

func NewPageOptions(page, size int, sortBy string) PageOptions {
	return PageOptions{
		page:   page,
		size:   size,
		sortBy: sortBy,
	}
}

func (o PageOptions) Page() int {
	return o.page
}
func (o PageOptions) Size() int {
	return o.size
}

func (o PageOptions) SortBy() string {
	return o.sortBy
}
