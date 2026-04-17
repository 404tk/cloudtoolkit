// Package argparse splits payload metadata strings into tokens with shell-style
// quoting so values containing spaces (passwords, etc.) survive intact.
//
// Rules:
//   - whitespace (space/tab) separates tokens
//   - 'single-quoted' preserves every byte literally until the next '
//   - "double-quoted" preserves bytes, but \\ and \" are unescaped
//   - \X outside any quote yields X (including \ to quote a space)
//   - unterminated quotes extend to end of input (lenient, matches user intent)
package argparse

import "strings"

// Split parses s into argv.
func Split(s string) []string {
	var (
		out []string
		cur strings.Builder
		inSingle, inDouble, hasTok bool
	)
	flush := func() {
		if hasTok {
			out = append(out, cur.String())
			cur.Reset()
			hasTok = false
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(c)
			}
		case inDouble:
			if c == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
				cur.WriteByte(s[i+1])
				i++
			} else if c == '"' {
				inDouble = false
			} else {
				cur.WriteByte(c)
			}
		case c == '\'':
			inSingle = true
			hasTok = true
		case c == '"':
			inDouble = true
			hasTok = true
		case c == '\\' && i+1 < len(s):
			cur.WriteByte(s[i+1])
			hasTok = true
			i++
		case c == ' ' || c == '\t':
			flush()
		default:
			cur.WriteByte(c)
			hasTok = true
		}
	}
	flush()
	return out
}

// SplitN behaves like Split but stops after the first n-1 tokens, returning
// the remainder as the last element verbatim (no further quote handling).
// Useful when the last argument is an opaque blob (e.g. base64-encoded command).
func SplitN(s string, n int) []string {
	if n <= 0 {
		return Split(s)
	}
	var (
		out []string
		cur strings.Builder
		inSingle, inDouble, hasTok bool
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(c)
			}
		case inDouble:
			if c == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
				cur.WriteByte(s[i+1])
				i++
			} else if c == '"' {
				inDouble = false
			} else {
				cur.WriteByte(c)
			}
		case c == '\'':
			inSingle = true
			hasTok = true
		case c == '"':
			inDouble = true
			hasTok = true
		case c == '\\' && i+1 < len(s):
			cur.WriteByte(s[i+1])
			hasTok = true
			i++
		case c == ' ' || c == '\t':
			if !hasTok {
				continue
			}
			out = append(out, cur.String())
			cur.Reset()
			hasTok = false
			if len(out) == n-1 {
				rest := strings.TrimLeft(s[i+1:], " \t")
				if rest != "" {
					out = append(out, rest)
				}
				return out
			}
		default:
			cur.WriteByte(c)
			hasTok = true
		}
	}
	if hasTok {
		out = append(out, cur.String())
	}
	return out
}
