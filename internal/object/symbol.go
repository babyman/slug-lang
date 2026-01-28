package object

import (
	"strings"
	"sync"
	"unicode"
)

type Symbol struct {
	Name string
	id   uint64
}

func (s *Symbol) Type() ObjectType { return SYMBOL_OBJ }
func (s *Symbol) Inspect() string {
	if isSymbolIdent(s.Name) {
		return ":" + s.Name
	}
	return `:"` + escapeSymbolLabel(s.Name) + `"`
}
func (s *Symbol) MapKey() MapKey { return MapKey{Type: s.Type(), Value: s.id} }

var (
	symbolMu    sync.Mutex
	symbolTable = map[string]*Symbol{}
	nextSymbol  uint64
)

func InternSymbol(name string) *Symbol {
	symbolMu.Lock()
	defer symbolMu.Unlock()

	if sym, ok := symbolTable[name]; ok {
		return sym
	}
	nextSymbol++
	sym := &Symbol{Name: name, id: nextSymbol}
	symbolTable[name] = sym
	return sym
}

func isSymbolIdent(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !(r == '_' || unicode.IsLetter(r) || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r)) {
				return false
			}
			continue
		}
		if !(r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r)) {
			return false
		}
	}
	return true
}

func escapeSymbolLabel(label string) string {
	var out strings.Builder
	for _, r := range label {
		switch r {
		case '\\':
			out.WriteString(`\\`)
		case '"':
			out.WriteString(`\"`)
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteString(`\t`)
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}
