package oui

import (
	_ "embed"
	"strings"
)

//go:embed oui.csv
var rawCSV string

type Entry struct {
	Vendor   string
	TypeHint string
}

type Lookup struct {
	data map[string]Entry
}

func NewLookup() *Lookup {
	l := &Lookup{data: make(map[string]Entry)}
	for _, line := range strings.Split(rawCSV, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		prefix := strings.ToUpper(strings.ReplaceAll(parts[0], ":", ""))
		l.data[prefix] = Entry{Vendor: strings.TrimSpace(parts[1]), TypeHint: strings.TrimSpace(parts[2])}
	}
	return l
}

func (l *Lookup) Lookup(mac string) (Entry, bool) {
	if l == nil {
		return Entry{}, false
	}
	mac = strings.ToUpper(strings.ReplaceAll(mac, ":", ""))
	if len(mac) < 6 {
		return Entry{}, false
	}
	if entry, ok := l.data[mac[:6]]; ok {
		return entry, true
	}
	return Entry{}, false
}
