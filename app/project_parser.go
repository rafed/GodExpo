package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func parsePath(file string) ([]Struct, []Method) {
	var structs []Struct
	var methods []Method

	if isDir(file) {
		filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
				fset := token.NewFileSet()
				f, err := parser.ParseFile(fset, path, nil, 0)

				if err != nil {
					log.Fatal(err)
				}

				fmt.Fprintf(os.Stderr, "[*] Analyzing: %s\n", path)

				structs = append(structs, findStructsFromFile(fset, f)...)
				methods = append(methods, findMethodsFromFile(fset, f, path)...)
			}
			return err
		})

		fmt.Println()
	} else {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)

		if err != nil {
			log.Fatal(err)
		}

		structs = findStructsFromFile(fset, f)
		methods = findMethodsFromFile(fset, f, file)
	}

	return structs, methods
}
