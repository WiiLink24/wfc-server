// Modified from github.com/zdebeer99/goexpression
package filter

import (
	"strings"
	"unicode/utf8"
)

// Pos represents a byte position in the original input text from which
// this template was parsed.
type Pos int

func (p Pos) Position() Pos {
	return p
}

const eof = -1

// Scanner, Iterates through a string.
type Scanner struct {
	input      string
	start      Pos
	pos        Pos
	width      Pos
	curr       rune
	prevState  ScannerState
	safebackup bool //insure backup is called only once after next.
}

// NewScanner Creates a New Scanner pointer.
func NewScanner(template string) *Scanner {
	return &Scanner{input: template}
}

func (this *Scanner) StartPosition() int {
	return int(this.start)
}

func (this *Scanner) SetPosition(pos int) {
	this.pos = Pos(pos)
}

func (this *Scanner) SetStartPosition(pos int) {
	this.start = Pos(pos)
}

// Token return the current selected text and move the start position to the current position
func (this *Scanner) Commit() string {
	r1 := this.input[this.start:this.pos]
	this.start = this.pos
	this.prevState = this.SaveState()
	return r1
}

// IsEOF check if the end of the current string has been reached.
func (this *Scanner) IsEOF() bool {
	return int(this.pos) >= len(this.input)
}

func (this *Scanner) Size() int {
	return len(this.input)
}

func (this *Scanner) MoveStart(pos int) {
	this.start = this.start + Pos(pos)
}

// Next returns the next rune in the input.
func (this *Scanner) Next() rune {
	this.safebackup = true
	if this.IsEOF() {
		this.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(this.input[this.pos:])
	this.width = Pos(w)
	this.pos += this.width
	this.curr = r
	return r
}

func (this *Scanner) Skip() {
	this.Next()
	this.Commit()
}

// Peek returns but does not consume the next rune in the input.
func (this *Scanner) Peek() rune {
	r := this.Next()
	this.Backup()
	return r
}

// Backup steps back one rune. Can only be called once per call of next.
func (this *Scanner) Backup() {
	this.pos -= this.width
}

// Rollback move the curr pos back to the start pos.
func (this *Scanner) Rollback() {
	this.LoadState(this.prevState)
}

// Ignore skips over the pending input before this point.
func (this *Scanner) Ignore() {
	this.start = this.pos
}

// accept consumes the next rune if it's from the valid set.
func (this *Scanner) Accept(valid string) bool {
	if strings.IndexRune(valid, this.Next()) >= 0 {
		return true
	}
	this.Backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (this *Scanner) AcceptRun(valid string) (found int) {
	for strings.IndexRune(valid, this.Next()) >= 0 {
		found++
	}
	this.Backup()
	return found
}

// runTo consumes a run of runes until an item in the valid set is found.
func (this *Scanner) RunTo(valid string) rune {
	for {
		r := this.Next()
		if r == eof {
			return r
		}
		if strings.IndexRune(valid, r) >= 0 {
			return r
		}
	}
	this.Backup()
	return eof
}

func (this *Scanner) Prefix(pre string) bool {
	if strings.HasPrefix(this.input[this.pos:], pre) {
		this.pos += Pos(len(pre))
		return true
	}
	return false
}

func (this *Scanner) SkipSpaces() {
	for IsSpace(this.Next()) {
	}
	this.Backup()
	this.Ignore()
}

func (this *Scanner) SkipToNewLine() {
	for {
		r := this.Next()
		if this.IsEOF() {
			break
		}
		if r == '\n' {
			break
		}
	}
	this.Ignore()
	return
}

// lineNumber reports which line we're on, based on the position of
// the previous item returned by nextItem. Doing it this way
// means we don't have to worry about peek double counting.
func (this *Scanner) LineNumber() int {
	return 1 + strings.Count(this.input[:this.pos], "\n")
}

type ScannerState struct {
	start Pos
	pos   Pos
	width Pos
}

func (this *Scanner) SaveState() ScannerState {
	return ScannerState{start: this.start, pos: this.pos, width: this.width}
}

func (this *Scanner) LoadState(state ScannerState) {
	this.start, this.pos, this.width = state.start, state.pos, state.width
}
