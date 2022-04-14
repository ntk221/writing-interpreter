package parser

import (
	"fmt"
	"monkey/ast"
	"monkey/lexer"
	"monkey/token"
)

type Parser struct {
	l         *lexer.Lexer // Lexer インスタンスへのポインタ、このインスタンスの NextToken() を呼び出し、入力から次のトークンを繰り返し取得する
	curToken  token.Token  // Parser が現在読んでいるトークン, Parser はこのトークンを見て次に何をするか判断する
	peekToken token.Token  // Parser が次に読むトークン
	errors    []string     // Parser が文字列で表現されたエラーの情報を保持するための配列
}

// Lexer を読み込んで、対応する Parser を生成する
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

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
	program.Statements = []ast.Statement{} //ルートノードに構文解析されたここの文を格納する、スライス（可変配列）を用意しておく

	// token.EOF に達するまで、入力のトークンを繰り返して読む
	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement() //現在読んでいるトークンタイプがEOF出ないとき、その文を構文解析してローカル変数 stmt に格納する
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program

}

// 文の種類によって対応した構文解析をするメソッド
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	default:
		return nil
	}
}

// let 文の構文を解析するメソッド
func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken} //Parser が現在読んでいるトークンをlet文として、AST のルートノードを作る

	if !p.expectPeek(token.IDENT) { //let の次にくるトークンのタイプは識別子でなければならない。ここで、expectPeek メソッドを使っていることで、Parser が現在読んでいる箇所が一つ進んでいることに注意！
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal} // トークンの情報を用いて、AST の Identifier ノードを生成する

	if !p.expectPeek(token.ASSIGN) { //識別子の次にくるトークンのタイプはASSIGN('='のこと)でなくてはダメ
		return nil
	}

	// TODO: セミコロンに遭遇するまで式を読み飛ばしてしまっている
	for !p.curTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// トークンタイプを入力すると、現在 Parser が読んでいるトークンのタイプと一致しているか判定する
func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

// 上のpeekTokenバージョン
func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

// トークンのタイプを入力して、Parser が次に読むトークンのタイプと一致する時に true を返す。 一致したい時にはpeekErrorを用いて、Parserにエラーの情報を追加する
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
