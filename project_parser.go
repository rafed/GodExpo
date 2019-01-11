package main

import (
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func parsePaths(file string) ([]Struct, []Method) {
	var structs []Struct
	var methods []Method

	println("wadafaq")

	if isDir(file) {
		println("faqala", file)
		filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
				fset := token.NewFileSet()
				f, err := parser.ParseFile(fset, path, nil, 0)

				if err != nil {
					log.Fatal(err)
				}

				println("Analyzing:", path)

				structs = append(structs, findStructsFromFile(fset, f)...)
				methods = append(methods, findMethodsFromFile(fset, f, path)...)
			}
			return err
		})
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
