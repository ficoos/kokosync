package komga

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"path"

	"github.com/ficoos/moonkosync/opds"
)

type ReadingOrder struct {
	Href string `json:"href"`
	Type string `json:"type"`
}

type WebPubManifest struct {
	ReadingOrder []ReadingOrder `json:"readingOrder"`
}

type Client struct {
	httpClient *http.Client
	apiRoot    *url.URL
	apiKey     string
}

func PasswordToApiKey(password string) string {
	h := md5.Sum([]byte(password))
	return hex.EncodeToString(h[:])
}

func New(apiRoot url.URL, apiKey string) *Client {
	return &Client{apiRoot: &apiRoot, apiKey: apiKey, httpClient: http.DefaultClient}
}

func (c *Client) Get(url *url.URL) (io.ReadCloser, error) {
	req := http.Request{
		Method: http.MethodGet,
		URL:    url,
		Header: http.Header{
			"X-API-Key": []string{c.apiKey},
			//"Accept":    []string{"application/json"},
		},
	}
	resp, err := c.httpClient.Do(&req)
	if err != nil {
		return nil, fmt.Errorf("send api request: %s", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		// TODO: see if there is anything useful in the body
		return nil, fmt.Errorf("error response %s [%d]", resp.Status, resp.StatusCode)
	}

	return resp.Body, nil
}

func (c *Client) getRelative(endpoint string) (io.ReadCloser, error) {
	u := *c.apiRoot
	u.Path = path.Join(u.Path, endpoint)

	return c.Get(&u)
}

func (c *Client) getJSON(endpoint string, result any) error {
	body, err := c.getRelative(endpoint)
	if err != nil {
		return err
	}

	defer body.Close()

	dec := json.NewDecoder(body)
	err = dec.Decode(result)
	if err != nil {
		return fmt.Errorf("decode respose: %s", err)
	}

	return nil
}

func (c *Client) getXML(url *url.URL, result any) error {
	body, err := c.Get(url)
	if err != nil {
		return err
	}

	defer body.Close()

	dec := xml.NewDecoder(body)
	err = dec.Decode(result)
	if err != nil {
		return fmt.Errorf("decode respose: %s", err)
	}

	return nil
}

func (c *Client) WebPubManifest(bookID string) (*WebPubManifest, error) {
	var result WebPubManifest
	err := c.getJSON(fmt.Sprintf("/api/v1/books/%s/manifest/epub", bookID), &result)
	if err != nil {
		return nil, fmt.Errorf("get webpub manifest %s: %s", bookID, err)
	}
	return &result, nil
}

func (c *Client) IterateOPDS(url *url.URL) iter.Seq2[*opds.Entry, error] {
	next := url
	return func(yield func(entry *opds.Entry, err error) bool) {
		for next != nil {
			var resp opds.Feed
			err := c.getXML(next, &resp)
			if err != nil {
				yield(nil, err)
				return
			}

			for i, entry := range resp.Entries {
				for _, l := range entry.Links {
					if l.Rel == "subsection" {
						u, err := url.Parse(l.Href)
						if err != nil {
							yield(nil, fmt.Errorf("error parsing subsection url: %s", err))
							return
						}
						ok := true
						c.IterateOPDS(u)(func(entry *opds.Entry, err error) bool {
							ok = yield(entry, err)
							return ok
						})
						if !ok {
							return
						}
					}
					if l.Rel == "http://opds-spec.org/acquisition" {
						// Is part of a series
						if resp.Title != entry.Title || len(resp.Entries) > 1 {
							// TODO: this is a hack to push the series into the title, not actually OPDS confomant
							entry.Title = fmt.Sprintf("%s %d %s", resp.Title, i+1, entry.Title)
						}
						ok := yield(&entry, nil)
						if !ok {
							return
						}
					}
				}
			}

			next = nil
			for _, link := range resp.Links {
				if link.Rel != "next" {
					continue
				}

				next, err = url.Parse(link.Href)
				if err != nil {
					return
				}
				break
			}
		}
	}
}

func (c *Client) IterateOPDSFeed(feed string) iter.Seq2[*opds.Entry, error] {
	var err error
	next, _ := url.Parse(c.apiRoot.String())
	// TODO: actually join path from api root
	next.Path, err = url.JoinPath("/", "opds/v1.2", feed)
	if err != nil {
		return func(yield func(entry *opds.Entry, err error) bool) { yield(nil, err) }
	}

	return c.IterateOPDS(next)
}
