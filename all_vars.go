package main

import (
	"go/ast"
	"go/token"
)

type selector struct {
	left  string
	right string
	pos   token.Pos
	line  string
}

type allSelectorVisitor struct {
	selectors []selector
}

func (v *allSelectorVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return v
	}

	if selectorExp, ok := n.(*ast.SelectorExpr); ok {
		if va, ok := selectorExp.X.(*ast.Ident); ok {
			if va.Obj == nil {
				return v
			}

			if va.Obj.Kind.String() == "var" {
				newSelector := selector{
					left:  va.Name,
					right: selectorExp.Sel.Name,
					pos:   va.Pos(),
				}
				v.add(newSelector)
			}
		}
	}
	return v
}

func (v *allSelectorVisitor) add(s selector) {
	if !v.exists(s.left) {
		v.selectors = append(v.selectors, s)
	}
}

func (v *allSelectorVisitor) exists(name string) bool {
	for _, n := range v.selectors {
		if n.left == name {
			return true
		}
	}
	return false
}
