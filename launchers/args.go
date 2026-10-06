package launchers

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// ParseArgs supports quoted arguments. Backslash escapes quotes, backslash and whitespace;
// other backslashes remain literal (including Windows paths). No shell evaluation occurs.
func ParseArgs(input string) ([]string, error) {
	args := []string{}
	var b strings.Builder
	var quote rune
	started := false
	r := []rune(input)
	for i := 0; i < len(r); i++ {
		c := r[i]
		if c == '\\' && i+1 < len(r) && (r[i+1] == '\\' || r[i+1] == '"' || r[i+1] == '\'' || unicode.IsSpace(r[i+1])) {
			i++
			b.WriteRune(r[i])
			started = true
			continue
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			} else {
				b.WriteRune(c)
			}
			started = true
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			started = true
			continue
		}
		if unicode.IsSpace(c) {
			if started {
				args = append(args, b.String())
				b.Reset()
				started = false
			}
			continue
		}
		b.WriteRune(c)
		started = true
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote in arguments")
	}
	if started {
		args = append(args, b.String())
	}
	return args, nil
}

var placeholder = regexp.MustCompile(`\$[A-Z_]+`)

func expand(template string, values map[string]string) (string, error) {
	var err error
	result := placeholder.ReplaceAllStringFunc(template, func(key string) string {
		v, ok := values[key[1:]]
		if !ok {
			err = fmt.Errorf("unknown placeholder %s", key)
		}
		return v
	})
	return result, err
}
