// Modified from github.com/zdebeer99/goexpression
package filter

import (
	"strings"
	"unicode"
)

const charValidString string = "_"

// isSpace reports whether r is a space character.
func IsSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func IsNumber(r rune) bool {
	return unicode.IsDigit(r) || r == '.'
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func IsAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(charValidString, r)
}

func IsQoute(r rune) bool {
	return strings.ContainsRune("\"'", r)
}

func HasChar(r rune, accept string) bool {
	return strings.ContainsRune(accept, r)
}

func (s *Scanner) Scan(valid func(r rune) bool) bool {
	var isvalid bool
	for valid(s.Next()) {
		isvalid = true
	}
	s.Backup()
	return isvalid
}

// scan upto to the end of a word, returns true if a word was scanned.
// a word must start with a letter or '_' and can contain numbers after the first character.
func (s *Scanner) ScanWord() bool {
	r := s.Next()
	if unicode.IsLetter(r) || strings.ContainsRune(charValidString, r) {
		for {
			r = s.Next()
			if IsAlphaNumeric(r) {
				continue
			} else {
				s.Backup()
				return true
			}
		}
	}
	s.Backup()
	return false
}

func (s *Scanner) ScanNumber() bool {
	state := s.SaveState()
	r := s.Next()
	isdigit := unicode.IsDigit(r)
	if !isdigit && (r == '-' || r == '.') {
		//if the first char is '-' or '.' the next char must be a digit.
		if !unicode.IsDigit(s.Next()) {
			s.LoadState(state)
			return false
		} else {
			isdigit = true
		}
	} else if !isdigit {
		s.Backup()
		return false
	}
	if s.Scan(IsNumber) || isdigit {
		return true
	} else {
		s.LoadState(state)
		return false
	}
}
