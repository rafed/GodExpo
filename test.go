package main

import (
	"bufio"
	"go/ast"
	"go/token"
	"os"
	"strings"
)

func main() {
	// fset := token.NewFileSet()

	// file := "utils.go"
	// f, err := parser.ParseFile(fset, file, nil, 0)

	// if err != nil {
	// 	log.Fatal(err)
	// }

	// v := visitor{}

	// for _, decl := range f.Decls {
	// 	if fn, ok := decl.(*ast.FuncDecl); ok {
	// 		ast.Walk(&v, fn.Body)
	// 	}
	// }

	// for i := range v.left {
	// 	println(v.left[i])
	// 	println(v.right[i])
	// 	l := line(file, fset.Position(v.pos[i]).Line)
	// 	println(l)
	// 	println()
	// }

	s := "m.m"
	println(isVariable(s, "m", "m"))
}

func isVariable(line string, leftVar string, rightVar string) bool {
	for len(line) > len(leftVar) {
		pos := strings.Index(line, leftVar)

		if pos == -1 {
			break
		}

		if line[pos+len(leftVar)] == '.' {
			line2 := line[pos+len(leftVar)+1:]

			if pos2 := strings.Index(line2, rightVar); pos2 == 0 {
				if len(line2) == len(rightVar) {
					return true
				}

				line3 := line2[len(rightVar)+1:]
				if line3[0] == ' ' {
					return true
				} else {
					line = line[pos+1:]
					continue
				}
			}
		}

		line = line[pos+1:]
	}

	return false
}

type visitor struct {
	left  []string
	right []string
	pos   []token.Pos
}

func (v *visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return v
	}

	if selectorExp, ok := n.(*ast.SelectorExpr); ok {
		if va, ok := selectorExp.X.(*ast.Ident); ok {
			if va.Obj == nil {
				return v
			}

			if va.Obj.Kind.String() == "var" {
				v.left = append(v.left, va.Name)
				v.right = append(v.right, selectorExp.Sel.Name)
				v.pos = append(v.pos, va.Pos())
			}
		}
	}

	return v

}

func line(file string, pos int) string {
	f, err := os.Open(file)
	defer f.Close()

	if err != nil {
		return "ERROR"
	}

	bf := bufio.NewReader(f)
	var line string
	for lnum := 0; lnum < pos; lnum++ {
		line, err = bf.ReadString('\n')
		if err != nil {
			return "ERROR"
		}
	}
	return line
}
