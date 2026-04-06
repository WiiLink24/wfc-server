// Modified from github.com/zdebeer99/goexpression
package filter

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type stateFn func(*parser) stateFn

type parser struct {
	scan  *Scanner
	root  *TreeNode
	curr  *TreeNode
	err   error
	state stateFn
}

func Parse(input string) (root *TreeNode, err error) {
	defer func() {
		if str := recover(); str != nil {
			root = nil
			err = errors.New(str.(string))
		}
	}()

	root = NewTreeNode(NewEmptyToken())
	parse := &parser{NewScanner(input), root, root, nil, nil}
	parse.parse()
	err = parse.err
	return root, err
}

func (p *parser) getCurr() Token {
	if p.curr != nil {
		return p.curr.Value
	}
	return nil
}

func (p *parser) parse() {
	p.pumpExpression()
}

func (p *parser) add(token Token) *TreeNode {
	return p.curr.Add(token)
}

func (p *parser) push(token Token) *TreeNode {
	return p.curr.Push(token)
}

func (p *parser) lastNode() *TreeNode {
	return p.curr.LastElement()
}

func (p *parser) commit() string {
	return p.scan.Commit()
}

// parseOpenBracket
func (p *parser) parseOpenBracket() bool {
	p.curr = p.add(NewGroupToken("()"))
	p.commit()
	return true
}

// parseCloseBracket
func (p *parser) parseCloseBracket() stateFn {
	for {
		v1, ok := p.curr.Value.(*GroupToken)
		if ok && v1.GroupType == "()" {
			p.commit()
			p.curr = p.curr.Parent()
			return branchExpressionOperatorPart
		}
		if ok && v1.GroupType == "" {
			//must be a bracket part of a parent loop, exit this sub loop.
			p.scan.Backup()
			return nil
		}
		if p.curr.Parent() == nil {
			panic("brackets not closed")
		}
		p.curr = p.curr.Parent()
	}
}

func (p *parser) AcceptOperator() bool {
	scan := p.scan
	state := scan.SaveState()
	for _, op := range operatorList {
		if scan.Prefix(op) || scan.Prefix(strings.ToUpper(op)) {
			p := scan.Peek()
			if unicode.IsLetter(rune(op[0])) && (unicode.IsLetter(p) || unicode.IsNumber(p) || strings.ContainsRune(charValidString, p)) {
				// this is a prefix of a longer word
				scan.LoadState(state)
				continue
			}
			return true
		}
	}
	return false
}

// parseOperator
func (p *parser) parseOperator() bool {
	operator := p.commit()
	lastnode := p.lastNode()
	onode, ok := p.getCurr().(*OperatorToken)
	//push excisting operator up in tree structure
	if ok {
		//operator is the same current operator ignore
		if onode.Operator == operator {
			return true
		}
		//change order for */ presedence
		//fmt.Println(onode, operator, onode.Precedence(operator))
		if onode.Precedence(operator) > 0 {
			if lastnode != nil {
				p.curr = lastnode.Push(NewOperatorToken(operator))
				return true
			}
		}
		//after */ presedence fallback and continue pushing +- operators from the bottom.
		if onode.Precedence(operator) < 0 {
			for {
				v1, ok := p.curr.Parent().Value.(*OperatorToken)
				//if ok && strings.Index("+-", v1.Name) >= 0 {
				if ok && operators.Level(v1.Operator) >= 0 {
					p.curr = p.curr.Parent()
				} else {
					break
				}
			}
		}
		//standard operator push
		p.curr = p.push(NewOperatorToken(operator))
		return true
	}
	//set previous found value as argument of the operator
	if lastnode != nil {
		p.curr = lastnode.Push(NewOperatorToken(operator))
	} else {
		p.state = nil
		panic(fmt.Sprintf("expecting a value before operator %q", operator))
	}
	return true
}

// parseLRFunc
func (p *parser) parseLRFunc() bool {
	lrfunc := p.commit()
	lastnode := p.lastNode()
	if lastnode != nil {
		p.curr = lastnode.Push(NewLRFuncToken(lrfunc))
	} else {
		p.state = nil
		panic(fmt.Sprintf("expecting a value before operator %q", lrfunc))
	}
	return false
}

func (p *parser) ParseText() string {
	scan := p.scan
	r := scan.Next()
	if r == '"' || r == '\'' {
		scan.Ignore()
		endqoute := r
		for {
			r = scan.Next()
			if r == endqoute {
				scan.Backup()
				txt := scan.Commit()
				scan.Next()
				scan.Ignore()
				return txt
			}
			if scan.IsEOF() {
				panic("missing quote and end of text")
			}
		}
	}
	return ""
}
