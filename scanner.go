package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type itemType int

const (
	itemError itemType = iota

	itemEOF

	itemComment
	itemCode
	itemString
)

type item struct {
	typ itemType
	val string
}

var eof rune = 0

func (i item) String() string {
	switch i.typ {
	case itemEOF:
		return "EOF"
	case itemError:
		return i.val
	case itemCode:
		if len(i.val) > 10 {
			return fmt.Sprintf("Code: %.10q...", i.val)
		}
		return fmt.Sprintf("Code: %q", i.val)

	case itemComment:
		if len(i.val) > 10 {
			return fmt.Sprintf("Comment: %.10q...", i.val)
		}
		return fmt.Sprintf("Comment: %q", i.val)

	case itemString:
		if len(i.val) > 10 {
			return fmt.Sprintf("String: %.10q...", i.val)
		}
		return fmt.Sprintf("String: %q", i.val)
	default:
		panic("unknown item type")
	}
}

type lexer struct {
	input string    // the string being scanned.
	start int       // start position of this item.
	pos   int       // current position in the input.
	width int       // width of last rune read from input.
	items chan item // channel of scanned items.
}

func lex(input string) (*lexer, chan item) {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
	go l.run() // Concurrently run state machine.
	return l, l.items
}

// run lexes the input by executing state functions until
// the state is nil.
func (l *lexer) run() {
	for state := lexCode; state != nil; {
		state = state(l)
	}
	close(l.items) // No more tokens will be delivered.
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// peek returns but does not consume
// the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune.
// Can be called only once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// stateFn represents the state of the scanner
// as a function that returns the next state.
type stateFn func(*lexer) stateFn

const shortCommentStart = "//"

func lexShortComment(l *lexer) stateFn {
	l.pos += len(shortCommentStart)
	for {
		r := l.next()
		if r == eof || r == '\n' {
			break
		}
	}
	l.emit(itemComment)
	return lexCode

}

const longCommentStart = "/*"
const longCommentEnd = "*/"

func lexLongComment(l *lexer) stateFn {
	l.pos += len(longCommentStart)
	for {
		if strings.HasPrefix(l.input[l.pos:], longCommentEnd) {
			break
		}
		if l.next() == eof {
			panic("eof while scanning long comment")
		}
	}
	l.pos += len(longCommentEnd)
	l.emit(itemComment)
	return lexCode

}

const stringStart = `"`
const stringEnd = `"`
const stringEscapeRune = '\\'

func lexString(l *lexer) stateFn {
	l.pos += len(stringStart)
	for {
		if strings.HasPrefix(l.input[l.pos:], stringEnd) {
			break
		}

		// TODO: this doesn't really scan escaped chars, just enough to scan a string
		if l.next() == stringEscapeRune {
			l.pos += len(stringEnd)
		}
	}
	l.pos += len(stringEnd)
	l.emit(itemString)
	return lexCode
}

func lexCode(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], longCommentStart) {
			if l.pos > l.start {
				l.emit(itemCode)
			}
			return lexLongComment
		}
		if strings.HasPrefix(l.input[l.pos:], shortCommentStart) {
			if l.pos > l.start {
				l.emit(itemCode)
			}
			return lexShortComment
		}

		if strings.HasPrefix(l.input[l.pos:], stringStart) {
			if l.pos > l.start {
				l.emit(itemCode)
			}
			return lexString
		}

		if l.next() == eof {
			break
		}
	}
	// Correctly reached EOF.
	if l.pos > l.start {
		l.emit(itemCode)
	}
	l.emit(itemEOF) // Useful to make EOF a token.
	return nil      // Stop the run loop.
}
