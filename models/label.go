package models

import (
	"strings"
	"unicode"
)

type LabelForm string

const (
	LabelPreferred   LabelForm = "preferred"
	LabelAlternative LabelForm = "alternative"
	LabelHidden      LabelForm = "hidden"
)

type Label struct {
	Text string    `json:"text"`
	Lang string    `json:"lang,omitempty"`
	Form LabelForm `json:"form"`
}

type Abbreviations map[string]map[string]string

func (f LabelForm) Known() bool {
	return f == LabelPreferred || f == LabelAlternative || f == LabelHidden
}

func LabelsWellFormed(labels []Label) bool {
	preferred := map[string]bool{}
	for _, l := range labels {
		if strings.TrimSpace(l.Text) == "" || !l.Form.Known() {
			return false
		}
		if l.Form == LabelPreferred {
			if preferred[l.Lang] {
				return false
			}
			preferred[l.Lang] = true
		}
	}
	return true
}

func NormalLabel(s string) string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return unicode.IsSpace(r) || r == '.' || r == ','
	})
	return strings.Join(fields, " ")
}

func ExpandLabels(labels []Label, table Abbreviations) []Label {
	seen := map[string]bool{}
	for _, l := range labels {
		seen[l.Lang+"\x00"+NormalLabel(l.Text)] = true
	}
	out := append([]Label(nil), labels...)
	for _, l := range labels {
		pairs := table[l.Lang]
		if len(pairs) == 0 {
			continue
		}
		for _, form := range rewrites(NormalLabel(l.Text), pairs) {
			key := l.Lang + "\x00" + form
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, Label{Text: form, Lang: l.Lang, Form: LabelHidden})
		}
	}
	return out
}

func rewrites(normal string, pairs map[string]string) []string {
	swap := map[string]string{}
	for long, short := range pairs {
		long, short = NormalLabel(long), NormalLabel(short)
		swap[long] = short
		swap[short] = long
	}
	words := strings.Split(normal, " ")
	var out []string
	for i, w := range words {
		other, ok := swap[w]
		if !ok {
			continue
		}
		changed := append([]string(nil), words...)
		changed[i] = other
		out = append(out, strings.Join(changed, " "))
	}
	return out
}

func LeadingNumber(s string) (int, bool) {
	s = strings.TrimSpace(s)
	n, digits := 0, 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
		digits++
	}
	return n, digits > 0
}
