package main

import (
	"go/ast"
	"go/token"
)

type Struct struct {
	PkgName    string
	StructName string
	Attributes []variable
	Methods    []Method
	Pos        token.Position
	WMC        int
	NDC        int
	NP         int
	ATFD       int
	TCC        float64
	God        bool
	DemiGod    bool
}

func (c *Struct) addMethod(m Method) {
	c.Methods = append(c.Methods, m)
}

func findStructsFromFile(fset *token.FileSet, f *ast.File) []Struct {
	var structs []Struct

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
				// println("zzzzzzzzzzzzz:", recvString(i.Type))
				// switch t := i.Type.(type) {
				// case *ast.Ident:
				// 	println("doom ", t.Name)
				// 	println("doom ", t.Obj.Name)
				// case *ast.StarExpr:
				// 	println("doom2 *" + recvString(t.X))
				// case *ast.StructType:
				// 	println("doom3 ", t.Name)
				// }

				continue
			}

			for j := 0; j < len(i.Names); j++ {
				a := variable{
					name:    i.Names[j].Name,
					varType: recvString(i.Type),
				}

				attributes = append(attributes, a)
			}
		}

		c := Struct{
			PkgName:    pkgName,
			StructName: structName,
			Attributes: attributes,
			Pos:        fset.Position(t.Pos()),
		}

		structs = append(structs, c)
		return true
	}

	ast.Inspect(f, findStructs)

	return structs
}
