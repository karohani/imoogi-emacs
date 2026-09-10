package render

import "github.com/karohani/imoogi-emacs/internal/orgpreview"

type Mapping struct {
	ID        string
	StartByte int
	EndByte   int
}

type Page struct {
	Markup   string
	Mappings []Mapping
}

func HTML(doc orgpreview.Document) (Page, error) {
	page := Page{Markup: orgpreview.Renderer{}.Render(doc)}
	for _, node := range orgpreview.Flatten(doc) {
		page.Mappings = append(page.Mappings, Mapping{
			ID:        node.ID,
			StartByte: node.StartByte,
			EndByte:   node.EndByte,
		})
	}
	return page, nil
}
