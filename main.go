package main

import (
	"fmt"
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

func analyze(paths []string) {
	var classes []class
	var methods []method

	for _, path := range paths {
		if isDir(path) {
			classes = classAnalyzeDir(path, classes)
		} else {
			classes = classAnalyzeFile(path, classes)
		}
	}

	for _, path := range paths {
		if isDir(path) {
			methods = methodAnalyzeDir(path, methods)
		} else {
			methods = methodAnalyzeFile(path, methods)
		}
	}

	// for _, class := range classes {
	// 	fmt.Printf("Position: %s\n", class.Pos)
	// 	fmt.Printf("Package: %s\n", class.PkgName)
	// 	fmt.Printf("\tName: %s\n", class.StructName)

	// 	for _, a := range class.Attributes {
	// 		fmt.Printf("\t\tatttribute: %s || %s\n", a.name, a.varType)
	// 	}
	// }

	// for _, method := range methods {
	// 	fmt.Printf("Position: %s\n", method.Pos)
	// 	// fmt.Printf("Package: %s\n", method.PkgName)
	// 	fmt.Printf("\tName: %s\n", method.FuncName)
	// 	// fmt.Printf("\tComplexity: %d\n", method.Complexity)

	// 	for _, a := range method.OwnVars {
	// 		fmt.Printf("\t\tvar: %s || %s\n", a.name, a.varType)
	// 	}
	// }

}
