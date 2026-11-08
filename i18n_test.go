package main

import (
	"sort"
	"strings"
	"testing"
)

// Every text exists in both languages: a key in one map only would show up
// empty (or as a broken format) on the other language's page.
func TestTextsInBothLanguages(t *testing.T) {
	for _, pair := range [][2]string{{"pt", "en"}, {"en", "pt"}} {
		var missing []string
		for k := range translations[pair[0]] {
			if _, ok := translations[pair[1]][k]; !ok {
				missing = append(missing, k)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("in %q but not in %q: %v", pair[0], pair[1], missing)
		}
	}
	if len(translations) != 2 {
		t.Errorf("%d languages: the selector and this test know pt and en", len(translations))
	}
	for lang, tr := range translations {
		if n := len(strings.Fields(tr["months"])); n != 12 {
			t.Errorf("%s: %d month names", lang, n)
		}
		// fillChoices labels each option with the text "<key>_<value>".
		for key, values := range map[string][]string{"theme": themes, "bg": backgrounds, "footer": footers, "mark": marks} {
			for _, v := range values {
				if tr[key+"_"+v] == "" {
					t.Errorf("%s: no label %s_%s", lang, key, v)
				}
			}
		}
	}
}
