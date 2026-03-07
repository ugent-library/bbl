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
	window      int
}

type Page struct {
	Number  int
	Offset  int
	Current bool
}

func New(limit, offset, total int) *Pager {
	p := &Pager{
		Limit:  limit,
		Offset: offset,
		Total:  total,
		window: 10, // TODO make optional and configurable
	}

	p.TotalPages = int(math.Ceil(float64(total) / float64(limit)))
	p.CurrentPage = int(math.Floor(float64(offset)/float64(limit))) + 1
	p.HasPrevPage = p.CurrentPage > 1
	p.HasNextPage = total > offset+limit
	if p.HasPrevPage {
		p.PrevPage = Page{
			Number: p.CurrentPage - 1,
			Offset: (p.CurrentPage - 2) * limit,
		}
	}
	if p.HasNextPage {
		p.NextPage = Page{
			Number: p.CurrentPage + 1,
			Offset: p.CurrentPage * limit,
		}
	}
	if total > 0 {
		p.FirstOnPage = offset + 1
	}
	p.LastOnPage = min(total, offset+limit)

	window := min(p.TotalPages, p.window)
	p.Pages = make([]Page, 0, window)
	beforeCurrent := min(int(math.Ceil(float64(window)/2)), p.CurrentPage) - 1

	for i := beforeCurrent; i > 0; i-- {
		p.Pages = append(p.Pages, Page{
			Number: p.CurrentPage - i,
			Offset: (p.CurrentPage - i - 1) * limit,
		})
	}
	p.Pages = append(p.Pages, Page{
		Number:  p.CurrentPage,
		Offset:  (p.CurrentPage - 1) * limit,
		Current: true,
	})
	for i := 1; len(p.Pages) < window; i++ {
		offset := (p.CurrentPage + i - 1) * limit
		if offset > p.Total {
			break
		}
		p.Pages = append(p.Pages, Page{
			Number: p.CurrentPage + i,
			Offset: offset,
		})
	}

	return p
}
