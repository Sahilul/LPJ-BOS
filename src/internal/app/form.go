package app

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// Form adalah isian POST bersarang ala $_POST PHP: nilai berupa string atau Form.
// items[0][nama]=x menjadi Form{"items": Form{"0": Form{"nama": "x"}}}.
type Form map[string]any

func parseBracketKey(k string) []string {
	i := strings.IndexByte(k, '[')
	if i <= 0 || !strings.HasSuffix(k, "]") {
		return []string{k}
	}
	parts := []string{k[:i]}
	rest := k[i:]
	for len(rest) > 0 {
		if rest[0] != '[' {
			return []string{k}
		}
		j := strings.IndexByte(rest, ']')
		if j < 0 {
			return []string{k}
		}
		parts = append(parts, rest[1:j])
		rest = rest[j+1:]
	}
	return parts
}

func ParseForm(vals url.Values) Form {
	f := Form{}
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts := parseBracketKey(k)
		for _, v := range vals[k] {
			f.set(parts, v)
		}
	}
	return f
}

func (f Form) set(parts []string, v string) {
	cur := f
	for i, p := range parts {
		last := i == len(parts)-1
		if p == "" {
			p = strconv.Itoa(nextIndex(cur))
		}
		if last {
			cur[p] = v
			return
		}
		nx, ok := cur[p].(Form)
		if !ok {
			nx = Form{}
			cur[p] = nx
		}
		cur = nx
	}
}

func nextIndex(f Form) int {
	n := 0
	for k := range f {
		if i, err := strconv.Atoi(k); err == nil && i >= n {
			n = i + 1
		}
	}
	return n
}

// Str: nilai string (trim tidak dilakukan). Bukan string → "".
func (f Form) Str(k string) string {
	if f == nil {
		return ""
	}
	s, _ := f[k].(string)
	return s
}

func (f Form) Trim(k string) string { return strings.TrimSpace(f.Str(k)) }

func (f Form) Has(k string) bool {
	if f == nil {
		return false
	}
	_, ok := f[k]
	return ok
}

// Truthy meniru !empty($x) PHP untuk string.
func (f Form) Truthy(k string) bool {
	v, ok := f[k]
	if !ok {
		return false
	}
	switch x := v.(type) {
	case string:
		return x != "" && x != "0"
	case Form:
		return len(x) > 0
	}
	return false
}

func (f Form) Int(k string) int64 { return toInt(f.Str(k)) }

func (f Form) Sub(k string) Form {
	if f == nil {
		return nil
	}
	switch x := f[k].(type) {
	case Form:
		return x
	case map[string]any:
		return Form(x)
	}
	return nil
}

// SortedKeys: kunci angka diurutkan secara numerik, sisanya leksikal.
func (f Form) SortedKeys() []string {
	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(a, b int) bool {
		ia, ea := strconv.Atoi(keys[a])
		ib, eb := strconv.Atoi(keys[b])
		if ea == nil && eb == nil {
			return ia < ib
		}
		if (ea == nil) != (eb == nil) {
			return ea == nil
		}
		return keys[a] < keys[b]
	})
	return keys
}

// List: elemen array (mis. items[]) berupa Form, terurut.
func (f Form) List(k string) []Form {
	sub := f.Sub(k)
	out := []Form{}
	for _, key := range sub.SortedKeys() {
		if x, ok := sub[key].(Form); ok {
			out = append(out, x)
		} else if x, ok := sub[key].(map[string]any); ok {
			out = append(out, Form(x))
		}
	}
	return out
}

// Strings: elemen array string (mis. jadwal[]), terurut.
func (f Form) Strings(k string) []string {
	out := []string{}
	if s, ok := f[k].(string); ok {
		return append(out, s)
	}
	sub := f.Sub(k)
	for _, key := range sub.SortedKeys() {
		if s, ok := sub[key].(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// normalizeForm memulihkan Form dari JSON sesi (map[string]any).
func normalizeForm(m map[string]any) Form {
	if m == nil {
		return nil
	}
	f := Form{}
	for k, v := range m {
		switch x := v.(type) {
		case map[string]any:
			f[k] = normalizeForm(x)
		case Form:
			f[k] = normalizeForm(x)
		case string:
			f[k] = x
		case nil:
		default:
			f[k] = toStr(x)
		}
	}
	return f
}
