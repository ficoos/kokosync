package opds

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	apiRoot    *url.URL
	apiKey     string
	httpClient *http.Client
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
