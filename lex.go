package jsx

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// item represents a token or text string returned from the scanner.
type item struct {
	typ     itemType // The type of this item.
	pos     int      // The starting position, in bytes, of this item in the input string.
	lastPos int
	val     string // The value of this item.
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case len(i.val) > 10:
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

// itemType identifies the type of lex items.
type itemType int

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.pos, l.input[l.start:l.pos]}
	l.start = l.pos
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) reject(invalid string) bool {
	if strings.IndexRune(invalid, l.next()) >= 0 {
		l.backup()
		return true
	}
	return false
}

func (l *lexer) rejectRun(invalid string) {
	for strings.IndexRune(invalid, l.next()) < 0 {
	}
	l.backup()
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// lineNumber reports which line we're on, based on the position of
// the previous item returned by nextItem. Doing it this way
// means we don't have to worry about peek double counting.
func (l *lexer) lineNumber() int {
	return 1 + strings.Count(l.input[:l.lastPos], "\n")
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, l.pos, fmt.Sprintf(format, args...)}
	return nil
}

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.lastPos
	return item
}

// lex creates a new scanner for the input string.
func lex(input string) *lexer {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
	go l.run()
	return l
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for l.state = lexOpeningTag; l.state != nil; {
		l.state = l.state(l)
	}
}

const eof = -1

// lexer holds the state of the scanner.
type lexer struct {
	input   string  // the string being scanned
	state   stateFn // the next lexing function to enter
	pos     int     // current position in the input
	start   int     // start position of this item
	width   int     // width of last rune read from input
	lastPos int     // position of most recent item returned by nextItem
	inattr  bool
	items   chan item // channel of scanned items
}

const (
	validHtmlNameStartChar = `ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_`
	validHtmlNameChar      = `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-.`
	whiteSpace             = " \t\n\r"
)

const (
	itemError itemType = iota // error occurred; value is text of error
	itemEOF
	itemJS
	itemOpeningTag
	itemEndOpeningTag
	itemClosingTag
	itemSelfClosingTag
	itemAttributeName
	itemAttributeValue
	itemLeftDelim
	itemRightDelim
	itemEllipsis
	itemText
)

func lexOpeningTag(l *lexer) stateFn {
	if !l.accept("<") {
		l.emit(itemError)
		// this could also be interpreted as an EOF for JSX
		return nil
	}
	l.ignore()
	l.acceptRun(validHtmlNameStartChar)
	l.acceptRun(validHtmlNameChar)
	if l.pos > l.start {
		l.emit(itemOpeningTag)
		return lexAttributes
	}
	l.emit(itemError)
	return nil
}

func lexAttributes(l *lexer) stateFn {
	l.inattr = true
	l.acceptRun(whiteSpace)
	l.ignore()
	switch l.next() {
	case '{':
		l.emit(itemLeftDelim)
		return lexSpreadAttribute
	case '>':
		l.emit(itemEndOpeningTag)
		return lexChildren
	case '/':
		if l.peek() != '>' {
			l.emit(itemError)
			return nil
		}
		l.next()
		l.emit(itemSelfClosingTag)
		l.acceptRun(whiteSpace)
		l.ignore()
		return lexOpeningTag
	default:
		l.backup()
	}
	return lexAttributeName
}

func lexAttributeName(l *lexer) stateFn {
	if !l.accept(validHtmlNameStartChar) {
		l.emit(itemError)
		return nil
	}
	l.acceptRun(validHtmlNameChar)
	l.emit(itemAttributeName)
	if !l.accept("=") {
		l.emit(itemError)
		return nil
	}
	l.ignore()
	return lexAttributeValue
}

func lexAttributeValue(l *lexer) stateFn {
	switch l.next() {
	case '"':
		l.ignore()
		l.rejectRun(`"`)
		l.emit(itemAttributeValue)
		if !l.accept(`"`) {
			l.emit(itemError)
			return nil
		}
		l.ignore()
		return lexAttributes
	case '\'':
		l.ignore()
		l.rejectRun(`'`)
		l.emit(itemAttributeValue)
		if !l.accept(`'`) {
			l.emit(itemError)
			return nil
		}
		l.ignore()
		return lexAttributes
	case '{':
		l.emit(itemLeftDelim)
		return lexAssignment
	}
	l.emit(itemError)
	return nil
}

func acceptComments(l *lexer) (success bool) {
	if l.peek() == '/' {
		l.next()
		switch l.next() {
		case '/':
			l.rejectRun("\n\r")
			l.ignore()
			return true
		case '*':
			for {
				r := l.next()
				switch r {
				case eof:
					return false
				case '*':
					if l.peek() == '/' {
						l.next()
						return true
					}
				}
			}
		}
	}
	return true
}

func lexSpreadAttribute(l *lexer) stateFn {
	l.acceptRun(whiteSpace)
	l.ignore()
	if len(l.input[l.pos:]) < 3 {
		l.emit(itemError)
		return nil
	}

	if !acceptComments(l) {
		l.emit(itemError)
		return nil
	}
	if l.pos > l.start {
		l.ignore()
	}

	if l.input[l.pos:l.pos+3] != "..." {
		l.emit(itemError)
		return nil
	}
	l.pos += 3
	l.emit(itemEllipsis)
	return lexAssignment
}

func lexChildren(l *lexer) stateFn {
	l.inattr = false
	for {
		r := l.next()
		switch r {
		case eof:
			l.emit(itemError)
			return nil
		case '<':
			l.backup()
			if l.pos > l.start {
				l.emit(itemText)
			}
			l.next()
			if l.peek() == '/' {
				l.backup()
				return lexClosingTag
			}
			l.backup()
			return lexOpeningTag
		case '{':
			l.backup()
			if l.pos > l.start {
				l.emit(itemText)
			}
			l.next()
			l.emit(itemLeftDelim)
			return lexAssignment
		}
	}
}

func lexClosingTag(l *lexer) stateFn {
	if !l.accept("<") {
		l.emit(itemError)
		return nil
	}
	if !l.accept("/") {
		l.emit(itemError)
		return nil
	}
	l.ignore()
	l.acceptRun(validHtmlNameStartChar)
	l.acceptRun(validHtmlNameChar)
	if l.pos > l.start {
		l.emit(itemClosingTag)
		return lexChildren
	}
	l.emit(itemError)
	return nil
}

func lexAssignment(l *lexer) stateFn {
	var (
		depth          = 0
		inDoubleQuotes = false
		inSingleQuotes = false
	)
	for {
		if !inDoubleQuotes && !inSingleQuotes {
			if !acceptComments(l) {
				l.emit(itemError)
				return nil
			}
		}
		r := l.next()
		switch r {
		case eof:
			l.emit(itemError)
			return nil
		case '\\':
			switch l.peek() {
			case '\'', '"', '\\':
				l.next()
			case eof:
				l.emit(itemError)
				return nil
			}
		case '\'':
			if !inDoubleQuotes {
				inSingleQuotes = true
			}
		case '"':
			if !inSingleQuotes {
				inDoubleQuotes = true
			}
		case '{':
			if !inDoubleQuotes && !inSingleQuotes {
				depth++
			}
		case '}':
			if depth == 0 {
				l.backup()
				l.emit(itemJS)
				l.next()
				l.emit(itemRightDelim)
				if l.inattr {
					return lexAttributes
				} else {
					return lexChildren
				}
			}
			depth--
		}
	}
}
