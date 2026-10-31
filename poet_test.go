package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParsePostURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://bolha.us/@guto/123":              "123",
		" https://bolha.us/@guto/123/ ":           "123",
		"https://BOLHA.US/@a.b_c/9":               "9",
		"https://bolha.us/@far@other.social/55":   "55",
		"https://bolha.us/users/guto/statuses/77": "77",
		"https://bolha.us/@guto/123?x=1":          "123",
	} {
		if got, err := parsePostURL(in, "bolha.us"); err != nil || got != want {
			t.Errorf("%q = %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"", "bolha.us/@guto/1", "http://bolha.us/@guto/1", "https://other.social/@guto/1", "https://bolha.us.evil.example/@guto/1",
		"https://bolha.us/@guto", "https://bolha.us/@guto/abc", "https://bolha.us/@guto/1/../2", "https://bolha.us@evil.example/@guto/1"} {
		if _, err := parsePostURL(in, "bolha.us"); !errors.Is(err, errBadURL) {
			t.Errorf("%q accepted", in)
		}
	}
}

func TestOptionsFallBack(t *testing.T) {
	o := parseOptions(url.Values{"theme": {"x"}, "bg": {"<b>"}, "footer": {"y"}, "mark": {"z"}, "pad": {"99"}})
	if o != (options{"dark", "amber", "full", "on", defaultPad}) {
		t.Fatalf("%+v", o)
	}
	o = parseOptions(url.Values{"theme": {"light"}, "bg": {"night"}, "footer": {"none"}, "mark": {"off"}, "pad": {"0"}})
	if o != (options{"light", "night", "none", "off", 0}) {
		t.Fatalf("%+v", o)
	}
}

func TestPlainText(t *testing.T) {
	got := plainText(`<p>olá &amp; tudo<br>bem</p><p><a href="x"><span class="invisible">https://</span>site</a></p>`)
	if got != "olá & tudo\nbem\nhttps://site" {
		t.Fatalf("%q", got)
	}
}

func TestSanitize(t *testing.T) {
	proxy := func(u string) string { return "/m?" + u }
	got := string(sanitize(`<p onclick="x()">oi<script>alert(1)</script><a href="javascript:alert(1)">a</a><img src="https://x/y" onerror="z()"><iframe></iframe><span class="invisible">v</span></p>`, nil, proxy))
	for _, bad := range []string{"onclick", "<script", "alert(1)", "javascript:", "onerror", "<iframe", "<img", "href"} {
		if strings.Contains(got, bad) {
			t.Errorf("%q survived: %s", bad, got)
		}
	}
	got = string(sanitize("<p>:ok: :bad:</p>", []emoji{{"ok", "https://x/o.png"}, {"bad", "javascript:alert(1)"}, {`"><script>`, "https://x/s.png"}}, proxy))
	if !strings.Contains(got, `src="/m?https://x/o.png"`) || strings.Contains(got, "javascript") || strings.Contains(got, "<script") {
		t.Fatal(got)
	}
}

type fakeInstance struct {
	posts atomic.Int32
	media func(w http.ResponseWriter, r *http.Request)
}

func (f *fakeInstance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/statuses/1":
		f.posts.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"id": "1", "created_at": "2026-10-31T05:00:00Z", "content": "<p>olá <b>mundo</b><script>x</script> :blob:</p>", "spoiler_text": "cuidado",
			"replies_count": 0, "reblogs_count": 1, "favourites_count": 3,
			"account":           map[string]any{"username": "guto", "acct": "guto", "display_name": "<i>Guto</i> :me:", "avatar_static": "https://m.example/a.png", "emojis": []map[string]any{{"shortcode": "me", "static_url": "https://m.example/me.png"}}},
			"media_attachments": []map[string]any{{"type": "image", "url": "https://m.example/f.png", "preview_url": "https://m.example/p.png", "description": "um gato"}, {"type": "video", "preview_url": "http://insecure.example/v.png"}},
			"emojis":            []map[string]any{{"shortcode": "blob", "static_url": "https://m.example/b.png"}},
			"poll":              map[string]any{"options": []map[string]any{{"title": "sim", "votes_count": 3}, {"title": "não", "votes_count": 1}}}})
	case "/api/v1/statuses/2":
		json.NewEncoder(w).Encode(map[string]any{"id": "2", "reblog": map[string]any{"id": "3", "content": "<p>original</p>", "account": map[string]any{"username": "far", "acct": "far@other.social"}}})
	case "/ok.png":
		w.Header().Set("Content-Type", "image/png; x=1")
		fmt.Fprint(w, "PNG")
	case "/bad.svg":
		w.Header().Set("Content-Type", "image/svg+xml")
		fmt.Fprint(w, "<svg onload=x>")
	default:
		w.WriteHeader(404)
	}
}

func testServer(t *testing.T) (*server, *httptest.Server, *fakeInstance, *httptest.Server) {
	t.Helper()
	inst := &fakeInstance{}
	is := httptest.NewServer(inst)
	t.Cleanup(is.Close)
	cfg := config{domain: "bolha.us", host: "poet.bolha.us", lang: "pt", instance: is.URL}
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))
	s := newServer(cfg, tmpl, &http.Client{Timeout: time.Second}, []byte(strings.Repeat("k", 32)))
	ts := httptest.NewServer(s.routes())
	t.Cleanup(ts.Close)
	return s, ts, inst, is
}

func get(t *testing.T, u string) (*http.Response, string) {
	t.Helper()
	c := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	b.ReadFrom(resp.Body)
	return resp, b.String()
}

func page(id string) string { return "/?url=" + url.QueryEscape("https://bolha.us/@guto/"+id) }

func TestCard(t *testing.T) {
	_, ts, inst, _ := testServer(t)
	resp, body := get(t, ts.URL+page("1"))
	if resp.StatusCode != 200 {
		t.Fatalf("%d", resp.StatusCode)
	}
	for _, want := range []string{"mundo", "@guto@bolha.us", "cuidado", "75%", "w75", "25%", `<b>3</b> favoritos`, "<b>1</b> impulso", "poet.bolha.us", "poet-guto-1.png", "um gato", `class="emoji"`, "&lt;i&gt;Guto&lt;/i&gt;"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, bad := range []string{"<script>x", "resposta<", "m.example", "insecure.example", "<i>Guto"} {
		if strings.Contains(body, bad) {
			t.Errorf("%q on the page", bad)
		}
	}
	if !strings.Contains(resp.Header.Get("Content-Security-Policy"), "default-src 'none'") {
		t.Fatal("missing CSP")
	}
	get(t, ts.URL+page("1"))
	if n := inst.posts.Load(); n != 1 {
		t.Fatalf("not cached: %d", n)
	}
}

func TestOptionsChangeThePage(t *testing.T) {
	_, ts, _, _ := testServer(t)
	_, body := get(t, ts.URL+page("1")+"&theme=light&bg=teal&footer=none&mark=off&pad=2")
	for _, want := range []string{`class="card light"`, "bg-teal pad-2"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, bad := range []string{`class="tail"`, `class="mark"`} {
		if strings.Contains(body, bad) {
			t.Errorf("%q on the page", bad)
		}
	}
}

func TestBoostShowsTheOriginal(t *testing.T) {
	_, ts, _, _ := testServer(t)
	if _, body := get(t, ts.URL+page("2")); !strings.Contains(body, "original") || !strings.Contains(body, "@far@other.social") {
		t.Fatal(body)
	}
}

func TestErrors(t *testing.T) {
	_, ts, _, _ := testServer(t)
	for u, code := range map[string]int{page("404"): 404, "/?url=" + url.QueryEscape("https://other.social/@a/1"): 422, "/?url=lixo": 422, "/@guto": 404} {
		resp, body := get(t, ts.URL+u)
		if resp.StatusCode != code || strings.Contains(body, "%!") {
			t.Errorf("%s: %d", u, resp.StatusCode)
		}
	}
}

func TestMediaOnlyServesSignedImages(t *testing.T) {
	s, ts, _, is := testServer(t)
	if resp, body := get(t, ts.URL+s.proxy(strings.Replace(is.URL, "http://", "https://", 1)+"/ok.png")); resp.StatusCode != 502 || body == "" {
		t.Fatalf("no TLS on the fake: %d", resp.StatusCode)
	}
	enc := base64.RawURLEncoding.EncodeToString([]byte(is.URL + "/ok.png"))
	if resp, _ := get(t, ts.URL+"/media/"+s.sign(enc)+"/"+enc); resp.StatusCode != 400 {
		t.Fatalf("plain http: %d", resp.StatusCode)
	}
	if resp, _ := get(t, ts.URL+"/media/forged/"+enc); resp.StatusCode != 403 {
		t.Fatalf("forged: %d", resp.StatusCode)
	}
	s2 := newServer(s.cfg, s.tmpl, &http.Client{Transport: rewriteToHTTP{}}, s.key)
	ts2 := httptest.NewServer(s2.routes())
	defer ts2.Close()
	t.Setenv("FAKE", is.URL)
	resp, body := get(t, ts2.URL+s.proxy("https://x.example/ok.png"))
	if resp.StatusCode != 200 || body != "PNG" || resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("image: %d %q", resp.StatusCode, body)
	}
	if resp, _ := get(t, ts2.URL+s.proxy("https://x.example/bad.svg")); resp.StatusCode != 502 {
		t.Fatalf("svg served: %d", resp.StatusCode)
	}
}

// rewriteToHTTP sends the proxy's https requests to the plain http fake.
type rewriteToHTTP struct{}

func (rewriteToHTTP) RoundTrip(r *http.Request) (*http.Response, error) {
	u, _ := url.Parse(os.Getenv("FAKE") + r.URL.Path)
	req := r.Clone(r.Context())
	req.URL, req.Host = u, u.Host
	return http.DefaultTransport.RoundTrip(req)
}

func TestLanguage(t *testing.T) {
	_, ts, _, _ := testServer(t)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+page("1"), nil)
	req.AddCookie(&http.Cookie{Name: langCookie, Value: "en"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	b.ReadFrom(resp.Body)
	if !strings.Contains(b.String(), `<html lang="en">`) || !strings.Contains(b.String(), "3</b> favourites") {
		t.Fatal("english page")
	}
	if r, _ := get(t, ts.URL+"/?lang=en&url=x"); r.StatusCode != 303 || r.Header.Get("Location") != "/?url=x" {
		t.Fatalf("switch: %d %s", r.StatusCode, r.Header.Get("Location"))
	}
}

func TestSettingsAreRequired(t *testing.T) {
	t.Setenv("POET_BASE_URL", "https://poet.example.social")
	t.Setenv("POET_INSTANCE", "")
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "POET_INSTANCE") {
		t.Fatalf("started without an instance: %v", err)
	}
}
