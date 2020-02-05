package jsx

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/parser"
)

// React implements Visitor.
// Should be called with Walk(NewReact(...), node).
type React struct {
	fset    *file.FileSet
	errList parser.ErrorList
	file    *file.File
	result  bytes.Buffer
	last    file.Idx
}

// NewReact returns a ready-to-use React object.
func NewReact(fset *file.FileSet, errList parser.ErrorList, file *file.File) *React {
	return &React{fset: fset, errList: errList, file: file}
}

// String returns the Javascript version of what the React visitor processed so far.
func (v *React) String() string {
	return v.result.String()
}

// TODO: refactor this and expose only something like the current `v.str()` to spare
// others from reimplementing the whole visitor when all they need is to provide a
// string given an *ElementNode.
func (v *React) Enter(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Body {
			ast.Walk(v, stmt)
		}
		src := v.file.Source()
		// print the rest
		if int(v.last) < len(src) {
			v.result.WriteString(src[v.last-1:])
		}
		return nil
	case *ast.BadExpression:
		for _, err := range v.errList {
			pos := v.fset.Position(n.From)
			// This is the hack to "identify" JSX code within Javascript.
			if pos.Column == err.Position.Column && pos.Line == err.Position.Line && strings.Contains(err.Message, "Unexpected token <") {
				src := v.file.Source()
				// Print everything from last time up until `<`, not included.
				v.result.WriteString(src[v.last : n.From-1])
				p := Parser{lexer: lex(src[n.From-1 : n.To-1])}
				if err := p.Parse(); err != nil {
					panic(err)
				}
				// Print the JSX code
				v.result.WriteString(v.str(p.root))
				v.last = n.From + file.Idx(p.lastPos)
				return v
			}
		}
	case *ast.AssignExpression:
		i, ok := n.Left.(*ast.Identifier)
		if !ok {
			break
		}
		v.ensureDisplayName(i.Name, n.Right)
	case *ast.VariableExpression:
		v.ensureDisplayName(n.Name, n.Initializer)
	}
	return v
}

func (r *React) str(n *ElementNode) string {
	var buf bytes.Buffer
	buf.WriteString("React.createElement(")
	if n.Name[0] >= 'a' && n.Name[0] <= 'z' {
		buf.WriteByte('"')
		buf.WriteString(n.Name)
		buf.WriteByte('"')
	} else if n.Name[0] >= 'A' && n.Name[0] <= 'Z' {
		buf.WriteString(n.Name)
	} else {
		panic("unexpected name: " + n.Name)
	}
	buf.WriteString(", ")
	if len(n.SpreadAttrs) > 0 {
		buf.WriteString("React.__spread({}, ")
		for _, attr := range n.SpreadAttrs {
			buf.WriteString(string(attr))
			buf.WriteString(", ")
		}
		if len(n.Attrs) > 0 {
			buf.WriteString("{")
			for k, v := range n.Attrs {
				buf.WriteString(k)
				switch v.Typ {
				case JsAttr:
					buf.WriteString(": ")
					buf.WriteString(v.Payload)
				case HtmlAttr:
					buf.WriteString(`:"`)
					buf.WriteString(v.Payload)
					buf.WriteString(`", `)
				default:
					panic(fmt.Errorf("unexpected type for %q: %v", v.Payload, v.Typ))
				}
			}
			buf.WriteString("}")
		}
		buf.WriteString(")")
	} else {
		if len(n.Attrs) == 0 {
			buf.WriteString("null")
		} else {
			buf.WriteString("{")
			for k, v := range n.Attrs {
				buf.WriteString(k)
				buf.WriteString(": ")
				switch v.Typ {
				case JsAttr:
					buf.WriteString(v.Payload)
				case HtmlAttr:
					buf.WriteString(`"`)
					buf.WriteString(v.Payload)
					buf.WriteString(`", `)
				default:
					panic("unexpected attr value type: " + v.Payload)
				}
			}
			buf.WriteString("}")
		}
	}
	if len(n.Children) > 0 {
		buf.WriteString(", ")
		for _, child := range n.Children {
			switch x := child.(type) {
			case *ElementNode:
				buf.WriteString(r.str(x))
			case TextNode:
				buf.WriteString(`"`)
				buf.WriteString(string(x))
				buf.WriteString(`"`)
			case JsNode:
				y, err := String(string(x))
				if err != nil {
					panic(err)
				}
				buf.WriteString(y)
			}
			buf.WriteString(", ")
		}
		buf.Truncate(buf.Len() - 2) // to remove the last `, `
	}
	buf.WriteString(")")
	return buf.String()
}

// Ensures that if `init` is a React.createClass() call, its first argument has a `displayName` key.
// If not, it will add it and set its value to `name`.
func (r *React) ensureDisplayName(name string, init ast.Expression) {
	c, ok := init.(*ast.CallExpression)
	if !ok {
		return
	}
	d, ok := c.Callee.(*ast.DotExpression)
	if !ok || d.Identifier.Name != "createClass" {
		return
	}
	i, ok := d.Left.(*ast.Identifier)
	if !ok || i.Name != "React" {
		return
	}
	o, ok := c.ArgumentList[0].(*ast.ObjectLiteral)
	if !ok {
		return
	}
	for _, val := range o.Value {
		if val.Key == "displayName" {
			return
		}
	}
	s := r.file.Source()[r.last:o.LeftBrace]
	r.result.WriteString(s)
	r.last += file.Idx(len(s))
	r.result.WriteString(`displayName: "` + name + `",`)
}

func (v *React) Exit(node ast.Node) {

}
