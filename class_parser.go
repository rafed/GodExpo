package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type class struct {
	PkgName    string
	StructName string
	Attributes []variable
	Methods    []method
	Pos        token.Position
}

func classAnalyzeDir(dirname string, classes []class) []class {
	filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
			classes = classAnalyzeFile(path, classes)
		}
		return err
	})
	return classes
}

func classAnalyzeFile(fname string, classes []class) []class {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, nil, 0)

	if err != nil {
		log.Fatal(err)
	}

	pkgName := f.Name.Name

	findStructs := func(n ast.Node) bool {
		t, ok := n.(*ast.TypeSpec)

		if !ok || t.Type == nil {
			return true
		}

		structName := t.Name.Name
		var attributes []variable

		x, ok := t.Type.(*ast.StructType)
		if !ok {
			return true
		}

		for _, i := range x.Fields.List {
			if i.Names == nil {
				continue
			}

			a := variable{
				name:    i.Names[0].Name,
				varType: recvString(i.Type),
			}
			attributes = append(attributes, a)
		}

		c := class{
			PkgName:    pkgName,
			StructName: structName,
			Attributes: attributes,
			Pos:        fset.Position(t.Pos()),
		}

		classes = append(classes, c)
		return true
	}

	ast.Inspect(f, findStructs)

	return classes
}
