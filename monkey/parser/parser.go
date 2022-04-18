package parser

import (
	"fmt"
	"monkey/ast"
	"monkey/lexer"
	"monkey/token"
	"strconv"
)

// 演算子の優先順位を決める部分
const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      //  -X or !X
	CALL        // myfunction(X)
)

type Parser struct {
	l         *lexer.Lexer // Lexer インスタンスへのポインタ、このインスタンスの NextToken() を呼び出し、入力から次のトークンを繰り返し取得する
	curToken  token.Token  // Parser が現在読んでいるトークン, Parser はこのトークンを見て次に何をするか判断する
	peekToken token.Token  // Parser が次に読むトークン
	errors    []string     // Parser が文字列で表現されたエラーの情報を保持するための配列

	// これらのマップを用いて、現在読み込んでいるトークンに対応する構文解析関数があるかチェックできる
	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression // infix構文解析関数は、構文解析中のinfix演算子の「左側の式」を引数にとる
)

// Lexer を読み込んで、対応する Parser を生成する
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// New()された時には、prefixParsoFnsマップを初期化して,構文解析関数を登録する
	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)   // トークンタイプ token.IDENT が出現したときに呼び出す構文解析関数はparseIdentifier
	p.registerPrefix(token.INT, p.parseIntegerLiteral) // トークンタイプ token.INT が出現したときに呼び出す構文解析関数はparseIntegerLiteral

	// まずは二つトークンを読み込む。これで curToken と peekToken の両方がセットされたことになる。
	p.nextToken()
	p.nextToken()

	return p
}

// Parser が現在読んでいるところと次に読むところを一つづつすすめる
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// トークン列を読み込んだParserに構文解析させるメソッド
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}              // AST のルートノードを作成する
	program.Statements = []ast.Statement{} //ルートノードに構文解析された文を格納する、スライス（可変配列）を用意しておく

	// token.EOF に達するまで、入力のトークンを繰り返して読む
	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement() //現在読んでいるトークンタイプがEOF出ないとき、その文を構文解析してローカル変数 stmt に格納する
		if stmt != nil {
			program.Statements = append(program.Statements, stmt) // program の Statements フィールドに追加していく
		}
		p.nextToken()
	}

	return program

}

// 現座読んでいるトークンの種類によって対応した構文解析をするメソッド
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	default: // let文でも,return文でもない時には式文の構文解析を始める
		return p.parseExpressionStatement()
	}
}

// let 文の構文を解析するメソッド
func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken} //Parser が現在読んでいるトークンをlet文として、let文のノードを作る

	if !p.expectPeek(token.IDENT) { //let の次にくるトークンのタイプは識別子でなければならない。ここで、expectPeek メソッドを使っていることで、Parser が現在読んでいる箇所が一つ進んでいることに注意！
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal} // トークンの情報を用いて、Identifier ノードを生成し、ルートの Name フィールドにこの Identifier ノードのアドレスを入れておく

	if !p.expectPeek(token.ASSIGN) { //識別子の次にくるトークンのタイプはASSIGN('='のこと)でなくてはダメ
		return nil
	}

	// TODO: セミコロンに遭遇するまで式を読み飛ばしてしまっている
	for !p.curTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	// TODO: セミコロンに遭遇するまで読み飛ばしてしまっている
	for !p.curTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// Parser が現在読んでいるトークンが式文である時に、構文解析関数parseEcpression()を用いて、stmtのExpression フィールドを埋める
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) { // セミコロンの部分は省略可能
		p.nextToken()
	}
	return stmt
}

// Parser が現在読んでいるトークンの"前置"に関連づけられた構文解析関数があるか確認し、sるときにはそれを呼び出す
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type] // 現在読んでいるトークンのタイプに関連づけられた構文解析関数があるとき、それを prefix に格納
	if prefix == nil {
		return nil
	}
	leftExp := prefix() // ある時にはそのprefix関数を呼び出し、その結果を返す

	return leftExp
}

// 現在のトークンを Token フィールドに,トークンのリテラル値を Value フィールドに格納する。
func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal} // Parser が現在読んでいるトークンは進めない！
}

// Parser が現在読んでいるトークンを用いて、IntegerLiteralのASTノードを生成し、トークンのリテラル値を整数値にパースして、Valueフィールドを埋める
func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value

	return lit
}

// トークンタイプを入力すると、現在 Parser が読んでいるトークンのタイプと一致しているか判定する
func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

// 上のpeekTokenバージョン
func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

// トークンのタイプを入力して、Parser が次に読むトークンのタイプと一致する時に true を返す。 一致しない時には peekError() を用いて、Parserにエラーの情報を追加する
func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// Parser が保持しているエラー情報を返す。 テストで使う。
func (p *Parser) Errors() []string {
	return p.errors
}

// peekToken のタイプが期待に合わない時に、そのトークンのタイプを入力して、エラーメッセージをParserに追加するメソッド
func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

// Parser の prefixParserFns マップにエントリを追加するための補助関数
func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}
