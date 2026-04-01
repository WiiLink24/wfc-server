// Modified from github.com/zdebeer99/goexpression
package filter

import (
	"fmt"
	"strconv"
	"strings"
)

type Token interface {
	Category() TokenCategory
	SetError(err error)
	Error() error
	String() string
}

type TokenCategory int

const (
	CatOther TokenCategory = iota
	CatFunction
	CatValue
)

type EmptyToken struct {
	tokencat TokenCategory
	err      error
}

func NewEmptyToken() *EmptyToken {
	return &EmptyToken{CatOther, nil}
}

func (this *EmptyToken) Category() TokenCategory {
	return this.tokencat
}

func (this *EmptyToken) Error() error {
	return this.err
}

func (this *EmptyToken) SetError(err error) {
	this.err = err
}

func (this *EmptyToken) String() string {
	return "Base()"
}

type ErrorToken struct {
	EmptyToken
}

func NewErrorToken(err string) *ErrorToken {
	return &ErrorToken{EmptyToken{CatOther, fmt.Errorf(err)}}
}

type NumberToken struct {
	EmptyToken
	Value int64
}

func NewNumberToken(value string) *NumberToken {
	node := &NumberToken{EmptyToken{CatValue, nil}, 0}
	val1, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		panic("number node failed to parse string to number (" + value + ")")
		return node
	}
	node.Value = val1
	return node
}

func (this *NumberToken) String() string {
	return fmt.Sprintf("Number(%v)", this.Value)
}

type IdentityToken struct {
	EmptyToken
	Name string
}

func NewIdentityToken(name string) *IdentityToken {
	return &IdentityToken{EmptyToken{CatValue, nil}, name}
}

func (this *IdentityToken) String() string {
	return fmt.Sprintf("Identity(%s)", this.Name)
}

type FuncToken struct {
	EmptyToken
	Name      string
	Arguments []*TreeNode
}

func NewFuncToken(name string) *FuncToken {
	return &FuncToken{EmptyToken{CatFunction, nil}, name, make([]*TreeNode, 0)}
}

func (this *FuncToken) AddArgument(arg *TreeNode) {
	this.Arguments = append(this.Arguments, arg)
}

func (this *FuncToken) String() string {
	args := make([]string, len(this.Arguments))
	for i, v := range this.Arguments {
		args[i] = fmt.Sprintf("%s", strings.Replace(v.String(), "\n", ",", -1))
	}
	return fmt.Sprintf("Func %s(%s)", this.Name, args)
}

type OperatorToken struct {
	EmptyToken
	Operator string
	lvl      int
}

func NewOperatorToken(operator string) *OperatorToken {
	op := &OperatorToken{EmptyToken{CatFunction, nil}, "", -1}
	op.SetOperator(operator)
	return op
}

func (this *OperatorToken) SetOperator(operator string) {
	this.Operator = operator
	this.lvl = operators.Level(operator)
	if this.lvl < 0 {
		panic(fmt.Sprintf("invalid operator %q", operator))
	}
}

// OperatorPrecedence return true if the operator argument is lower than the current operator.
func (this *OperatorToken) Precedence(operator string) int {
	lvl := operators.Level(operator)
	switch {
	case lvl == this.lvl:
		return 0
	case lvl > this.lvl:
		return 1
	case lvl < this.lvl:
		return -1
	}
	panic("unreachable code")
}

func (this *OperatorToken) String() string {
	return fmt.Sprintf("Func(%s)", this.Operator)
}

type OperatorPrecedence [][]string

func (this OperatorPrecedence) Level(operator string) int {
	for level, operators := range this {
		for _, op := range operators {
			if op == strings.ToLower(operator) {
				return 5 - level
			}
		}
	}
	return -1
}

func (this OperatorPrecedence) All() []string {
	out := make([]string, 0)
	for _, operators := range this {
		for _, op := range operators {
			out = append(out, op)
		}
	}
	return out
}

var operators = OperatorPrecedence{
	{"*", "/", "%"},
	{"+", "-", "&", "^", "|"},
	{"==", "=", "!=", ">=", "<=", ">", "<"},
	{"&&", "and"},
	{"||", "or", "like"},
}

var operatorList = operators.All()

type LRFuncToken struct {
	EmptyToken
	Name string
}

func NewLRFuncToken(name string) *LRFuncToken {
	return &LRFuncToken{EmptyToken{CatFunction, nil}, name}
}

func (this *LRFuncToken) String() string {
	return fmt.Sprintf("Func(%s)", this.Name)
}

type GroupToken struct {
	EmptyToken
	GroupType string
}

func NewGroupToken(group string) *GroupToken {
	return &GroupToken{EmptyToken{CatOther, nil}, group}
}

func (this *GroupToken) String() string {
	return fmt.Sprintf("Group(%s)", this.GroupType)
}

type TextToken struct {
	EmptyToken
	Text string
}

func NewTextToken(text string) *TextToken {
	return &TextToken{EmptyToken{CatValue, nil}, text}
}

func (this *TextToken) String() string {
	return fmt.Sprintf("%q", this.Text)
}
