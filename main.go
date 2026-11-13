// poesia turns a public post of a Mastodon instance into an image to share. See
// README.md.
package main

import (
	"crypto/rand"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed assets
var assetFS embed.FS

type config struct {
	listen   string
	host     string
	instance string
	domain   string
	lang     string
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadConfig() (config, error) {
	c := config{
		listen:   env("POET_LISTEN", ":8080"),
		instance: strings.TrimRight(os.Getenv("POET_INSTANCE"), "/"),
		lang:     env("POET_LANG", "en"),
	}
	inst, err := url.Parse(c.instance)
	if err != nil || inst.Host == "" {
		return c, fmt.Errorf("POET_INSTANCE must be a URL like https://example.social")
	}
	c.domain = inst.Host
	base, err := url.Parse(os.Getenv("POET_BASE_URL"))
	if err != nil || base.Host == "" {
		return c, fmt.Errorf("POET_BASE_URL must be a URL like https://poesia.example.social")
	}
	c.host = base.Host
	return c, nil
}

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the local server and exit")
	flag.Parse()
	if *healthcheck {
		resp, err := http.Get("http://127.0.0.1" + env("POET_LISTEN", ":8080") + "/healthz")
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		return
	}
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
	// Media links are only valid while this process runs, so no secret to configure.
	key := make([]byte, 32)
	rand.Read(key)
	s := newServer(cfg, tmpl, &http.Client{Timeout: 15 * time.Second}, key)
	log.Printf("listening on %s for %s", cfg.listen, cfg.domain)
	srv := &http.Server{
		Addr:              cfg.listen,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
