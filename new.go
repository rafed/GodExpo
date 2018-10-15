package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
)

// type File struct {
//     Doc        *CommentGroup   // associated documentation; or nil
//     Package    token.Pos       // position of "package" keyword
//     Name       *Ident          // package name
//     Decls      []Decl          // top-level declarations; or nil
//     Scope      *Scope          // package scope (this file only)
//     Imports    []*ImportSpec   // imports in this file
//     Unresolved []*Ident        // unresolved identifiers in this file
//     Comments   []*CommentGroup // list of all comments in the source file
// }

// https://medium.com/justforfunc/understanding-go-programs-with-go-parser-c4e88a6edb87
// https://zupzup.org/go-ast-traversal/
// https://github.com/fzipp/gocyclo/blob/master/gocyclo.go

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "try.go", nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Imports:")
	for _, i := range node.Imports {
		fmt.Println(i.Path.Value)
	}

	println()

	fmt.Println("Comments:")
	for _, c := range node.Comments {
		fmt.Print(c.Text())
	}

	println()

	fmt.Println("Functions:")
	for _, f := range node.Decls {
		fn, ok := f.(*ast.FuncDecl)
		if !ok {
			continue
		}
		fmt.Println(fn.Name.Name)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		// Find Return Statements
		ret, ok := n.(*ast.ReturnStmt)
		if ok {
			fmt.Printf("return statement found on line %d: ", fset.Position(ret.Pos()).Line)
			printer.Fprint(os.Stdout, fset, ret)
			println()
			return true
		}
		return true
	})

}
