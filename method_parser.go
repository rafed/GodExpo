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
	AllVars        []variable
	LocalVars      []variable
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

			v := variableVisitor{}

			if fn.Recv == nil {
				continue
			}
			rcv := variable{
				name:    fn.Recv.List[0].Names[0].Name,
				varType: recvString(fn.Recv.List[0].Type),
			}

			var params []variable
			for _, l := range fn.Type.Params.List {
				temp := variable{
					name:    l.Names[0].Name,
					varType: recvString(l.Type),
				}
				params = append(params, temp)
			}

			ast.Walk(&v, fn.Body)

			fmt.Print("*** ")
			fmt.Println(funcName(fn))
			for _, n := range v.variables {
				fmt.Printf("%s (%s): %d\n", n.name, n.varType, n.count)
			}
			fmt.Println()

			methods = append(methods, method{
				PkgName:    f.Name.Name,
				FuncName:   funcName(fn),
				Receiver:   rcv,
				Parameters: params,
				AllVars:    v.variables,
				Complexity: complexity(fn),
				Pos:        fset.Position(fn.Pos()),
			})

		}

	}

	return methods
}

type variableVisitor struct {
	variables []variable
}

func (v *variableVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return v
	}

	v.findVariables(n)
	return v

}

func (v *variableVisitor) findVariables(n ast.Node) {
	if n == nil {
		return
	}

	ident, ok := n.(*ast.Ident)
	if !ok {
		return
	}
	if ident.Name == "_" || ident.Name == "" {
		return
	}

	v.add(ident)
}

func (v *variableVisitor) exists(name string) bool {
	for _, n := range v.variables {
		if n.name == name {
			return true
		}
	}

	return false
}

func (v *variableVisitor) increment(name string) {
	for i, n := range v.variables {
		if n.name == name {
			v.variables[i].count++
			break
		}
	}
}

func (v *variableVisitor) addNew(name string, varType string) {
	v.variables = append(v.variables, variable{
		name:    name,
		varType: varType,
		count:   1,
	})
}

func (v *variableVisitor) add(ident *ast.Ident) {
	name := ident.Name

	if ident.Obj == nil {
		// println("other vars: ", name)
		return
	}

	varType := ident.Obj.Kind.String()

	// println(name, "*************", ident.Obj.Kind)
	// if ident.Obj.Kind == 4 {
	// 	varType = ident.Obj.Name
	// } else {
	// 	varType = "yo"
	// }

	if v.exists(name) {
		v.increment(name)
	} else {
		v.addNew(name, varType)
	}
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

func complexity(fn *ast.FuncDecl) int {
	v := complexityVisitor{}
	ast.Walk(&v, fn)
	return v.Complexity
}

type complexityVisitor struct {
	// Complexity is the cyclomatic complexity
	Complexity int
}

func (v *complexityVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.FuncDecl, *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CaseClause, *ast.CommClause:
		v.Complexity++
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			v.Complexity++
		}
	}
	return v
}
