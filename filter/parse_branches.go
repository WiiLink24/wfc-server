// Modified from github.com/zdebeer99/goexpression
package filter

func (p *parser) pumpExpression() {
	p.state = branchExpressionValuePart
	for p.state != nil {
		if p.err != nil {
			break
		}
		p.state = p.state(p)
	}
	endo := p.commit()
	if len(endo) > 0 || !p.scan.IsEOF() {
		panic("unexpected end of expression '" + endo + "' not parsed")
	}
}

/*
parse expressions
[value part][operator part] repeat
*/

func branchExpressionValuePart(this *parser) stateFn {
	scan := this.scan
	scan.SkipSpaces()
	if scan.IsEOF() {
		return nil
	}
	if scan.ScanNumber() {
		this.add(NewNumberToken(scan.Commit()))
		return branchExpressionOperatorPart
	}
	if scan.ScanWord() {
		return branchExpressionAfterWord
	}
	c := scan.Next()
	switch c {
	case '"', '\'':
		scan.Backup()
		txt := this.ParseText()
		this.add(NewTextToken(txt))
		return branchExpressionOperatorPart
	case '(':
		this.parseOpenBracket()
		return branchExpressionValuePart
	}

	panic("unexpected token: " + string(c))
}

func branchExpressionAfterWord(this *parser) stateFn {
	scan := this.scan
	switch scan.Peek() {
	case '(':
		this.curr = this.add(NewFuncToken(scan.Commit()))
		return branchFunctionArguments
	}
	this.add(NewIdentityToken(scan.Commit()))
	return branchExpressionOperatorPart
}

func branchFunctionArguments(this *parser) stateFn {
	scan := this.scan
	r := scan.Next()
	if r != '(' {
		panic("expecting '(' before arguments")
	}
	ftoken, ok := this.curr.Value.(*FuncToken)
	if !ok {
		panic("expecting function token to add arguments to")
	}
	state := branchExpressionValuePart
	currnode := this.curr
	for {
		this.curr = NewTreeNode(NewGroupToken(""))
		for state != nil {
			state = state(this)
		}
		r = scan.Next()
		switch r {
		case ' ':
			scan.Ignore()
			continue
		case ',':
			ftoken.AddArgument(this.curr.Root())
			state = branchExpressionValuePart
			scan.Ignore()
			continue
		case ')':
			ftoken.AddArgument(this.curr.Root())
			this.curr = currnode.parent
			scan.Ignore()
			return branchExpressionOperatorPart
		}
		this.curr = currnode
		if scan.IsEOF() {
			panic("arguments missing end bracket")
		}
		panic("invalid char '" + string(r) + "' in arguments")
	}
}

func branchExpressionOperatorPart(this *parser) stateFn {
	scan := this.scan
	scan.SkipSpaces()

	if scan.IsEOF() {
		return nil
	}
	if this.AcceptOperator() {
		this.parseOperator()
		return branchExpressionValuePart
	}
	if scan.Accept("=") {
		this.parseLRFunc()
		this.curr = this.add(NewGroupToken(""))
		return branchExpressionValuePart
	}
	switch scan.Next() {
	case ')':
		return this.parseCloseBracket()
	}
	scan.Rollback()
	return nil
}
