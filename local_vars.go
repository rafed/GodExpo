package main

import "go/ast"

type localVariableVisitor struct {
	variables []variable
}

func (v *localVariableVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return v
	}

	switch d := n.(type) {
	case *ast.AssignStmt:
		for _, name := range d.Lhs {
			v.findLocalVariables(name)

			// if ident, ok := name.(*ast.Ident); ok {
			// }
		}
	}

	return v
}

func (v *localVariableVisitor) findLocalVariables(n ast.Node) {
	ident, ok := n.(*ast.Ident)

	if !ok {
		return
	}

	if ident.Name == "_" || ident.Name == "" {
		return
	}
	if ident.Obj != nil && ident.Obj.Pos() == ident.Pos() {
		v.add(ident)
	}

}

func (v *localVariableVisitor) add(ident *ast.Ident) {
	name := ident.Name

	if ident.Obj == nil {
		return
	}

	varType := ident.Obj.Kind.String()

	if v.exists(name) {
		v.increment(name)
	} else {
		v.addNew(name, varType)
	}
}

func (v *localVariableVisitor) exists(name string) bool {
	for _, n := range v.variables {
		if n.name == name {
			return true
		}
	}

	return false
}

func (v *localVariableVisitor) increment(name string) {
	for i, n := range v.variables {
		if n.name == name {
			v.variables[i].count++
			break
		}
	}
}

func (v *localVariableVisitor) addNew(name string, varType string) {
	v.variables = append(v.variables, variable{
		name:    name,
		varType: varType,
		count:   1,
	})
}
