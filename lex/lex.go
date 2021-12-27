package lex

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type Pos struct {
	Line int
	Char int
}

func (p Pos) String() string {
	return fmt.Sprintf("%d:%d", p.Line+1, p.Char+1)
}

type Token struct {
	Pos Pos
	Typ ItemType // The type of this item.
	Val string   // The value of this item.
}

type ItemType int

const (
	ItemError      ItemType = iota
	ItemBeginBlock          // starts a block
	ItemEndBlock            // ends a block
	ItemWord
	ItemString
	ItemSemicolon
	ItemAssign
	ItemSet
	ItemLease
)

type Lexer struct {
	Input  *bufio.Reader
	Tokens chan Token
	Pos    Pos
}

func (l *Lexer) Next() (rune, error) {
	r, _, err := l.Input.ReadRune()
	if err != nil {
		return r, err
	}
	if r == '\n' {
		l.Pos.Char = 0
		l.Pos.Line++
	} else {
		l.Pos.Char++
	}
	return r, nil
}

func (l *Lexer) Emit(typ ItemType, val string) {
	l.Tokens <- Token{Typ: typ, Val: val, Pos: l.Pos}
}

func Lex(input io.Reader) chan Token {
	l := &Lexer{
		Input:  bufio.NewReader(input),
		Tokens: make(chan Token),
	}
	go func() {
		if err := l.lex(); err != nil {
			l.Emit(ItemError, err.Error())
		}
	}()
	return l.Tokens
}

func (l *Lexer) lex() error {
	for {
		r, err := l.Next()
		if err != nil {
			if err == io.EOF {
				close(l.Tokens)
				return nil
			}
			return fmt.Errorf("lexing top-level: %w", err)
		}
		switch {
		case r == '#':
			if err := l.lexComment(); err != nil {
				return err
			}
		case unicode.IsSpace(r):
		case r == '"':
			if err := l.lexString(); err != nil {
				return err
			}
		case r == ';':
			l.Emit(ItemSemicolon, ";")
		case r == '{':
			l.Emit(ItemBeginBlock, "{")
		case r == '}':
			l.Emit(ItemEndBlock, "}")
		case r == '=':
			l.Emit(ItemAssign, "=")
		default:
			if err := l.lexWord(r); err != nil {
				return err
			}
		}
	}
}

func (l *Lexer) lexComment() error {
	for {
		r, err := l.Next()
		if err != nil {
			return fmt.Errorf("lexing comment: %w", err)
		}
		switch r {
		case '\n':
			return nil
		}
	}
}

func (l *Lexer) lexString() error {
	var token strings.Builder
	token.WriteRune('"')
	for {
		r, err := l.Next()
		if err != nil {
			return fmt.Errorf("lexing string: %w", err)
		}
		token.WriteRune(r)
		switch r {
		case '"':
			l.Emit(ItemString, token.String())
			return nil
		}
	}
}

func (l *Lexer) lexWord(r rune) error {
	var token strings.Builder
	token.WriteRune(r)

	emit := func() {
		s := token.String()
		if s == "set" {
			l.Emit(ItemSet, token.String())
		} else if s == "lease" || s == "ia-na" || s == "ia-ta" || s == "ia-pd" {
			l.Emit(ItemLease, token.String())
		} else {
			l.Emit(ItemWord, token.String())
		}
	}

	for {
		r, err := l.Next()
		if err != nil {
			return fmt.Errorf("lexing word: %w", err)
		}
		switch {
		case unicode.IsSpace(r):
			emit()
			return nil
		case r == ';':
			emit()
			l.Emit(ItemSemicolon, ";")
			return nil
		case r == '{':
			emit()
			l.Emit(ItemBeginBlock, "{")
			return nil
		case r == '=':
			emit()
			l.Emit(ItemAssign, "=")
			return nil
		}
		token.WriteRune(r)
	}
}
