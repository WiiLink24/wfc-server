// Modified from github.com/zdebeer99/goexpression
package filter

import (
	"errors"
	"regexp"
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

	e := &expression{basenode, context, queryGame}
	return e.eval(basenode), nil
}

func (e *expression) eval(basenode *TreeNode) int64 {
	for _, node := range basenode.items {
		switch node.Value.Category() {
		case CatFunction:
			return e.switchFunction(node)

		case CatValue:
			return e.getNumber(node)

		case CatOther:
			return e.switchOther(node)
		}
	}
	panic("eval failed")
}

func (e *expression) switchOther(node *TreeNode) int64 {
	switch v1 := node.Value.(type) {
	case *GroupToken:
		if v1.GroupType == "()" {
			return e.eval(node)
		}
	}
	panic("invalid node " + node.String())
}

func (e *expression) switchFunction(node *TreeNode) int64 {
	val1 := node.Value.(*OperatorToken)
	switch strings.ToLower(val1.Operator) {
	case "=":
		return e.evalEquals(node.Items())
	case "==":
		return e.evalEquals(node.Items())
	case "!=":
		return e.evalNotEquals(node.Items())

	case ">":
		return e.evalMathOperator(e.evalMathGreater, node.Items())
	case "<":
		return e.evalMathOperator(e.evalMathLess, node.Items())
	case ">=":
		return e.evalMathOperator(e.evalMathGreaterOrEqual, node.Items())
	case "<=":
		return e.evalMathOperator(e.evalMathLessOrEqual, node.Items())
	case "+":
		return e.evalMathOperator(e.evalMathPlus, node.Items())
	case "-":
		return e.evalMathOperator(e.evalMathMinus, node.Items())
	case "&":
		return e.evalMathOperator(e.evalMathAnd, node.Items())
	case "|":
		return e.evalMathOperator(e.evalMathOr, node.Items())
	case "^":
		return e.evalMathOperator(e.evalMathXor, node.Items())
	case "<<":
		return e.evalMathOperator(e.evalMathLShift, node.Items())
	case ">>":
		return e.evalMathOperator(e.evalMathRShift, node.Items())

	case "and":
		return e.evalAnd(node.Items())
	case "or":
		return e.evalOr(node.Items())
	case "&&":
		return e.evalAnd(node.Items())
	case "||":
		return e.evalOr(node.Items())

	case "like":
		return e.evalLike(node.Items())

	default:
		panic("function not supported: " + val1.Operator)
	}

}

func (e *expression) getString(node *TreeNode) string {
	switch v := node.Value.(type) {
	case *NumberToken:
		return strconv.FormatInt(v.Value, 10)
	case *IdentityToken:
		return e.getValue(v)
	case *OperatorToken:
		return strconv.FormatInt(e.switchFunction(node), 10)
	case *GroupToken:
		if v.GroupType == "()" {
			return strconv.FormatInt(e.eval(node), 10)
		}
		panic("unexpected grouping type: " + node.String())
	case *TextToken:
		return node.Value.(*TextToken).Text

	default:
		panic("unexpected value: " + node.String())
	}
}

func (e *expression) evalEquals(args []*TreeNode) int64 {
	cnt := len(args)
	switch {
	case cnt < 2:
		panic("operator missing arguments")
	case cnt == 2:
		if n, ok := args[0].Value.(*IdentityToken); ok {
			if n.Name == "rk" && e.queryGame == "mariokartwii" {
				return e.evalEqualsRK(e.getString(args[1]))
			}
		}

		if e.getString(args[0]) == e.getString(args[1]) {
			return 1
		}
		return 0
	default:
		arg := e.getString(args[0])
		for i := 1; i < cnt; i++ {
			if arg != e.getString(args[i]) {
				return 0
			}
		}
		return 1
	}
}

// Operator override
func (e *expression) evalEqualsRK(value string) int64 {
	rk := e.context["rk"]
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

func (e *expression) evalNotEquals(args []*TreeNode) int64 {
	cnt := len(args)
	switch {
	case cnt < 2:
		panic("operator missing arguments")
	case cnt == 2:
		if e.getString(args[0]) != e.getString(args[1]) {
			return 1
		}
		return 0
	default:
		arg := e.getString(args[0])
		for i := 1; i < cnt; i++ {
			if arg == e.getString(args[i]) {
				return 0
			}
		}
		return 1
	}
}

func (e *expression) evalAnd(args []*TreeNode) int64 {
	cnt := len(args)
	if cnt < 2 {
		panic("operator missing arguments")
	}

	for i := 0; i < cnt; i++ {
		if e.getString(args[i]) == "0" {
			return 0
		}
	}
	return 1
}

func (e *expression) evalOr(args []*TreeNode) int64 {
	cnt := len(args)
	if cnt < 2 {
		panic("operator missing arguments")
	}

	for i := 0; i < cnt; i++ {
		if e.getString(args[i]) != "0" {
			return 1
		}
	}
	return 0
}

func (e *expression) getNumber(node *TreeNode) int64 {
	switch v := node.Value.(type) {
	case *NumberToken:
		return v.Value
	case *IdentityToken:
		r1 := e.getValue(v)
		return e.toInt64(r1)
	case *OperatorToken:
		return e.switchFunction(node)
	case *GroupToken:
		if v.GroupType == "()" {
			return e.eval(node)
		}
		panic("unexpected grouping type: " + node.String())
	case *TextToken:
		return e.toInt64(node.Value.(*TextToken).Text)

	default:
		panic("unexpected value: " + node.String())
	}
}

func (e *expression) evalMathOperator(fn func(int64, int64) int64, args []*TreeNode) int64 {
	cnt := len(args)
	switch {
	case cnt < 2:
		panic("operator missing arguments")
	case cnt == 2:
		if n, ok := args[0].Value.(*IdentityToken); ok {
			// Remove VR search due to the limited player count
			if (n.Name == "ev" || n.Name == "eb") && e.queryGame == "mariokartwii" {
				return 1
			}
		}

		return fn(e.getNumber(args[0]), e.getNumber(args[1]))
	default:
		answ := fn(e.getNumber(args[0]), e.getNumber(args[1]))
		for i := 2; i < cnt; i++ {
			answ = fn(answ, e.getNumber(args[i]))
		}
		return answ
	}
}

func (e *expression) evalMathPlus(val1, val2 int64) int64 {
	return val1 + val2
}

func (e *expression) evalMathMinus(val1, val2 int64) int64 {
	return val1 - val2
}

func (e *expression) evalMathGreater(val1, val2 int64) int64 {
	if val1 > val2 {
		return 1
	}
	return 0
}

func (e *expression) evalMathLess(val1, val2 int64) int64 {
	if val1 < val2 {
		return 1
	}
	return 0
}

func (e *expression) evalMathGreaterOrEqual(val1, val2 int64) int64 {
	if val1 >= val2 {
		return 1
	}
	return 0
}

func (e *expression) evalMathLessOrEqual(val1, val2 int64) int64 {
	if val1 <= val2 {
		return 1
	}
	return 0
}

func (e *expression) evalMathAnd(val1, val2 int64) int64 {
	return val1 & val2
}

func (e *expression) evalMathOr(val1, val2 int64) int64 {
	return val1 | val2
}

func (e *expression) evalMathXor(val1, val2 int64) int64 {
	return val1 ^ val2
}

func (e *expression) evalMathLShift(val1, val2 int64) int64 {
	return val1 << val2
}

func (e *expression) evalMathRShift(val1, val2 int64) int64 {
	return val1 >> val2
}

func (e *expression) evalLike(args []*TreeNode) int64 {
	cnt := len(args)
	switch {
	case cnt < 2:
		panic("operator missing arguments")
	case cnt == 2:
		return e.evalLikeSingle(args[0], args[1])
	default:
		panic("operator LIKE does not support multiple arguments")
	}
}

func (e *expression) evalLikeSingle(arg1, arg2 *TreeNode) int64 {
	val1 := e.getString(arg1)
	val2 := e.getString(arg2)

	allowedCharacters := `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_%\`

	regexString := "^"

	// Convert SQL LIKE pattern to regex
	for i, c := range val2 {
		if !strings.ContainsRune(allowedCharacters, c) {
			panic("invalid character in LIKE pattern: " + string(c))
		}

		if i != 0 && val2[i-1] == '\\' {
			if c == '\\' {
				regexString += "\\\\"
				continue
			}

			regexString += string(c)
			continue
		}

		switch c {
		case '%':
			regexString += ".*"

		case '_':
			regexString += "."

		case '\\':
			// Do nothing

		default:
			regexString += string(c)
		}
	}

	regexString += "$"

	if matched, err := regexp.MatchString(regexString, val1); err != nil {
		panic(err)
	} else if matched {
		return 1
	}

	return 0
}

// Get a value from the context.
func (e *expression) getValue(token *IdentityToken) string {
	return e.context[token.Name]
}

func (e *expression) toInt64(value string) int64 {
	val, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		panic(err)
	}

	return val
}
