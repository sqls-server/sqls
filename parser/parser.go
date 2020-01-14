package parser

import (
	"io"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
	"github.com/pkg/errors"
)

type writeContext struct {
	node     ast.TokenList
	curNode  ast.Node
	peekNode ast.Node
	index    uint
}

func newWriteContext(list ast.TokenList) *writeContext {
	wc := &writeContext{
		node: list,
	}
	wc.nextNode()
	return wc
}

func (wc *writeContext) nodesWithRange(startIndex, endIndex uint) []ast.Node {
	return wc.node.GetTokens()[startIndex:endIndex]
}

func (wc *writeContext) replace(add ast.Node, startIndex, endIndex uint) {
	oldList := wc.node.GetTokens()

	start := oldList[:startIndex]
	end := oldList[endIndex:]

	var out []ast.Node
	out = append(out, start...)
	out = append(out, add)
	out = append(out, end...)
	wc.node.SetTokens(out)

	offset := (endIndex - startIndex)
	wc.index = wc.index - uint(offset)
	wc.nextNode()
}

func (wc *writeContext) hasNext() bool {
	return wc.index < uint(len(wc.node.GetTokens()))
}

func (wc *writeContext) nextNode() bool {
	if !wc.hasNext() {
		return false
	}
	wc.curNode = wc.node.GetTokens()[wc.index]
	wc.index++
	return true
}

func (wc *writeContext) hasTokenList() bool {
	_, ok := wc.curNode.(ast.TokenList)
	return ok
}

func (wc *writeContext) getTokenList() ast.TokenList {
	if !wc.hasTokenList() {
		return nil
	}
	children, _ := wc.curNode.(ast.TokenList)
	return children
}

func (wc *writeContext) hasToken() bool {
	_, ok := wc.curNode.(ast.Token)
	return ok
}

func (wc *writeContext) getToken() *ast.SQLToken {
	if !wc.hasToken() {
		return nil
	}
	token, _ := wc.curNode.(ast.Token)
	return token.GetToken()
}

type Parser struct {
	root ast.TokenList
}

func NewParser(src io.Reader, d dialect.Dialect) (*Parser, error) {
	tokenizer := token.NewTokenizer(src, d)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		return nil, errors.Errorf("tokenize err failed: %w", err)
	}

	parsed := []ast.Node{}
	for _, tok := range tokens {
		parsed = append(parsed, ast.NewItem(tok))
	}

	parser := &Parser{
		root: &ast.Query{Toks: parsed},
	}

	return parser, nil
}

func (p *Parser) Parse() (ast.TokenList, error) {
	// var result []ast.Node
	var err error

	if err = parseStatement(newWriteContext(p.root)); err != nil {
		return nil, err
	}
	return p.root, nil
	// p.tokens = result

	// result, err = parseIdentifier(p.tokens)
	// if err != nil {
	// 	return nil, err
	// }
	// p.tokens = result

	// return p.tokens, nil
}

// func (p *Parser) nextTokenByKind(expect token.Kind) (ast.Node, error) {
// 	for {
// 		tok, err := p.nextToken()
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		if expect == tok.Kind {
// 			return tok, nil
// 		}
// 	}
// 	return nil, EOF
// }

// func (p *Parser) expectKeyword(expected string) ast.Node {
// 	ok, tok, err := p.parseKeyword(expected)
// 	if err != nil || !ok {
// 		for i := 0; i < int(p.index); i++ {
// 			fmt.Printf("%v", p.tokens[i].Value)
// 		}
// 		fmt.Println()
// 		log.Fatalf("should be expected keyword: %s err: %v", expected, err)
// 	}
//
// 	return tok
// }
//
// func (p *Parser) parseKeyword(expected string) (bool, ast.Node, error) {
// 	tok, err := p.peekToken()
// 	if err != nil {
// 		return false, nil, errors.Errorf("parseKeyword %s failed: %w", expected, err)
// 	}
//
// 	word, ok := tok.Value.(*token.SQLWord)
// 	if !ok {
// 		return false, tok, nil
// 	}
//
// 	if strings.EqualFold(word.Value, expected) {
// 		p.mustNextToken()
// 		return true, tok, nil
// 	}
// 	return false, tok, nil
// }

// func (p *Parser) peekToken() (ast.Node, error) {
// 	u, err := p.tilNonWhitespace()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return p.tokens[u], nil
// }

// func (p *Parser) tilNonWhitespace() (uint, error) {
// 	idx := p.index
// 	for {
// 		if idx >= uint(len(p.tokens)) {
// 			return 0, EOF
// 		}
// 		tok := p.tokens[idx]
// 		if tok.Kind == token.Whitespace || tok.Kind == token.Comment {
// 			idx += 1
// 			continue
// 		}
// 		return idx, nil
// 	}
// }
//
// func (p *Parser) expectToken(expected token.Kind) {
// 	ok, err := p.consumeToken(expected)
// 	if err != nil || !ok {
// 		tok, _ := p.peekToken()
//
// 		for i := 0; i < int(p.index); i++ {
// 			fmt.Printf("%v", p.tokens[i].Value)
// 		}
// 		fmt.Println()
// 		log.Fatalf("should be %s token, but %+v,  err: %+v", expected, tok, err)
// 	}
// }
//
// func (p *Parser) consumeToken(expected token.Kind) (bool, error) {
// 	tok, err := p.peekToken()
// 	if err != nil {
// 		return false, err
// 	}
//
// 	if tok.Kind == expected {
// 		if _, err := p.nextToken(); err != nil {
// 			return false, err
// 		}
// 		return true, nil
// 	}
//
// 	return false, nil
// }

func parseStatement(wc *writeContext) error {
	var startIndex uint
	for wc.nextNode() {
		if wc.hasTokenList() {
			parseStatement(newWriteContext(wc.getTokenList()))
		}

		tok := wc.getToken()
		if tok == nil {
			return errors.Errorf("failed parse statement want Token")
		}
		if tok.MatchKind(token.Semicolon) {
			stmt := &ast.Statement{Toks: wc.nodesWithRange(startIndex, wc.index)}
			wc.replace(stmt, startIndex, wc.index)
			startIndex = wc.index
		}
	}
	if wc.index != startIndex {
		stmt := &ast.Statement{Toks: wc.nodesWithRange(startIndex, wc.index)}
		wc.replace(stmt, startIndex, wc.index)
	}
	return nil
}

// parseComments

// func (p *Parser) parseBrackets() {
// }

// func (p *Parser) parseParenthesis() ([]ast.Node, error) {
// 	start := token.LBracket
// 	end := token.RBracket
//
// 	opens := []uint{}
// 	parsed := []ast.Node{}
//
// 	for _, t := range p.tokens {
// 		tok, ok := t.(ast.Token)
// 		if !ok {
// 			return nil, errors.Errorf("err")
// 		}
// 		realToken := tok.Token()
//
// 		if realToken.Kind == token.Whitespace || realToken.Kind == token.Comment {
// 			continue
// 		}
//
// 		if realToken.Kind == start {
// 			opens = append(opens, p.index)
// 		}
// 		if realToken.Kind == end {
// 			n := len(opens) - 1
// 			openIdx := opens[n]
// 			opens = opens[:n]
// 			closeIdx := p.index
// 			ast.NewGrouped(p.tokens[openIdx:closeIdx])
// 		}
// 	}
// 	return parsed, nil
// }

// parseCase
// parseIf
// parseFor
// parseBegin

// parseFunctions
// parseWhere

// func parsePeriod(tokens []ast.Node) ([]ast.Node, error) {
// 	parsed := []ast.Node{}
//
// 	for _, t := range tokens {
// 		if tokList, ok := t.(ast.TokenList); ok {
// 			res, err := parseIdentifier(tokList.Tokens())
// 			if err != nil {
// 				return nil, err
// 			}
// 			tokList.SetTokens(res)
// 			parsed = append(parsed, tokList)
// 			continue
// 		}
//
// 		if tok, ok := t.(ast.Token); ok {
// 			realToken := tok.Token()
// 			if realToken.MatchSQLKind(dialect.Unmatched) {
// 				parsed = append(parsed, ast.NewIdentifier(realToken))
// 			} else {
// 				parsed = append(parsed, t)
// 			}
// 			continue
// 		}
//
// 		return nil, errors.Errorf("parse error want Token or TokenList got %T", t)
// 	}
// 	return parsed, nil
// }

// parseArrays

// func parseIdentifier(tokens []ast.Node) ([]ast.Node, error) {
// 	parsed := []ast.Node{}
//
// 	for _, t := range tokens {
// 		if tokList, ok := t.(ast.TokenList); ok {
// 			res, err := parseIdentifier(tokList.Tokens())
// 			if err != nil {
// 				return nil, err
// 			}
// 			tokList.SetTokens(res)
// 			parsed = append(parsed, tokList)
// 			continue
// 		}
//
// 		if tok, ok := t.(ast.Token); ok {
// 			realToken := tok.Token()
// 			if realToken.MatchSQLKind(dialect.Unmatched) {
// 				parsed = append(parsed, ast.NewIdentifier(realToken))
// 			} else {
// 				parsed = append(parsed, t)
// 			}
// 			continue
// 		}
//
// 		return nil, errors.Errorf("parse error want Token or TokenList got %T", t)
// 	}
// 	return parsed, nil
// }

// parseOrder
// parseTypecasts
// parseTzcasts
// parseTyped_literal
// parseOperator
// parseComparison
// parseAs
// parseAliased
// parseAssignment

// alignComments
// parseIdentifierList
// parseValues
