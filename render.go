package main

import (
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

// The instance already sanitizes posts. This second pass means a bug there or a
// strange post from a fork can't reach the page.
var postPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "strong", "b", "em", "i", "u", "s", "del", "code", "pre",
		"blockquote", "ul", "ol", "li", "h1", "h2", "h3", "h4", "h5", "h6")
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^(invisible|ellipsis)$`)).OnElements("span")
	p.AllowElements("a", "span")
	return p
}()

var (
	shortcodeRe = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	emojiNameRe = regexp.MustCompile(`:[a-zA-Z0-9_]+:`)
	blockEndRe  = regexp.MustCompile(`(?i)</p>|<br\s*/?>`)
)

// sanitize cleans a post's HTML and turns custom emoji into images. Links stay
// as text: this is a picture, there is nothing to click.
func sanitize(content string, emojis []emoji, proxy func(string) string) template.HTML {
	clean := postPolicy.Sanitize(content)
	for _, e := range emojis {
		if !shortcodeRe.MatchString(e.Shortcode) || !strings.HasPrefix(e.URL, "https://") {
			continue
		}
		tag := `<img class="emoji" src="` + html.EscapeString(proxy(e.URL)) + `" alt=":` + e.Shortcode + `:">`
		clean = strings.ReplaceAll(clean, ":"+e.Shortcode+":", tag)
	}
	return template.HTML(clean) // sanitized above
}

// plainText turns a post into text for the image description.
func plainText(content string) string {
	text := blockEndRe.ReplaceAllString(content, "\n")
	text = html.UnescapeString(bluemonday.StrictPolicy().Sanitize(text))
	var lines []string
	for line := range strings.SplitSeq(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

type picture struct {
	Src, Alt string
}

type pollRow struct {
	Title   string
	Percent int
	Bar     int // rounded to the nearest 5, there's a CSS class for each
}

// card is a post ready for the template.
type card struct {
	Name     string
	NameHTML template.HTML
	Handle   string
	Avatar   string
	Warning  string
	Body     template.HTML
	Pictures []picture
	Poll     []pollRow
	Counters []counter
	Date     string
	Alt      string
	File     string
}

type counter struct {
	Number int
	Label  string
}

func formatDate(stamp string, t map[string]string) string {
	d, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return ""
	}
	months := strings.Split(t["months"], " ")
	return strings.NewReplacer("{d}", strconv.Itoa(d.Day()), "{m}", months[d.Month()-1], "{y}", strconv.Itoa(d.Year())).Replace(t["date"])
}

func newCard(st *status, domain string, t map[string]string, proxy func(string) string) *card {
	name := strings.TrimSpace(st.Account.DisplayName)
	if strings.TrimSpace(emojiNameRe.ReplaceAllString(name, "")) == "" {
		name = st.Account.Username
	}
	if r := []rune(name); len(r) > 40 {
		name = string(r[:40]) + "…"
	}
	handle := st.Account.Acct
	if !strings.Contains(handle, "@") {
		handle += "@" + domain
	}
	c := &card{
		Name:     strings.TrimSpace(emojiNameRe.ReplaceAllString(name, "")),
		NameHTML: sanitize(html.EscapeString(name), st.Account.Emojis, proxy),
		Handle:   "@" + handle,
		Avatar:   proxy(st.Account.Avatar),
		Warning:  st.Spoiler,
		Body:     sanitize(st.Content, st.Emojis, proxy),
		Date:     formatDate(st.CreatedAt, t),
	}
	c.File = "poet-" + st.Account.Username + "-" + st.ID
	alts := []string{}
	for _, m := range st.Media {
		if len(c.Pictures) == 4 {
			break
		}
		if !strings.HasPrefix(m.Preview, "https://") {
			continue
		}
		src := m.Preview
		if m.Type == "image" && strings.HasPrefix(m.URL, "https://") {
			src = m.URL // the preview is too small for a picture this size
		}
		c.Pictures = append(c.Pictures, picture{Src: proxy(src), Alt: m.Description})
		if m.Description != "" {
			alts = append(alts, m.Description)
		}
	}
	if st.Poll != nil {
		total := 0
		for _, o := range st.Poll.Options {
			total += o.Votes
		}
		for _, o := range st.Poll.Options {
			pct := 0
			if total > 0 {
				pct = (o.Votes*100 + total/2) / total
			}
			c.Poll = append(c.Poll, pollRow{Title: o.Title, Percent: pct, Bar: (pct + 2) / 5 * 5})
		}
	}
	for _, v := range []struct {
		n   int
		key string
	}{{st.Replies, "replies"}, {st.Boosts, "boosts"}, {st.Favourites, "favourites"}} {
		switch {
		case v.n == 1:
			c.Counters = append(c.Counters, counter{1, t[v.key+"_one"]})
		case v.n > 1:
			c.Counters = append(c.Counters, counter{v.n, t[v.key+"_many"]})
		}
	}
	text := plainText(st.Content)
	c.Alt = fmt.Sprintf(t["alt"], c.Name, c.Handle, text)
	if len(alts) > 0 {
		c.Alt += "\n" + t["alt_media"] + " " + strings.Join(alts, " / ")
	}
	return c
}
