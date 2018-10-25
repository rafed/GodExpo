package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type method struct {
	PkgName        string
	StructName     string
	FuncName       string
	Complexity     int
	Receiver       variable
	Parameters     []variable
	Selectors      []selector
	AccessedOwn    int
	AccessedOthers int
	Pos            token.Position
}

func methodAnalyzeDir(dirname string, methods []method) []method {
	filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
			methods = methodAnalyzeFile(path, methods)
		}
		return err
	})
	return methods
}

func methodAnalyzeFile(fname string, methods []method) []method {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, nil, 0)

	if err != nil {
		log.Fatal(err)
	}

	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {

			if fn.Recv == nil || fn.Recv.List[0].Names == nil {
				continue
			}

			// Get receivers
			rcv := variable{
				name:    fn.Recv.List[0].Names[0].Name,
				varType: recvString(fn.Recv.List[0].Type),
			}

			// Get parameters
			var params []variable
			for _, l := range fn.Type.Params.List {
				temp := variable{
					name:    l.Names[0].Name,
					varType: recvString(l.Type),
				}
				params = append(params, temp)
			}

			// Get selectors
			varAll := allSelectorVisitor{}
			ast.Walk(&varAll, fn.Body)

			for i, n := range varAll.selectors {
				varAll.selectors[i].line = findLine(fname, fset.Position(n.pos).Line)
			}

			accessOwn, accessOthers := countAccess(rcv.name, varAll.selectors)

			methods = append(methods, method{
				PkgName:        f.Name.Name,
				FuncName:       funcName(fn),
				Receiver:       rcv,
				Parameters:     params,
				Selectors:      varAll.selectors,
				Complexity:     complexity(fn),
				AccessedOwn:    accessOwn,
				AccessedOthers: accessOthers,
				Pos:            fset.Position(fn.Pos()),
			})
		}
	}
	return methods
}

func countAccess(recv string, selectors []selector) (int, int) {
	accessOwn := 0
	accessOthers := 0

	for _, s := range selectors {
		if !isVariable(s.line, s.left, s.right) {
			break
		}

		if s.left == recv {
			accessOwn++
		} else {
			accessOthers++
		}
	}

	return accessOwn, accessOthers
}

func funcName(fn *ast.FuncDecl) string {
	if fn.Recv != nil {
		if fn.Recv.NumFields() > 0 {
			typ := fn.Recv.List[0].Type
			return fmt.Sprintf("(%s).%s", recvString(typ), fn.Name)
		}
	}
	return fn.Name.Name
}
