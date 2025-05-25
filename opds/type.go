package opds

type Feed struct {
	ID      string  `xml:"id"`
	Title   string  `xml:"title"`
	Links   []Link  `xml:"link"`
	Entries []Entry `xml:"entry"`
}

type Link struct {
	Type string `xml:"type,attr"`
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type Entry struct {
	Title   string   `xml:"title"`
	ID      string   `xml:"id"`
	Authors []Author `xml:"author"`
	Links   []Link   `xml:"link"`
}

type Author struct {
	Name string `xml:"name"`
}
