package main

import (
	"flag"
	"fmt"
)

func main() {

	wmc := flag.Int("wmc", 47, "Weighted method complexity")
	atfd := flag.Int("atfd", 5, "Access to foreign data")
	tcc := flag.Float64("tcc", 0.3, "Tight class cohesion")

	d := flag.String("d", "", "find god structs in a project direcotry")
	e := flag.String("e", "", "evolution of god structs with in each release")

	flag.Parse()

	WMC = *wmc
	ATFD = *atfd
	TCC = *tcc

	if (*e == "" && *d == "") || (*e != "" && *d != "") {
		flag.PrintDefaults()
	}

	// fmt.Println(flag.Args())

	// var structs []Struct

	// start := time.Now()

	// structs = analyze(args)
	// showMetrics(structs)

	// elapsed := time.Since(start)
	// fmt.Fprintf(os.Stderr, "Execution time: %s\n", elapsed)
}

func analyze(paths []string) []Struct {
	var structs []Struct
	var methods []Method

	// Find all methods and structs
	for _, path := range paths {
		newStructs, newMethods := parsePaths(path)
		structs = append(structs, newStructs...)
		methods = append(methods, newMethods...)
	}

	// Assign the methods to structs
	for _, m := range methods {
		for i, c := range structs {
			if m.PkgName == c.PkgName && m.StructName == c.StructName {
				structs[i].addMethod(m)
			}
		}
	}

	// Calculate metrics
	for i, s := range structs {
		s.WMC = calc_WMC(s)
		s.NP = NP(s)
		s.NDC = NDC(s)
		s.ATFD = calc_ATFD(s)
		s.TCC = calc_TCC(s)
		s.God = GodStruct(s)
		s.DemiGod = DemiGodStruct(s)
		structs[i] = s
	}

	return structs
}

func showMetrics(structs []Struct) {
	godCount := 0
	demiGodCount := 0

	for _, _struct := range structs {
		// fmt.Printf("Package: %s | Struct: %s\n", _struct.StructName, _struct.PkgName)
		// fmt.Printf("Position: %s\n", _struct.Pos)

		// fmt.Printf("Attributes:\n")
		// for _, a := range _struct.Attributes {
		// 	fmt.Printf("\t%s || %s\n", a.name, a.varType)
		// }

		// if len(_struct.Methods) > 0 {
		// 	fmt.Printf("Methods:\n")
		// 	for _, m := range _struct.Methods {
		// 		// fmt.Print("ALL methods and its complexities: ")
		// 		fmt.Printf("\t%-20v | Complexity: %d\n", m.FuncName+"()", m.Complexity)

		// 		// fmt.Print("ALL accesed Variables: ")
		// 		// for _, v := range m.Selectors {
		// 		// 	fmt.Printf("%s.%s ", v.left, v.right)
		// 		// }
		// 		// fmt.Println()

		// 		// fmt.Print("Accessed own: ")
		// 		// for _, v := range m.SelfVarAccessed {
		// 		// 	fmt.Printf("%s.%s ", v.left, v.right)
		// 		// }
		// 		// fmt.Println()

		// 		// fmt.Print("Accessed others: ")
		// 		// for _, v := range m.OthersVarAccessed {
		// 		// 	fmt.Printf("%s.%s ", v.left, v.right)
		// 		// }
		// 		// fmt.Println()
		// 	}
		// }

		// fmt.Printf("\tWMC: %d\n", _struct.WMC)
		// fmt.Printf("\tNDC: %d\n", _struct.NDC)
		// fmt.Printf("\tNP: %d\n", _struct.NP)
		// fmt.Printf("\tATFD: %d\n", _struct.ATFD)

		// if _struct.TCC == TCC_Null {
		// 	fmt.Printf("\tTCC: --\n")
		// } else {
		// 	fmt.Printf("\tTCC: %f\n", _struct.TCC)
		// }

		classificationString := "none"

		if _struct.God == true {
			godCount++
			classificationString = "god"
		} else if _struct.DemiGod == true {
			continue
			demiGodCount++
			classificationString = "demigod"
		}

		if classificationString == "none" {
			continue
		}

		fmt.Printf("[%s] %s: %s\n", _struct.PkgName, _struct.StructName, classificationString)
		fmt.Printf("Position: %s\n", _struct.Pos)

		fmt.Printf("\tWMC: %d\n", _struct.WMC)
		fmt.Printf("\tNDC: %d\n", _struct.NDC)
		fmt.Printf("\tNP: %d\n", _struct.NP)
		fmt.Printf("\tATFD: %d\n", _struct.ATFD)

		if _struct.TCC == TCC_Null {
			fmt.Printf("\tTCC: --\n")
		} else {
			fmt.Printf("\tTCC: %f\n", _struct.TCC)
		}
	}

	fmt.Println()
	fmt.Println("Num of structs:", len(structs))

	fmt.Println("God structs:", godCount)
	fmt.Printf("God percentage: %f\n", float32(godCount)/float32(len(structs)))

	fmt.Println("DemiGod structs:", demiGodCount)
	fmt.Printf("DemiGod percentage: %f\n", float32(demiGodCount)/float32(len(structs)))
}
