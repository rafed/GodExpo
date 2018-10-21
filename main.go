package main

import (
	"fmt"
	"go/token"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		usage()
	}

	analyze(args)
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("gometrics file.go")
	fmt.Println("gometrics [a directory]")
}

type attribute struct {
	name    string
	varType string
}

type class struct {
	PkgName    string
	StructName string
	Attributes []attribute
	Methods    []method
	Pos        token.Position
}

type method struct {
	PkgName    string
	StructName string
	FuncName   string
	Complexity int
	VarStruct  int
	VarOutside int
	Pos        token.Position
}

func analyze(paths []string) {
	var classes []class

	for _, path := range paths {
		if isDir(path) {
			classes = classAnalyzeDir(path, classes)
		} else {
			classes = classAnalyzeFile(path, classes)
		}
	}

	for _, class := range classes {
		fmt.Printf("Position: %s\n", class.Pos)
		fmt.Printf("Package: %s\n", class.PkgName)
		fmt.Printf("\tName: %s\n", class.StructName)

		for _, a := range class.Attributes {
			fmt.Printf("\t\tatttribute: %s || %s\n", a.name, a.varType)
		}
	}

}
