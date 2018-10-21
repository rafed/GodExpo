package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		usage()
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, args[0], nil, 0)

	if err != nil {
		os.Exit(1)
	}

	for _, n := range f.Decls {
		if fnDl, ok := n.(*ast.FuncDecl); ok {

			if fnDl.Recv.NumFields() > 0 {
				println(fnDl.Name.Name)

				typ := fnDl.Recv.List[0].Type
				nam := fnDl.Recv.List[0].Names[0].Name
				println("reciever name", nam)

				switch t := typ.(type) {
				case *ast.Ident:
					println("receiver type:", t.Name)
				case *ast.StarExpr:
					u, _ := t.X.(*ast.Ident)
					println("receiver type:*", u.Name)
				}

				if fnDl.Type.Params.NumFields() > 0 {
					params := fnDl.Type.Params.List

					for _, param := range params {
						println("\tpara:", param.Names[0].Name)

						paramType := param.Type.(ast.Expr).(*ast.Ident).Name
						// paramType = .Name
						println("\tparamtype:", paramType)
					}
				}
			}
			// if recv, ok := fnDl.(*ast.FieldList); ok {
			// 	for _, field := range recv {
			// 		fmt.Println("A:", field)
			// 	}
			// }

		}
	}

}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("gometrics file.go")
	fmt.Println("gometrics [a directory]")
}
