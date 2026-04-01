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
	return unicode.IsLetter(r) || unicode.IsDigit(r) || strings.IndexRune(charValidString, r) >= 0
}

func IsQoute(r rune) bool {
	return strings.IndexRune("\"'", r) >= 0
}

func HasChar(r rune, accept string) bool {
	return strings.IndexRune(accept, r) >= 0
}

func (this *Scanner) Scan(valid func(r rune) bool) bool {
	var isvalid bool
	for valid(this.Next()) {
		isvalid = true
	}
	this.Backup()
	return isvalid
}

//scan upto to the end of a word, returns true if a word was scanned.
//a word must start with a letter or '_' and can contain numbers after the first character.
func (this *Scanner) ScanWord() bool {
	r := this.Next()
	if unicode.IsLetter(r) || strings.IndexRune(charValidString, r) >= 0 {
		for {
			r = this.Next()
			if IsAlphaNumeric(r) {
				continue
			} else {
				this.Backup()
				return true
			}
		}
	}
	this.Backup()
	return false
}

func (this *Scanner) ScanNumber() bool {
	state := this.SaveState()
	r := this.Next()
	isdigit := unicode.IsDigit(r)
	if !isdigit && (r == '-' || r == '.') {
		//if the first char is '-' or '.' the next char must be a digit.
		if !unicode.IsDigit(this.Next()) {
			this.LoadState(state)
			return false
		} else {
			isdigit = true
		}
	} else if !isdigit {
		this.Backup()
		return false
	}
	if this.Scan(IsNumber) || isdigit {
		return true
	} else {
		this.LoadState(state)
		return false
	}
}
