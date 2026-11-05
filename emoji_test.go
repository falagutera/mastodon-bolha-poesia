package main

import (
	"strings"
	"testing"
)

// A :shortcode: inside a link href must NOT be turned into an <img> (which would
// break out of the attribute); only the one in the text run becomes an image.
func TestEmojiOnlyInTextNodes(t *testing.T) {
	in := `<p><a href="https://example.com/:blob:">x</a> :blob:</p>`
	emojis := []emoji{{Shortcode: "blob", URL: "https://m.example/b.png"}}
	out := string(sanitize(in, emojis, func(u string) string { return u }))
	if !strings.Contains(out, `href="https://example.com/:blob:"`) {
		t.Errorf("shortcode in href was altered: %s", out)
	}
	if strings.Count(out, "<img") != 1 {
		t.Errorf("expected exactly one emoji image, got: %s", out)
	}
}
