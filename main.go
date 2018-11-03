package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		usage()
	}

	start := time.Now()
	analyze(args)

	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "Execution time: %s\n", elapsed)
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("gometrics [file.go]")
	fmt.Println("gometrics [directory]")
}

func analyze(paths []string) {
	var classes []class
	var methods []method

	// Find structs (we're calling them class here) and methods
	for _, path := range paths {
		if isDir(path) {
			classes = classAnalyzeDir(path, classes)
			methods = methodAnalyzeDir(path, methods)
		} else {
			classes = classAnalyzeFile(path, classes)
			methods = methodAnalyzeFile(path, methods)
		}
	}

	// Separate all selectors
	for i := range methods {
		methods[i].separateAccessedVars()
	}

	// Assign the methods to structs
	for _, m := range methods {
		for i, c := range classes {
			if m.PkgName == c.PkgName && m.StructName == c.StructName {
				classes[i].addMethod(m)
			}

		}
	}

	// Finding the metrics
	for i, c := range classes {
		c.WMC = WMC(c)
		c.NP = NP(c)
		c.NDC = NDC(c)
		c.ATFD = ATFD(c)
		c.TCC = TCC(c)
		c.God = GodStruct(c)
		classes[i] = c
	}

	godCount := 0

	for _, class := range classes {
		fmt.Printf("Package: %s\n", class.PkgName)
		fmt.Printf("StructName: %s\n", class.StructName)
		fmt.Printf("Position: %s\n", class.Pos)

		fmt.Printf("Attributes:\n")
		for _, a := range class.Attributes {
			fmt.Printf("\t%s || %s\n", a.name, a.varType)
		}

		if len(class.Methods) > 0 {
			fmt.Printf("Methods:\n")
			for _, m := range class.Methods {
				fmt.Printf("\tFuncName: %s()\n", m.FuncName)
				fmt.Printf("\t\tComplexity: %d\n", m.Complexity)

				// fmt.Print("ALL Variables: ")
				// for _, v := range m.Selectors {
				// 	fmt.Printf("%s.%s ", v.left, v.right)
				// }
				// fmt.Println()

				// fmt.Print("Accessed own: ")
				// for _, v := range m.SelfVarAccessed {
				// 	fmt.Printf("%s.%s ", v.left, v.right)
				// }
				// fmt.Println()

				// fmt.Print("Accessed others: ")
				// for _, v := range m.OthersVarAccessed {
				// 	fmt.Printf("%s.%s ", v.left, v.right)
				// }
				// fmt.Println()
			}
		}

		fmt.Println("Metrics:")
		fmt.Printf("\tWMC: %d\n", class.WMC)
		fmt.Printf("\tNDC: %d\n", class.NDC)
		fmt.Printf("\tNP: %d\n", class.NP)
		fmt.Printf("\tATFD: %d\n", class.ATFD)
		if class.TCC == 99999 {
			fmt.Printf("\tTCC: --\n")
		} else {
			fmt.Printf("\tTCC: %f\n", class.TCC)
		}
		fmt.Printf("\tGod: %v\n", class.God)

		if class.God == true {
			godCount++
		}

		fmt.Println()
	}

	fmt.Println(paths[0])
	fmt.Println("Num of structs:", len(classes))
	fmt.Println("God structs:", godCount)
	fmt.Printf("God percentage: %f\n", float32(godCount)/float32(len(classes)))
}
