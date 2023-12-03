// Modified from github.com/zdebeer99/goexpression
package filter

import (
	"errors"
	"fmt"
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

func (this *parser) getCurr() Token {
	if this.curr != nil {
		return this.curr.Value
	}
	return nil
}

func (this *parser) parse() {
	this.pumpExpression()
}

func (this *parser) add(token Token) *TreeNode {
	return this.curr.Add(token)
}

func (this *parser) push(token Token) *TreeNode {
	return this.curr.Push(token)
}

func (this *parser) lastNode() *TreeNode {
	return this.curr.LastElement()
}

func (this *parser) parentNode() *TreeNode {
	return this.curr.Parent()
}

func (this *parser) commit() string {
	return this.scan.Commit()
}

// parseOpenBracket
func (this *parser) parseOpenBracket() bool {
	this.curr = this.add(NewGroupToken("()"))
	this.commit()
	return true
}

// parseCloseBracket
func (this *parser) parseCloseBracket() stateFn {
	for {
		v1, ok := this.curr.Value.(*GroupToken)
		if ok && v1.GroupType == "()" {
			this.commit()
			this.curr = this.curr.Parent()
			return branchExpressionOperatorPart
		}
		if ok && v1.GroupType == "" {
			//must be a bracket part of a parent loop, exit this sub loop.
			this.scan.Backup()
			return nil
		}
		if this.curr.Parent() == nil {
			panic("brackets not closed")
		}
		this.curr = this.curr.Parent()
	}
	panic("unreachable code")
}

func (this *parser) AcceptOperator() bool {
	scan := this.scan
	for _, op := range operatorList {
		if scan.Prefix(op) {
			return true
		}
	}
	return false
}

// parseOperator
func (this *parser) parseOperator() bool {
	operator := this.commit()
	lastnode := this.lastNode()
	onode, ok := this.getCurr().(*OperatorToken)
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
				this.curr = lastnode.Push(NewOperatorToken(operator))
				return true
			}
		}
		//after */ presedence fallback and continue pushing +- operators from the bottom.
		if onode.Precedence(operator) < 0 {
			for {
				v1, ok := this.curr.Parent().Value.(*OperatorToken)
				//if ok && strings.Index("+-", v1.Name) >= 0 {
				if ok && operators.Level(v1.Operator) >= 0 {
					this.curr = this.curr.Parent()
				} else {
					break
				}
			}
		}
		//standard operator push
		this.curr = this.push(NewOperatorToken(operator))
		return true
	}
	//set previous found value as argument of the operator
	if lastnode != nil {
		this.curr = lastnode.Push(NewOperatorToken(operator))
	} else {
		this.state = nil
		panic(fmt.Sprintf("expecting a value before operator %q", operator))
	}
	return true
}

// parseLRFunc
func (this *parser) parseLRFunc() bool {
	lrfunc := this.commit()
	lastnode := this.lastNode()
	if lastnode != nil {
		this.curr = lastnode.Push(NewLRFuncToken(lrfunc))
	} else {
		this.state = nil
		panic(fmt.Sprintf("expecting a value before operator %q", lrfunc))
	}
	return false
}

func (this *parser) ParseText() string {
	scan := this.scan
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
