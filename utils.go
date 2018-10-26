package main

import (
	"bufio"
	"go/ast"
	"os"
	"strings"
)

type variable struct {
	name    string
	varType string
	count   int
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func recvString(recv ast.Expr) string {
	switch t := recv.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + recvString(t.X)
	}
	return "BADRECV"
}

func recvOnlyNameString(recv ast.Expr) string {
	switch t := recv.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return recvOnlyNameString(t.X)
	}
	return "BADRECV"
}

func isAlphaNumeric(c byte) bool {
	s := "123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := []byte(s)

	for _, x := range b {
		if x == c {
			return true
		}
	}

	return false
}

func findLine(file string, pos int) string {
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

	return strings.TrimSpace(line)
}

func isVariable(line string, leftVar string, rightVar string) bool {
	for len(line) > len(leftVar)+len(rightVar) {
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

				line3 := line2[len(rightVar):]

				if line3[0] == '(' {
					return false
				} else if !isAlphaNumeric(line3[0]) {
					return true
				} else {
					return true
				}
			}
		}

		if pos = strings.Index(line, " "); pos == -1 {
			break
		}

		line = line[pos+1:]
	}

	return false
}

type uniqeSelectors struct {
	selectors []selector
}

func (u *uniqeSelectors) exists(s selector) bool {
	for _, v := range u.selectors {
		// println("WKAALAKLKA", v.toString(), s.toString)
		if v.toString() == s.toString() {
			return true
		}
	}
	return false
}

func (u *uniqeSelectors) add(s selector) {
	u.selectors = append(u.selectors, s)
}
