package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	errBadURL   = errors.New("not a post of this instance")
	errNotFound = errors.New("post not found")
	postPathRe  = regexp.MustCompile(`^/(?:@[\w.-]+(?:@[\w.-]+)?|users/[\w.-]+/statuses)/(\d{1,24})/?$`)
)

// parsePostURL returns the id of a post of this instance. A post from another
// server has a local address too, /@name@server/id, and that one works.
func parsePostURL(in, domain string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(in))
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, domain) {
		return "", errBadURL
	}
	m := postPathRe.FindStringSubmatch(u.Path)
	if m == nil {
		return "", errBadURL
	}
	return m[1], nil
}

type emoji struct {
	Shortcode string `json:"shortcode"`
	URL       string `json:"static_url"`
}

type account struct {
	Username    string  `json:"username"`
	Acct        string  `json:"acct"`
	DisplayName string  `json:"display_name"`
	Avatar      string  `json:"avatar_static"`
	Emojis      []emoji `json:"emojis"`
}

type attachment struct {
	Type        string `json:"type"`
	URL         string `json:"url"`
	Preview     string `json:"preview_url"`
	Description string `json:"description"`
}

type pollOption struct {
	Title string `json:"title"`
	Votes int    `json:"votes_count"`
}

type poll struct {
	Options []pollOption `json:"options"`
}

type status struct {
	ID         string       `json:"id"`
	CreatedAt  string       `json:"created_at"`
	Content    string       `json:"content"`
	Spoiler    string       `json:"spoiler_text"`
	Replies    int          `json:"replies_count"`
	Boosts     int          `json:"reblogs_count"`
	Favourites int          `json:"favourites_count"`
	Account    account      `json:"account"`
	Media      []attachment `json:"media_attachments"`
	Emojis     []emoji      `json:"emojis"`
	Poll       *poll        `json:"poll"`
	Reblog     *status      `json:"reblog"`
}

type api struct {
	base   string
	client *http.Client
}

// post reads a post as an anonymous visitor, so only what is public comes back.
func (a *api) post(ctx context.Context, id string) (*status, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.base+"/api/v1/statuses/"+id, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusUnauthorized:
		return nil, errNotFound
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("instance answered %s", resp.Status)
	}
	var st status
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&st); err != nil {
		return nil, err
	}
	if st.Reblog != nil {
		return st.Reblog, nil // a boost's address shows the original
	}
	return &st, nil
}
