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

	this := &expression{basenode, "", conn}
	this.filterAppendRoot(basenode)
	return "(" + this.query + ")", nil
}

func (this *expression) filterAppendRoot(basenode *filter.TreeNode) {
	for _, node := range basenode.Items() {
		switch node.Value.Category() {
		case filter.CatFunction:
			this.filterSwitchFunction(node)
			return

		case filter.CatValue:
			this.filterAppendNode(node)
			return

		case filter.CatOther:
			this.filterSwitchOther(node)
			return
		}
	}
	panic("eval failed")
}

func (this *expression) filterSwitchOther(node *filter.TreeNode) {
	switch v1 := node.Value.(type) {
	case *filter.GroupToken:
		if v1.GroupType == "()" {
			this.filterAppendRoot(node)
			return
		}
	}
	panic("invalid node " + node.String())
}

func (this *expression) filterSwitchFunction(node *filter.TreeNode) {
	val1 := node.Value.(*filter.OperatorToken)
	switch strings.ToLower(val1.Operator) {
	case "=", "!=":
		this.filterAppendOperator(strings.ToLower(val1.Operator), node.Items())

	case ">", "<", ">=", "<=", "+", "-", "&", "|", "^", "<<", ">>":
		this.filterAppendMathOperator(strings.ToLower(val1.Operator), node.Items())

	case "and":
		this.filterAppendAnd(node.Items())
	case "or":
		this.filterAppendOr(node.Items())

	default:
		panic("function not supported: " + val1.Operator)
	}

}

func (this *expression) filterAppendNode(node *filter.TreeNode) {
	switch v := node.Value.(type) {
	case *filter.NumberToken:
		this.query += "'" + strconv.FormatInt(v.Value, 10) + "'"
	case *filter.IdentityToken:
		this.filterAppendQueryValue(v)
	case *filter.OperatorToken:
		this.filterSwitchFunction(node)
	case *filter.GroupToken:
		if v.GroupType == "()" {
			this.query += "("
			this.filterAppendRoot(node)
			this.query += ")"
			return
		}
		panic("unexpected grouping type '" + v.GroupType + "': " + node.String())
	case *filter.TextToken:
		this.query += "(" + this.filterPushArg(v.Text) + ")::varchar"

	default:
		panic("unexpected value: " + node.String())
	}
}

func (this *expression) filterAppendAnd(args []*filter.TreeNode) {
	cnt := len(args)
	if cnt < 2 {
		panic("operator missing arguments")
	}

	this.query += "( "
	this.filterAppendNode(args[0])
	this.query += " AND "
	this.filterAppendNode(args[1])
	this.query += " )"
}

func (this *expression) filterAppendOr(args []*filter.TreeNode) {
	cnt := len(args)
	if cnt < 2 {
		panic("operator missing arguments")
	}

	this.query += "( "
	this.filterAppendNode(args[0])
	this.query += " OR "
	this.filterAppendNode(args[1])
	this.query += " )"
}

func (this *expression) filterAppendOperator(operator string, args []*filter.TreeNode) {
	cnt := len(args)
	if cnt != 2 {
		panic("operator requires exactly 2 arguments")
	}
	this.query += "( "
	this.filterAppendNode(args[0])
	this.query += " " + operator + " "
	this.filterAppendNode(args[1])
	this.query += " )"
}

func (this *expression) filterAppendMathOperator(operator string, args []*filter.TreeNode) {
	cnt := len(args)
	if cnt != 2 {
		panic("operator requires exactly 2 arguments")
	}
	this.query += "( ("
	this.filterAppendNode(args[0])
	this.query += ")::int " + operator + " ("
	this.filterAppendNode(args[1])
	this.query += ")::int )"
}

// Get a value from the record
func (this *expression) filterAppendQueryValue(token *filter.IdentityToken) {
	if token.Name == "ownerid" {
		this.query += "(owner_id)"
		return
	}
	if token.Name == "recordid" {
		this.query += "(record_id)"
		return
	}
	if token.Name == "gameid" {
		this.query += "(game_id)"
		return
	}
	if token.Name == "tableid" {
		this.query += "(table_id)"
		return
	}

	this.query += "COALESCE(fields->" + this.filterPushArg(token.Name) + "->>'value', '0')"

}

func (this *expression) filterPushArg(arg string) string {
	// This is scary!!!
	if this.conn == nil {
		return `'` + strings.Replace(arg, "'", "''", -1) + `'`
	}

	str, err := this.conn.EscapeString(arg)
	if err != nil {
		panic(err)
	}
	return `'` + str + `'`
}
