package main

import (
	"net/http"
	"strings"
)

const langCookie = "__Host-poet-lang"

var translations = map[string]map[string]string{
	"en": {
		"lang":              "en",
		"language":          "Language",
		"title":             "Poet",
		"lede":              "Turn a post from %s into a nice image to share.",
		"url_label":         "Post address",
		"url_hint":          "https://%s/@name/123456789",
		"generate":          "Generate",
		"opt_theme":         "Theme",
		"opt_bg":            "Background",
		"opt_footer":        "Footer",
		"opt_pad":           "Margin",
		"opt_mark":          "Credit",
		"apply":             "Apply",
		"theme_dark":        "Dark",
		"theme_light":       "Light",
		"bg_amber":          "Amber",
		"bg_violet":         "Violet",
		"bg_teal":           "Teal",
		"bg_pink":           "Pink",
		"bg_night":          "Night",
		"bg_plain":          "Plain",
		"footer_full":       "Numbers and date",
		"footer_date":       "Date only",
		"footer_none":       "None",
		"mark_on":           "Show",
		"mark_off":          "Hide",
		"download":          "Download PNG",
		"download_pdf":      "Download PDF",
		"copy_alt":          "Copy alt text",
		"copied":            "Copied",
		"alt_title":         "Alt text",
		"alt":               "Screenshot of a post by %s (%s): %s",
		"alt_media":         "Attached:",
		"warning":           "Content warning:",
		"replies_one":       "reply",
		"replies_many":      "replies",
		"boosts_one":        "boost",
		"boosts_many":       "boosts",
		"favourites_one":    "favourite",
		"favourites_many":   "favourites",
		"months":            "Jan Feb Mar Apr May Jun Jul Aug Sep Oct Nov Dec",
		"date":              "{m} {d}, {y}",
		"source":            "Source code",
		"credit":            "Based on Mastopoet",
		"error_url":         "That isn't the address of a post on %s.",
		"error_notfound":    "Post not found. It may be private or deleted.",
		"error_unavailable": "That didn't work. Please try again in a moment.",
		"error_capture":     "The image couldn't be generated.",
	},
	"pt": {
		"lang":              "pt-BR",
		"language":          "Idioma",
		"title":             "Poet",
		"lede":              "Transforme um post da %s em uma imagem bonita para compartilhar.",
		"url_label":         "Endereço do post",
		"url_hint":          "https://%s/@nome/123456789",
		"generate":          "Gerar",
		"opt_theme":         "Tema",
		"opt_bg":            "Fundo",
		"opt_footer":        "Rodapé",
		"opt_pad":           "Margem",
		"opt_mark":          "Crédito",
		"apply":             "Aplicar",
		"theme_dark":        "Escuro",
		"theme_light":       "Claro",
		"bg_amber":          "Âmbar",
		"bg_violet":         "Violeta",
		"bg_teal":           "Turquesa",
		"bg_pink":           "Rosa",
		"bg_night":          "Noite",
		"bg_plain":          "Liso",
		"footer_full":       "Números e data",
		"footer_date":       "Só data",
		"footer_none":       "Nada",
		"mark_on":           "Mostrar",
		"mark_off":          "Esconder",
		"download":          "Baixar PNG",
		"download_pdf":      "Baixar PDF",
		"copy_alt":          "Copiar texto alternativo",
		"copied":            "Copiado",
		"alt_title":         "Texto alternativo",
		"alt":               "Captura de um post de %s (%s): %s",
		"alt_media":         "Anexos:",
		"warning":           "Aviso de conteúdo:",
		"replies_one":       "resposta",
		"replies_many":      "respostas",
		"boosts_one":        "impulso",
		"boosts_many":       "impulsos",
		"favourites_one":    "favorito",
		"favourites_many":   "favoritos",
		"months":            "jan fev mar abr mai jun jul ago set out nov dez",
		"date":              "{d} {m} {y}",
		"source":            "Código-fonte",
		"credit":            "Baseado no Mastopoet",
		"error_url":         "Esse não é o endereço de um post da %s.",
		"error_notfound":    "Post não encontrado. Ele pode ser privado ou ter sido apagado.",
		"error_unavailable": "Não deu certo. Tente de novo em instantes.",
		"error_capture":     "Não consegui gerar a imagem.",
	},
}

// language picks the page language: the visitor's choice, then their browser,
// then the instance default.
func language(r *http.Request, fallback string) string {
	if c, err := r.Cookie(langCookie); err == nil {
		if _, ok := translations[c.Value]; ok {
			return c.Value
		}
	}
	for part := range strings.SplitSeq(r.Header.Get("Accept-Language"), ",") {
		tag, _, _ := strings.Cut(strings.TrimSpace(part), ";")
		base, _, _ := strings.Cut(strings.ToLower(tag), "-")
		if _, ok := translations[base]; ok {
			return base
		}
	}
	return fallback
}

func strs(lang string) map[string]string {
	if t, ok := translations[lang]; ok {
		return t
	}
	return translations["en"]
}
