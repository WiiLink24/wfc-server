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

func (s *Scanner) StartPosition() int {
	return int(s.start)
}

func (s *Scanner) SetPosition(pos int) {
	s.pos = Pos(pos)
}

func (s *Scanner) SetStartPosition(pos int) {
	s.start = Pos(pos)
}

// Token return the current selected text and move the start position to the current position
func (s *Scanner) Commit() string {
	r1 := s.input[s.start:s.pos]
	s.start = s.pos
	s.prevState = s.SaveState()
	return r1
}

// IsEOF check if the end of the current string has been reached.
func (s *Scanner) IsEOF() bool {
	return int(s.pos) >= len(s.input)
}

func (s *Scanner) Size() int {
	return len(s.input)
}

func (s *Scanner) MoveStart(pos int) {
	s.start = s.start + Pos(pos)
}

// Next returns the next rune in the input.
func (s *Scanner) Next() rune {
	s.safebackup = true
	if s.IsEOF() {
		s.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(s.input[s.pos:])
	s.width = Pos(w)
	s.pos += s.width
	s.curr = r
	return r
}

func (s *Scanner) Skip() {
	s.Next()
	s.Commit()
}

// Peek returns but does not consume the next rune in the input.
func (s *Scanner) Peek() rune {
	r := s.Next()
	s.Backup()
	return r
}

// Backup steps back one rune. Can only be called once per call of next.
func (s *Scanner) Backup() {
	s.pos -= s.width
}

// Rollback move the curr pos back to the start pos.
func (s *Scanner) Rollback() {
	s.LoadState(s.prevState)
}

// Ignore skips over the pending input before this point.
func (s *Scanner) Ignore() {
	s.start = s.pos
}

// accept consumes the next rune if it's from the valid set.
func (s *Scanner) Accept(valid string) bool {
	if strings.ContainsRune(valid, s.Next()) {
		return true
	}
	s.Backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (s *Scanner) AcceptRun(valid string) (found int) {
	for strings.ContainsRune(valid, s.Next()) {
		found++
	}
	s.Backup()
	return found
}

// runTo consumes a run of runes until an item in the valid set is found.
func (s *Scanner) RunTo(valid string) rune {
	for {
		r := s.Next()
		if r == eof {
			return r
		}
		if strings.ContainsRune(valid, r) {
			return r
		}
	}
}

func (s *Scanner) Prefix(pre string) bool {
	if strings.HasPrefix(s.input[s.pos:], pre) {
		s.pos += Pos(len(pre))
		return true
	}
	return false
}

func (s *Scanner) SkipSpaces() {
	for IsSpace(s.Next()) {
	}
	s.Backup()
	s.Ignore()
}

func (s *Scanner) SkipToNewLine() {
	for {
		r := s.Next()
		if s.IsEOF() {
			break
		}
		if r == '\n' {
			break
		}
	}
	s.Ignore()
}

// lineNumber reports which line we're on, based on the position of
// the previous item returned by nextItem. Doing it this way
// means we don't have to worry about peek double counting.
func (s *Scanner) LineNumber() int {
	return 1 + strings.Count(s.input[:s.pos], "\n")
}

type ScannerState struct {
	start Pos
	pos   Pos
	width Pos
}

func (s *Scanner) SaveState() ScannerState {
	return ScannerState{start: s.start, pos: s.pos, width: s.width}
}

func (s *Scanner) LoadState(state ScannerState) {
	s.start, s.pos, s.width = state.start, state.pos, state.width
}
