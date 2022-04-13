package parser

import (
	"monkey/ast"
	"monkey/lexer"
	"monkey/token"
)

type Parser struct {
	l         *lexer.Lexer //Lexerインスタンスへのポインタ、このインスタンスのNextToken()を呼び出し、入力から次のトークンを繰り返し取得する
	curToken  token.Token  //Parserが現在読んでいるトークン,Parserはこのトークンを見て次に何をするか判断する
	peekToken token.Token  //Parserが次に読むトークン
}

// Lexerを読み込んで、対応するParserを生成する
func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}

	// まずは二つトークンを読み込む。これでcurTokenとpeekTokenの両方がセットされたことになる。
	p.nextToken()
	p.nextToken()

	return p
}

// Parserが現在読んでいるところと次に読むところを一つづつすすめる
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}              //ASTのルートノードを作成する
	program.Statements = []ast.Statement{} //ルートノードに構文解析されたここの文を格納する、スライス（可変配列）を用意しておく

	// token.EOFに達するまで、入力のトークンを繰り返して読む
	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement() //ここの文を構文解析して、ローカル変数stmtに格納する
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program

}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	default:
		return nil
	}
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken} //Parserが現在読んでいるトークンをlet文として、ASTを作る

	if !p.expectPeek(token.IDENT) { //letの次のトークンのタイプが識別子でなければダメ
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) { //識別子の次にくるトークンのタイプはASSIGN('='のこと)でなくてはダメ
		return nil
	}

	// TODO: セミコロンに遭遇するまで式を読み飛ばしてしまっている
	for !p.curTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// トークンタイプを入力すると、現在Parserが読んでいるトークンのタイプと一致しているか判定する
func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

// 上のpeekTokenバージョン
func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		return false
	}
}
