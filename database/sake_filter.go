package database

import (
	"errors"
	"strconv"
	"strings"
	"wwfc/filter"

	"github.com/jackc/pgconn"
)

type expression struct {
	ast   *filter.TreeNode
	query string
	conn  *pgconn.PgConn
}

func createSqlFilter(conn *pgconn.PgConn, basenode *filter.TreeNode) (value string, err error) {
	defer func() {
		if str := recover(); str != nil {
			value = ""
			err = errors.New(str.(string))
		}
	}()

	e := &expression{basenode, "", conn}
	e.filterAppendRoot(basenode)
	return "(" + e.query + ")", nil
}

func (e *expression) filterAppendRoot(basenode *filter.TreeNode) {
	for _, node := range basenode.Items() {
		switch node.Value.Category() {
		case filter.CatFunction:
			e.filterSwitchFunction(node)
			return

		case filter.CatValue:
			e.filterAppendNode(node)
			return

		case filter.CatOther:
			e.filterSwitchOther(node)
			return
		}
	}
	panic("eval failed")
}

func (e *expression) filterSwitchOther(node *filter.TreeNode) {
	switch v1 := node.Value.(type) {
	case *filter.GroupToken:
		if v1.GroupType == "()" {
			e.filterAppendRoot(node)
			return
		}
	}
	panic("invalid node " + node.String())
}

func (e *expression) filterSwitchFunction(node *filter.TreeNode) {
	switch v := node.Value.(type) {
	case *filter.OperatorToken:
		e.filterSwitchOperator(node, v)
	case *filter.FuncToken:
		switch strings.ToLower(v.Name) {
		case "substring":
			e.filterAppendFuncSubstring(v)
		default:
			panic("function not supported: " + v.Name)
		}
	default:
		panic("unexpected function type: " + node.String())
	}
}

func (e *expression) filterSwitchOperator(node *filter.TreeNode, val1 *filter.OperatorToken) {
	switch strings.ToLower(val1.Operator) {
	case "=", "!=":
		e.filterAppendOperator(strings.ToLower(val1.Operator), node.Items())

	case ">", "<", ">=", "<=", "+", "-", "&", "|", "^", "<<", ">>":
		e.filterAppendMathOperator(strings.ToLower(val1.Operator), node.Items())

	case "and":
		e.filterAppendAnd(node.Items())
	case "or":
		e.filterAppendOr(node.Items())

	default:
		panic("operator not supported: " + val1.Operator)
	}
}

func (e *expression) filterAppendNode(node *filter.TreeNode) {
	switch v := node.Value.(type) {
	case *filter.NumberToken:
		e.query += "'" + strconv.FormatInt(v.Value, 10) + "'"
	case *filter.IdentityToken:
		e.filterAppendQueryValue(v)
	case *filter.OperatorToken, *filter.FuncToken:
		e.filterSwitchFunction(node)
	case *filter.GroupToken:
		if v.GroupType == "()" {
			e.query += "("
			e.filterAppendRoot(node)
			e.query += ")"
			return
		}
		panic("unexpected grouping type '" + v.GroupType + "': " + node.String())
	case *filter.TextToken:
		e.query += "(" + e.filterPushArg(v.Text) + ")::varchar"

	default:
		panic("unexpected value: " + node.String())
	}
}

func (e *expression) filterAppendAnd(args []*filter.TreeNode) {
	cnt := len(args)
	if cnt < 2 {
		panic("operator missing arguments")
	}

	e.query += "( "
	e.filterAppendNode(args[0])
	e.query += " AND "
	e.filterAppendNode(args[1])
	e.query += " )"
}

func (e *expression) filterAppendOr(args []*filter.TreeNode) {
	cnt := len(args)
	if cnt < 2 {
		panic("operator missing arguments")
	}

	e.query += "( "
	e.filterAppendNode(args[0])
	e.query += " OR "
	e.filterAppendNode(args[1])
	e.query += " )"
}

func (e *expression) filterAppendOperator(operator string, args []*filter.TreeNode) {
	cnt := len(args)
	if cnt != 2 {
		panic("operator requires exactly 2 arguments")
	}
	e.query += "( "
	e.filterAppendNode(args[0])
	e.query += " " + operator + " "
	e.filterAppendNode(args[1])
	e.query += " )"
}

func (e *expression) filterAppendMathOperator(operator string, args []*filter.TreeNode) {
	cnt := len(args)
	if cnt != 2 {
		panic("operator requires exactly 2 arguments")
	}
	e.query += "( ("
	e.filterAppendNode(args[0])
	e.query += ")::bigint " + operator + " ("
	e.filterAppendNode(args[1])
	e.query += ")::bigint )"
}

func (e *expression) filterAppendFuncSubstring(token *filter.FuncToken) {
	if len(token.Arguments) != 3 {
		panic("substring requires exactly 3 arguments")
	}

	e.query += "SUBSTRING( "
	e.filterAppendRoot(token.Arguments[0])
	e.query += ", ("
	e.filterAppendRoot(token.Arguments[1])
	e.query += ")::bigint, ("
	e.filterAppendRoot(token.Arguments[2])
	e.query += ")::bigint )"
}

// Get a value from the record
func (e *expression) filterAppendQueryValue(token *filter.IdentityToken) {
	if token.Name == "ownerid" {
		e.query += "(owner_id)"
		return
	}
	if token.Name == "recordid" {
		e.query += "(record_id)"
		return
	}
	if token.Name == "gameid" {
		e.query += "(game_id)"
		return
	}
	if token.Name == "tableid" {
		e.query += "(table_id)"
		return
	}

	e.query += "COALESCE(fields->" + e.filterPushArg(token.Name) + "->>'value', '0')"
}

func (e *expression) filterPushArg(arg string) string {
	// This is scary!!!
	if e.conn == nil {
		return `'` + strings.ReplaceAll(arg, "'", "''") + `'`
	}

	str, err := e.conn.EscapeString(arg)
	if err != nil {
		panic(err)
	}
	return `'` + str + `'`
}
