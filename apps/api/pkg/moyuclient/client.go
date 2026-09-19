// Package moyuclient reads the NextMoe moyu face at /v2/moyu: the patch
// resources 鲲 Galgame 补丁 (www.moyu.moe) holds for a game.
package moyuclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotConfigured = errors.New("moyuclient: not configured (empty base URL or API key)")
	ErrUpstream      = errors.New("moyuclient: moyu face error")
)

type Config struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		apiKey:     cfg.APIKey,
		httpClient: hc,
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.baseURL != "" && c.apiKey != ""
}

type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// Resource carries no download link, code or password by design; WebURL is the
// way to the file.
type Resource struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Storage       string    `json:"storage"`
	Size          string    `json:"size"`
	ModelName     string    `json:"model_name"`
	Note          string    `json:"note"`
	Type          []string  `json:"type"`
	Language      []string  `json:"language"`
	Platform      []string  `json:"platform"`
	DownloadCount int       `json:"download_count"`
	WebURL        string    `json:"web_url"`
	UpdatedAt     time.Time `json:"updated_at"`
	Publisher     *User     `json:"publisher"`
}

type Patch struct {
	ID        string     `json:"id"`
	WebURL    string     `json:"web_url"`
	Resources []Resource `json:"resources"`
}

// PatchesForWork answers moyu's pages for one catalog work with their resources
// and publishers attached. It is usually one page, but moyu dedupes on the VNDB
// string, so a game that arrived under two spellings has two; the page a reader
// should land on comes first.
func (c *Client) PatchesForWork(ctx context.Context, catalogWorkID int64) ([]Patch, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	q := url.Values{
		"refs":    {"catalog:" + strconv.FormatInt(catalogWorkID, 10)},
		"include": {"resources,publisher"},
		// The face defaults to nsfw=false, which drops every page catalog rates
		// as adult — most of the galgames this forum lists.
		"nsfw": {"true"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v2/moyu/patches?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w (status %d): %s", ErrUpstream, resp.StatusCode, problemDetail(raw))
	}

	var list struct {
		Items []Patch `json:"items"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("%w: malformed patch list: %v", ErrUpstream, err)
	}
	return list.Items, nil
}

func problemDetail(raw []byte) string {
	var p struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	}
	if json.Unmarshal(raw, &p) != nil || p.Code == "" {
		return "non-problem body"
	}
	return p.Code + ": " + p.Detail
}
