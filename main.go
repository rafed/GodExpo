package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func main() {

	wmc := flag.Int("wmc", 47, "Weighted method complexity")
	atfd := flag.Int("atfd", 5, "Access to foreign data")
	tcc := flag.Float64("tcc", 0.3, "Tight class cohesion")

	f := flag.String("f", "", "show metrics of a file")
	d := flag.String("d", "", "find god structs in a project direcotry")
	e := flag.String("e", "", "evolution of god structs with each release")

	flag.Parse()

	WMC = *wmc
	ATFD = *atfd
	TCC = *tcc

	argsProvided := 0

	if *f != "" {
		if isDir(*f) {
			fmt.Fprintf(os.Stderr, "Provide a file, Usage:\n")
			flag.PrintDefaults()
			os.Exit(1)
		}

		argsProvided++
	}
	if *d != "" {
		if !isDir(*d) {
			fmt.Fprintf(os.Stderr, "Provide a directory, Usage:\n")
			flag.PrintDefaults()
			os.Exit(1)
		}

		argsProvided++
	}
	if *e != "" {
		if !isDir(*e) {
			fmt.Fprintf(os.Stderr, "Provide a directory, Usage:\n")
			flag.PrintDefaults()
			os.Exit(1)
		}

		argsProvided++
	}

	if argsProvided > 1 || argsProvided == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// FLAG STUFF DONE

	var structs []Struct

	start := time.Now()

	if *f != "" {
		structs = analyze(*f)
		viewFileMetrics(structs)
	} else if *d != "" {
		structs = analyze(*d)
		viewProjectMetrics(structs)
	} else if *e != "" {
		dirs, err := ioutil.ReadDir(*e)
		if err != nil {
			log.Fatal(err)
		}

		releases := 0
		var versions []string

		for _, d := range dirs {
			if d.IsDir() {
				versions = append(versions, filepath.Join(*e, d.Name()))
				releases++
			}
		}

		table := map[string]map[string]string{}

		for _, r := range versions {
			structs := analyze(r)

			for _, s := range structs {
				if s.God {
					structIdentifier := fmt.Sprintf("[%s] %s", s.PkgName, s.StructName)

					if _, ok := table[structIdentifier]; !ok {
						table[structIdentifier] = map[string]string{}
					}

					table[structIdentifier][r] = fmt.Sprintf("%d,%d,%.2f", s.WMC, s.ATFD, s.TCC)
				}
			}
		}

		// Get all god structs as keys
		var keys []string
		for key := range table {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			fmt.Printf("%-20v ", key)
			for _, v := range versions {
				fmt.Printf("%s ", table[key][v])
			}
			fmt.Println()
		}
	}

	fmt.Fprintf(os.Stderr, "Execution time: %s\n", time.Since(start))
}

func analyze(path string) []Struct {
	// var structs []Struct
	// var methods []Method

	// // Find all methods and structs
	// for _, path := range paths {
	// 	newStructs, newMethods := parsePaths(path)
	// 	structs = append(structs, newStructs...)
	// 	methods = append(methods, newMethods...)
	// }

	structs, methods := parsePath(path)

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
