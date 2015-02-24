package jsx

import (
	"fmt"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/parser"
)

func File(filename string) (string, error) {
	fset := new(file.FileSet)
	program, err := parser.ParseFile(fset, filename, nil, 0)
	return eval(fset, program, err)
}

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

	v := &React{fset: fset, errList: errList, file: program.File}
	ast.Walk(v, program)
	return v.result.String(), nil
}
