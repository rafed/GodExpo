package main

import "go/ast"

type allVariableVisitor struct {
	variables []variable
}

func (v *allVariableVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return v
	}

	v.findAllVariables(n)
	return v

}

func (v *allVariableVisitor) findAllVariables(n ast.Node) {
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

func (v *allVariableVisitor) exists(name string) bool {
	for _, n := range v.variables {
		if n.name == name {
			return true
		}
	}

	return false
}

func (v *allVariableVisitor) increment(name string) {
	for i, n := range v.variables {
		if n.name == name {
			v.variables[i].count++
			break
		}
	}
}

func (v *allVariableVisitor) addNew(name string, varType string) {
	v.variables = append(v.variables, variable{
		name:    name,
		varType: varType,
		count:   1,
	})
}

func (v *allVariableVisitor) add(ident *ast.Ident) {
	name := ident.Name

	if ident.Obj == nil {
		println("other vars: ", name)
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
