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

func (t *EmptyToken) Category() TokenCategory {
	return t.tokencat
}

func (t *EmptyToken) Error() error {
	return t.err
}

func (t *EmptyToken) SetError(err error) {
	t.err = err
}

func (t *EmptyToken) String() string {
	return "Base()"
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
	}
	node.Value = val1
	return node
}

func (t *NumberToken) String() string {
	return fmt.Sprintf("Number(%v)", t.Value)
}

type IdentityToken struct {
	EmptyToken
	Name string
}

func NewIdentityToken(name string) *IdentityToken {
	return &IdentityToken{EmptyToken{CatValue, nil}, name}
}

func (t *IdentityToken) String() string {
	return fmt.Sprintf("Identity(%s)", t.Name)
}

type FuncToken struct {
	EmptyToken
	Name      string
	Arguments []*TreeNode
}

func NewFuncToken(name string) *FuncToken {
	return &FuncToken{EmptyToken{CatFunction, nil}, name, make([]*TreeNode, 0)}
}

func (t *FuncToken) AddArgument(arg *TreeNode) {
	t.Arguments = append(t.Arguments, arg)
}

func (t *FuncToken) String() string {
	args := make([]string, len(t.Arguments))
	for i, v := range t.Arguments {
		args[i] = strings.ReplaceAll(v.String(), "\n", ",")
	}
	return fmt.Sprintf("Func %s(%s)", t.Name, args)
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

func (t *OperatorToken) SetOperator(operator string) {
	t.Operator = operator
	t.lvl = operators.Level(operator)
	if t.lvl < 0 {
		panic(fmt.Sprintf("invalid operator %q", operator))
	}
}

// OperatorPrecedence return true if the operator argument is lower than the current operator.
func (t *OperatorToken) Precedence(operator string) int {
	lvl := operators.Level(operator)
	switch {
	case lvl == t.lvl:
		return 0
	case lvl > t.lvl:
		return 1
	case lvl < t.lvl:
		return -1
	}
	panic("unreachable code")
}

func (t *OperatorToken) String() string {
	return fmt.Sprintf("Func(%s)", t.Operator)
}

type OperatorPrecedence [][]string

func (op OperatorPrecedence) Level(operator string) int {
	for level, operators := range op {
		for _, op := range operators {
			if op == strings.ToLower(operator) {
				return 5 - level
			}
		}
	}
	return -1
}

func (op OperatorPrecedence) All() []string {
	out := make([]string, 0)
	for _, operators := range op {
		out = append(out, operators...)
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

func (t *LRFuncToken) String() string {
	return fmt.Sprintf("Func(%s)", t.Name)
}

type GroupToken struct {
	EmptyToken
	GroupType string
}

func NewGroupToken(group string) *GroupToken {
	return &GroupToken{EmptyToken{CatOther, nil}, group}
}

func (t *GroupToken) String() string {
	return fmt.Sprintf("Group(%s)", t.GroupType)
}

type TextToken struct {
	EmptyToken
	Text string
}

func NewTextToken(text string) *TextToken {
	return &TextToken{EmptyToken{CatValue, nil}, text}
}

func (t *TextToken) String() string {
	return fmt.Sprintf("%q", t.Text)
}
