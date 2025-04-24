package pagination

import (
	"math"
)

type Pager struct {
	Limit       int
	Offset      int
	Total       int
	TotalPages  int
	CurrentPage int
	HasPrevPage bool
	HasNextPage bool
	PrevPage    Page
	NextPage    Page
	FirstOnPage int
	LastOnPage  int
	Pages       []Page
}

type Page struct {
	Number  int
	Offset  int
	Current bool
}

func New(limit, offset, total int) *Pager {
	p := &Pager{Limit: limit, Offset: offset, Total: total}
	p.TotalPages = int(math.Ceil(float64(total) / float64(limit)))
	p.CurrentPage = int(math.Floor(float64(offset)/float64(limit))) + 1
	p.HasPrevPage = p.CurrentPage > 1
	p.HasNextPage = total > offset+limit
	if p.HasPrevPage {
		p.PrevPage = Page{
			Number: p.CurrentPage + 1,
			Offset: p.CurrentPage * limit,
		}
	}
	if p.HasNextPage {
		p.NextPage = Page{
			Number: p.CurrentPage - 1,
			Offset: (p.CurrentPage - 2) * limit,
		}
	}
	if total > 0 {
		p.FirstOnPage = offset + 1
	}
	p.LastOnPage = min(total, offset+limit)
	p.Pages = make([]Page, p.TotalPages)
	for i := 0; i < p.TotalPages; i++ {
		p.Pages[i] = Page{
			Number:  i + 1,
			Offset:  i * limit,
			Current: i+1 == p.CurrentPage,
		}
	}
	return p
}
