package jsx

import (
	"fmt"
	"runtime"
)

type AttrType int

const (
	HtmlAttr AttrType = iota
	JsAttr
)

type Attr struct {
	Payload string
	Typ     AttrType
}

type Node interface {
	Node()
}

type ElementNode struct {
	Name        string
	Children    []Node
	Attrs       map[string]Attr
	SpreadAttrs []JsNode
}

func (ElementNode) Node() {}

type TextNode string

func (TextNode) Node() {}

type JsNode string

func (JsNode) Node() {}

type Parser struct {
	lexer   *lexer
	err     error
	it      *item //lookahead
	root    *ElementNode
	lastPos int
}

func (p *Parser) nextItem() (it item) {
	if p.it != nil {
		it = *p.it
		p.it = nil
		return it
	}
	return p.lexer.nextItem()
}

func (p *Parser) peekItem() item {
	it := p.nextItem()
	p.it = &it
	return it
}

func (p *Parser) expectItem(typ itemType) item {
	it := p.nextItem()
	if it.typ != typ {
		pcs := make([]uintptr, 5)
		pcs = pcs[:runtime.Callers(1, pcs)]
		s := ""
		for _, pc := range pcs {
			s += " <- " + runtime.FuncForPC(pc).Name()
		}
		panic(fmt.Errorf("[%s]: expected token of type %v", s, typ))
	}
	return it
}

func (p *Parser) Parse() (finalErr error) {
	defer func() {
		if x := recover(); x != nil {
			if err, ok := x.(error); ok {
				p.err = err
				finalErr = err
			} else {
				panic(x)
			}
		}
	}()
	p.root = p.parseElement()
	return nil
}

func (p *Parser) parseElement() *ElementNode {
	// opening tag
	it := p.expectItem(itemOpeningTag)
	node := &ElementNode{
		Name:  it.val,
		Attrs: make(map[string]Attr),
	}

	for {
		it = p.peekItem()
		switch it.typ {
		case itemAttributeName:
			// <node attr
			// eat token
			key, val := p.parseAttribute()
			node.Attrs[key] = val
		case itemLeftDelim:
			// <node {
			node.SpreadAttrs = append(node.SpreadAttrs, p.parseSpreadAttributes())
		case itemEndOpeningTag:
			p.it = nil
			return p.parseChildren(node)
		case itemSelfClosingTag:
			p.lastPos = p.it.lastPos
			p.it = nil
			return node
		default:
			panic(fmt.Errorf("parseElement: unexpected token %#v in %q", it, p.lexer.input[p.lexer.start:p.lexer.start+10]))
		}
	}
}

func (p *Parser) parseChildren(node *ElementNode) *ElementNode {
	for {
		it := p.peekItem()
		switch it.typ {
		case itemOpeningTag:
			// <node><child>
			node.Children = append(node.Children, p.parseElement())
		case itemText:
			// <node>text
			node.Children = append(node.Children, p.parseText())
		case itemClosingTag:
			p.lastPos = p.it.lastPos
			// eat token
			p.it = nil
			// </node>
			// the end
			return node
		case itemLeftDelim:
			// <node>{
			js := p.parseJs()
			node.Children = append(node.Children, js)
		default:
			panic(fmt.Errorf("parseChildren: unexpected token %#v in %q", it, p.lexer.input[p.lexer.start:p.lexer.start+10]))
		}
	}
}

func (p *Parser) parseText() TextNode {
	it := p.expectItem(itemText)
	return TextNode(it.val)
}

func (p *Parser) parseJs() JsNode {
	p.expectItem(itemLeftDelim)
	it := p.expectItem(itemJS)
	p.expectItem(itemRightDelim)
	return JsNode(it.val)
}

func (p *Parser) parseAttribute() (string, Attr) {
	it := p.expectItem(itemAttributeName)
	key := it.val
	switch p.peekItem().typ {
	case itemAttributeValue:
		return key, p.parseAttributeValue()
	case itemLeftDelim:
		val := p.parseJs()
		return key, Attr{Payload: string(val), Typ: JsAttr}
	default:
		panic(fmt.Errorf("parseAttribute: unexpected token %#v", *p.it))
	}
}

func (p *Parser) parseAttributeValue() Attr {
	it := p.expectItem(itemAttributeValue)
	return Attr{Payload: string(it.val), Typ: HtmlAttr}
}

func (p *Parser) parseSpreadAttributes() JsNode {
	p.expectItem(itemLeftDelim)
	p.expectItem(itemEllipsis)
	it := p.expectItem(itemJS)
	p.expectItem(itemRightDelim)
	return JsNode(it.val)
}
