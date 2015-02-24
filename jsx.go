// Jsx transpiles React JSX code to Javascript.
//
// This package also provides primitives to write your own visitors
package jsx

import (
	"fmt"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/parser"
)

// File takes the name of a JSX file as input and returns the Javascript version of the content.
func File(filename string) (string, error) {
	fset := new(file.FileSet)
	program, err := parser.ParseFile(fset, filename, nil, 0)
	return eval(fset, program, err)
}

// String takes JSX source code as input and returns Javascript.
func String(src string) (string, error) {
	fset := new(file.FileSet)
	program, err := parser.ParseFile(fset, "", src, 0)
	return eval(fset, program, err)
}

func eval(fset *file.FileSet, program *ast.Program, err error) (string, error) {
	if err == nil {
		return program.File.Source(), nil
	}

	errList, ok := err.(parser.ErrorList)
	if !ok {
		return "", fmt.Errorf("unexpected error: %v", err)
	}

	r := NewReact(fset, errList, program.File)
	Walk(r, program)
	return r.String(), nil
}
