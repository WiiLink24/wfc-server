// Modified from github.com/zdebeer99/goexpression
package filter

import (
	"errors"
	"strconv"
	"strings"
)

type expression struct {
	ast       *TreeNode
	context   map[string]string
	queryGame string
}

// Bug(zdebeer): functions is eval from right to left instead from left to right.
func Eval(basenode *TreeNode, context map[string]string, queryGame string) (value int64, err error) {
	defer func() {
		if str := recover(); str != nil {
			value = 0
			err = errors.New(str.(string))
		}
	}()

	this := &expression{basenode, context, queryGame}
	return this.eval(basenode), nil
}

func (this *expression) eval(basenode *TreeNode) int64 {
	for _, node := range basenode.items {
		switch node.Value.Category() {
		case CatFunction:
			return this.switchFunction(node)

		case CatValue:
			return this.getNumber(node)

		case CatOther:
			this.switchOther(node)
		}
	}
	panic("eval failed")
}

func (this *expression) switchOther(node *TreeNode) {
	switch v1 := node.Value.(type) {
	case *GroupToken:
		if v1.GroupType == "()" {
			this.eval(node)
			return
		}
	}
	panic("invalid node " + node.String())
}

func (this *expression) switchFunction(node *TreeNode) int64 {
	val1 := node.Value.(*OperatorToken)
	switch val1.Operator {
	case "=":
		return this.evalEquals(node.Items())
	case "==":
		return this.evalEquals(node.Items())
	case "!=":
		return this.evalNotEquals(node.Items())

	case ">":
		return this.evalMathOperator(this.evalMathGreater, node.Items())
	case "<":
		return this.evalMathOperator(this.evalMathLess, node.Items())
	case ">=":
		return this.evalMathOperator(this.evalMathGreaterOrEqual, node.Items())
	case "<=":
		return this.evalMathOperator(this.evalMathLessOrEqual, node.Items())
	case "+":
		return this.evalMathOperator(this.evalMathPlus, node.Items())
	case "-":
		return this.evalMathOperator(this.evalMathMinus, node.Items())

	case "and":
		return this.evalAnd(node.Items())
	case "or":
		return this.evalOr(node.Items())
	case "&&":
		return this.evalAnd(node.Items())
	case "||":
		return this.evalOr(node.Items())

	default:
		panic("function not supported: " + val1.Operator)
	}

}

func (this *expression) getString(node *TreeNode) string {
	switch v := node.Value.(type) {
	case *NumberToken:
		return strconv.FormatInt(v.Value, 10)
	case *IdentityToken:
		return this.getValue(v)
	case *OperatorToken:
		return strconv.FormatInt(this.switchFunction(node), 10)
	case *GroupToken:
		if v.GroupType == "()" {
			return strconv.FormatInt(this.eval(node), 10)
		}
		panic("unexpected grouping type: " + node.String())
	case *TextToken:
		return node.Value.(*TextToken).Text

	default:
		panic("unexpected value: " + node.String())
	}
}

func (this *expression) evalEquals(args []*TreeNode) int64 {
	cnt := len(args)
	switch {
	case cnt < 2:
		panic("operator missing arguments")
	case cnt == 2:
		if n, ok := args[0].Value.(*IdentityToken); ok {
			if n.Name == "rk" && this.queryGame == "mariokartwii" {
				return this.evalEqualsRK(this.getString(args[1]))
			}
		}

		if this.getString(args[0]) == this.getString(args[1]) {
			return 1
		}
		return 0
	default:
		arg := this.getString(args[0])
		for i := 1; i < cnt; i++ {
			if arg != this.getString(args[i]) {
				return 0
			}
		}
		return 1
	}
}

// Operator override
func (this *expression) evalEqualsRK(value string) int64 {
	rk := this.context["rk"]
	// Check and remove regional searches due to the limited player count
	// China (ID 6) gets a pass because it was never released
	if len(rk) == 4 && (strings.HasPrefix(rk, "vs_") || strings.HasPrefix(rk, "bt_")) && rk[3] >= '0' && rk[3] < '6' {
		rk = rk[:2]
	}

	if len(value) == 4 && (strings.HasPrefix(value, "vs_") || strings.HasPrefix(value, "bt_")) && value[3] >= '0' && value[3] < '6' {
		value = value[:2]
	}

	if rk == value {
		return 1
	}
	return 0
}

func (this *expression) evalNotEquals(args []*TreeNode) int64 {
	cnt := len(args)
	switch {
	case cnt < 2:
		panic("operator missing arguments")
	case cnt == 2:
		if this.getString(args[0]) != this.getString(args[1]) {
			return 1
		}
		return 0
	default:
		arg := this.getString(args[0])
		for i := 1; i < cnt; i++ {
			if arg == this.getString(args[i]) {
				return 0
			}
		}
		return 1
	}
}

func (this *expression) evalAnd(args []*TreeNode) int64 {
	cnt := len(args)
	if cnt < 2 {
		panic("operator missing arguments")
	}

	for i := 0; i < cnt; i++ {
		if this.getString(args[i]) == "0" {
			return 0
		}
	}
	return 1
}

func (this *expression) evalOr(args []*TreeNode) int64 {
	cnt := len(args)
	if cnt < 2 {
		panic("operator missing arguments")
	}

	for i := 0; i < cnt; i++ {
		if this.getString(args[i]) != "0" {
			return 1
		}
	}
	return 0
}

func (this *expression) getNumber(node *TreeNode) int64 {
	switch v := node.Value.(type) {
	case *NumberToken:
		return v.Value
	case *IdentityToken:
		r1 := this.getValue(v)
		return this.toInt64(r1)
	case *OperatorToken:
		return this.switchFunction(node)
	case *GroupToken:
		if v.GroupType == "()" {
			return this.eval(node)
		}
		panic("unexpected grouping type: " + node.String())
	case *TextToken:
		return this.toInt64(node.Value.(*TextToken).Text)

	default:
		panic("unexpected value: " + node.String())
	}
}

func (this *expression) evalMathOperator(fn func(int64, int64) int64, args []*TreeNode) int64 {
	cnt := len(args)
	switch {
	case cnt < 2:
		panic("operator missing arguments")
	case cnt == 2:
		if n, ok := args[0].Value.(*IdentityToken); ok {
			// Remove VR search due to the limited player count
			if (n.Name == "ev" || n.Name == "eb") && this.queryGame == "mariokartwii" {
				return 1
			}
		}

		return fn(this.getNumber(args[0]), this.getNumber(args[1]))
	default:
		answ := fn(this.getNumber(args[0]), this.getNumber(args[1]))
		for i := 2; i < cnt; i++ {
			answ = fn(answ, this.getNumber(args[i]))
		}
		return answ
	}
}

func (this *expression) evalMathPlus(val1, val2 int64) int64 {
	return val1 + val2
}

func (this *expression) evalMathMinus(val1, val2 int64) int64 {
	return val1 - val2
}

func (this *expression) evalMathGreater(val1, val2 int64) int64 {
	if val1 > val2 {
		return 1
	}
	return 0
}

func (this *expression) evalMathLess(val1, val2 int64) int64 {
	if val1 < val2 {
		return 1
	}
	return 0
}

func (this *expression) evalMathGreaterOrEqual(val1, val2 int64) int64 {
	if val1 >= val2 {
		return 1
	}
	return 0
}

func (this *expression) evalMathLessOrEqual(val1, val2 int64) int64 {
	if val1 <= val2 {
		return 1
	}
	return 0
}

// Get a value from the context.
func (this *expression) getValue(token *IdentityToken) string {
	return this.context[token.Name]
}

func (this *expression) toInt64(value string) int64 {
	val, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		panic(err)
	}

	return val
}
