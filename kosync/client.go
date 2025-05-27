package kosync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ficoos/kokosync/urlutil"
)

type Progress struct {
	Document   string  `json:"document"`
	Progress   string  `json:"progress"`
	Percentage float64 `json:"percentage"`
	Device     string  `json:"device"`
	DeviceID   string  `json:"device_id"`
}

func (p *Progress) String() string {
	return fmt.Sprintf(
		"Progress[document=%s, progress=%s, percentage=%.2f%%, device=%s, device-id=%s]",
		p.Document,
		p.Progress,
		p.Percentage,
		p.Device,
		p.DeviceID,
	)
}

type UpdateProgressResult struct {
	Document  string `json:"document"`
	Timestamp int64  `json:"timestamp"`
}

type Client struct {
	userName   string
	userKey    string
	apiRoot    *url.URL
	httpClient *http.Client
}

func NewClient(apiRoot *url.URL, userName, userKey string) *Client {
	return &Client{
		userName:   userName,
		userKey:    userKey,
		apiRoot:    apiRoot,
		httpClient: http.DefaultClient,
	}
}

func (c *Client) request(method string, url *url.URL, payload any, result any) error {
	var body io.ReadCloser
	if payload != nil {
		var buff bytes.Buffer
		enc := json.NewEncoder(&buff)
		err := enc.Encode(payload)
		if err != nil {
			return fmt.Errorf("encode payload: %s", err)
		}
		body = io.NopCloser(bytes.NewReader(buff.Bytes()))
	}
	req := &http.Request{
		Method: method,
		URL:    url,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Auth-User":  []string{c.userName},
			"X-Auth-Key":   []string{c.userKey},
		},
		Body: body,
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send http request")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return ErrUnauthorized
		}
		if resp.StatusCode == http.StatusNotFound {
			return ErrDocNotFound
		}
		return fmt.Errorf("server retuned an error: %s [%d]", resp.Status, resp.StatusCode)
	}

	defer resp.Body.Close()

	if result != nil {
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(result)
		if err != nil {
			return fmt.Errorf("unmarshak result: %s", err)
		}
	}

	return nil
}

func (c *Client) Progress(document string) (*Progress, error) {
	u := urlutil.Join(c.apiRoot, "/syncs/progress/", document)
	var result Progress
	// TODO: handle not found
	err := c.request(http.MethodGet, u, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) UpdateProgress(progress *Progress) (*UpdateProgressResult, error) {
	var result UpdateProgressResult
	err := c.request(http.MethodPut, urlutil.Join(c.apiRoot, "/syncs/progress"), progress, &result)
	return &result, err
}

func (c *Client) Authorize() error {
	return c.request(http.MethodGet, urlutil.Join(c.apiRoot, "/users/auth"), nil, nil)
}
