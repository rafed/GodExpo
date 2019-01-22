package main

import (
	"fmt"
	"sort"
)

type PackageSorter []Struct

func (a PackageSorter) Len() int      { return len(a) }
func (a PackageSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a PackageSorter) Less(i, j int) bool {
	if a[i].PkgName < a[j].PkgName {
		return true
	}
	if a[i].PkgName > a[j].PkgName {
		return false
	}
	return a[i].StructName < a[j].StructName
}

func structNameLocation(s Struct) {
	fmt.Printf("Package: %s\n", s.PkgName)
	fmt.Printf("Struct: %s\n", s.StructName)
	fmt.Printf("Location: %s\n", s.Pos)
}

func viewFileMetrics(structs []Struct) {
	sort.Sort(PackageSorter(structs))

	for _, _struct := range structs {
		structNameLocation(_struct)

		// fmt.Printf("Attributes:\n")
		// for _, a := range _struct.Attributes {
		// 	fmt.Printf("\t%s || %s\n", a.name, a.varType)
		// }

		if len(_struct.Methods) > 0 {
			fmt.Printf("Methods:\n")
			for _, m := range _struct.Methods {
				fmt.Printf("\t%-20v | Complexity: %d | Accessed self: %d | Accessed others: %d\n", m.FuncName+"()", m.Complexity, len(m.SelfVarAccessed), len(m.OthersVarAccessed))
			}
		}
		fmt.Println()
	}
}

func viewProjectMetrics(structs []Struct) {
	sort.Sort(PackageSorter(structs))

	godCounter := 0

	for _, _struct := range structs {
		if _struct.God != true {
			continue
		}
		godCounter++

		structNameLocation(_struct)
		fmt.Printf("\tWMC: %d\n", _struct.WMC)
		fmt.Printf("\tATFD: %d\n", _struct.ATFD)

		if _struct.TCC == TCC_Null {
			fmt.Printf("\tTCC: --\n")
		} else {
			fmt.Printf("\tTCC: %f\n", _struct.TCC)
		}

		fmt.Println()
	}

	fmt.Printf("[*] Total %d God Structs found.\n", godCounter)
}

func viewEvolutionMetrics(structs []Struct) {
	for _, _struct := range structs {
		structNameLocation(_struct)
	}
}

func showMetrics(structs []Struct) {
	// godCount := 0
	// demiGodCount := 0

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

		// classificationString := "none"

		// if _struct.God == true {
		// 	godCount++
		// 	classificationString = "god"
		// } else if _struct.DemiGod == true {
		// 	// continue
		// 	demiGodCount++
		// 	classificationString = "demigod"
		// }

		// if classificationString == "none" {
		// 	continue
		// }

		fmt.Printf("\tWMC: %d\n", _struct.WMC)
		// fmt.Printf("\tNDC: %d\n", _struct.NDC)
		// fmt.Printf("\tNP: %d\n", _struct.NP)
		fmt.Printf("\tATFD: %d\n", _struct.ATFD)

		if _struct.TCC == TCC_Null {
			fmt.Printf("\tTCC: --\n")
		} else {
			fmt.Printf("\tTCC: %f\n", _struct.TCC)
		}
	}

	// fmt.Println()
	// fmt.Println("Num of structs:", len(structs))

	// fmt.Println("God structs:", godCount)
	// fmt.Printf("God percentage: %f\n", float32(godCount)/float32(len(structs)))
}
