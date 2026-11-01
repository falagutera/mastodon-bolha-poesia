package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cacheTTL   = time.Minute
	cacheLimit = 256
	maxMedia   = 16 << 20
)

var (
	themes      = []string{"dark", "light"}
	backgrounds = []string{"amber", "violet", "teal", "pink", "night", "plain"}
	footers     = []string{"full", "date", "none"}
	marks       = []string{"on", "off"}
	imageTypes  = []string{"image/jpeg", "image/png", "image/webp", "image/gif", "image/avif"}
)

const (
	defaultPad = 5
	maxPad     = 8
)

// options are what the person picks. They live in the address, so a preview can
// be shared as a link.
type options struct {
	Theme, Background, Footer, Mark string
	Pad                             int
}

func pick(v string, allowed []string) string {
	if slices.Contains(allowed, v) {
		return v
	}
	return allowed[0]
}

func parseOptions(q url.Values) options {
	pad, err := strconv.Atoi(q.Get("pad"))
	if err != nil || pad < 0 || pad > maxPad {
		pad = defaultPad
	}
	return options{
		Theme:      pick(q.Get("theme"), themes),
		Background: pick(q.Get("bg"), backgrounds),
		Footer:     pick(q.Get("footer"), footers),
		Mark:       pick(q.Get("mark"), marks),
		Pad:        pad,
	}
}

// link is the address of this page with one option changed.
func (o options) link(post, key, value string) string {
	q := url.Values{"theme": {o.Theme}, "bg": {o.Background}, "footer": {o.Footer}, "mark": {o.Mark}, "pad": {strconv.Itoa(o.Pad)}}
	q.Set(key, value)
	if post != "" {
		q.Set("url", post)
	}
	return "/?" + q.Encode()
}

type choice struct {
	Label, Value, Href string
	On                 bool
}

type cached struct {
	at  time.Time
	st  *status
	err error
}

type server struct {
	cfg    config
	tmpl   *template.Template
	client *http.Client
	media  *http.Client
	api    *api
	key    []byte
	slots  chan struct{}

	mu    sync.Mutex
	cache map[string]cached
}

func newServer(cfg config, tmpl *template.Template, client *http.Client, key []byte) *server {
	return &server{cfg: cfg, tmpl: tmpl, client: client, media: mediaClient(15 * time.Second), api: &api{base: cfg.instance, client: client}, key: key,
		slots: make(chan struct{}, 16), cache: map[string]cached{}}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "User-agent: *\nDisallow: /\n")
	})
	assets, err := fs.Sub(assetFS, "assets")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(assets)))
	mux.HandleFunc("GET /media/{sig}/{target}", s.handleMedia)
	mux.HandleFunc("GET /{$}", s.handleIndex)
	return securityHeaders(mux)
}

// securityHeaders: scripts are the app's own, data: and blob: are how the capture
// library builds the image, and base-uri 'self' lets it resolve the fonts. Every
// picture comes through /media.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("X-Robots-Tag", "noindex, nofollow")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; connect-src 'self'; style-src 'self'; font-src 'self' data:; img-src 'self' data: blob:; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// proxy signs a media address that came from the API. The signature is what
// makes /media safe: it can't be used to fetch anything else.
func (s *server) proxy(target string) string {
	enc := base64.RawURLEncoding.EncodeToString([]byte(target))
	return "/media/" + s.sign(enc) + "/" + enc
}

func (s *server) sign(enc string) string {
	m := hmac.New(sha256.New, s.key)
	m.Write([]byte(enc))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (s *server) handleMedia(w http.ResponseWriter, r *http.Request) {
	enc := r.PathValue("target")
	if !hmac.Equal([]byte(r.PathValue("sig")), []byte(s.sign(enc))) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	target, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil || !strings.HasPrefix(string(target), "https://") {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-r.Context().Done():
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, string(target), nil)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	resp, err := s.media.Do(req)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	kind, _, _ := strings.Cut(resp.Header.Get("Content-Type"), ";")
	kind = strings.ToLower(strings.TrimSpace(kind))
	if resp.StatusCode != http.StatusOK || !slices.Contains(imageTypes, kind) || resp.ContentLength > maxMedia {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", kind)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	io.Copy(w, io.LimitReader(resp.Body, maxMedia))
}

type pageData struct {
	Version   string
	T         map[string]string
	Domain    string
	Host      string
	Post      string
	Message   string
	Card      *card
	Options   options
	Themes    []choice
	Footers   []choice
	Marks     []choice
	Backdrops []choice
	MaxPad    int
}

func (s *server) page(r *http.Request) pageData {
	return pageData{Version: version, T: strs(language(r, s.cfg.lang)), Domain: s.cfg.domain, Host: s.cfg.host, MaxPad: maxPad}
}

func (s *server) render(w http.ResponseWriter, status int, data pageData) {
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, "index.html", data); err != nil {
		log.Printf("template: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if lang := q.Get("lang"); lang != "" {
		if _, ok := translations[lang]; ok {
			http.SetCookie(w, &http.Cookie{Name: langCookie, Value: lang, Path: "/", MaxAge: 365 * 24 * 3600, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
		}
		q.Del("lang")
		target := "/"
		if len(q) > 0 {
			target += "?" + q.Encode()
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	d := s.page(r)
	d.Post = strings.TrimSpace(q.Get("url"))
	d.Options = parseOptions(q)
	d.fillChoices()
	if d.Post == "" {
		s.render(w, http.StatusOK, d)
		return
	}
	id, err := parsePostURL(d.Post, s.cfg.domain)
	if err == nil {
		var st *status
		if st, err = s.status(r.Context(), id); err == nil {
			d.Card = newCard(st, s.cfg.domain, d.T, s.proxy)
			s.render(w, http.StatusOK, d)
			return
		}
	}
	status := http.StatusBadGateway
	key := "unavailable"
	switch {
	case errors.Is(err, context.Canceled):
		return
	case errors.Is(err, errBadURL):
		status, key = http.StatusUnprocessableEntity, "url"
	case errors.Is(err, errNotFound):
		status, key = http.StatusNotFound, "notfound"
	default:
		log.Printf("post %s: %v", id, err)
	}
	d.Message = strings.ReplaceAll(d.T["error_"+key], "%s", s.cfg.domain)
	s.render(w, status, d)
}

func (d *pageData) fillChoices() {
	set := func(key string, values []string, current string) []choice {
		var out []choice
		for _, v := range values {
			out = append(out, choice{Label: d.T[key+"_"+v], Value: v, Href: d.Options.link(d.Post, key, v), On: v == current})
		}
		return out
	}
	d.Themes = set("theme", themes, d.Options.Theme)
	d.Backdrops = set("bg", backgrounds, d.Options.Background)
	d.Footers = set("footer", footers, d.Options.Footer)
	d.Marks = set("mark", marks, d.Options.Mark)
}

// status reads a post through a short cache, errors included.
func (s *server) status(ctx context.Context, id string) (*status, error) {
	s.mu.Lock()
	c, ok := s.cache[id]
	s.mu.Unlock()
	if ok && time.Since(c.at) < cacheTTL {
		return c.st, c.err
	}
	st, err := s.api.post(ctx, id)
	if errors.Is(err, context.Canceled) {
		return nil, err
	}
	s.mu.Lock()
	if len(s.cache) >= cacheLimit {
		clear(s.cache)
	}
	s.cache[id] = cached{at: time.Now(), st: st, err: err}
	s.mu.Unlock()
	return st, err
}
