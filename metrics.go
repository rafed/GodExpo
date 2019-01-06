package main

func WMC(c class) int {
	wmc := 0
	for _, m := range c.Methods {
		wmc += m.Complexity
	}
	return wmc
}

func NP(c class) int {
	n := len(c.Methods)
	if n <= 1 {
		return 0
	}
	return n * (n - 1) / 2
}

func NDC(c class) int {
	ndc := 0
	for j := 0; j < len(c.Methods)-1; j++ {
		for k := j + 1; k < len(c.Methods); k++ {
			if commonAttributeAccessExists(c.Methods[j], c.Methods[k]) {
				ndc++
			}
		}
	}
	return ndc
}

func commonAttributeAccessExists(m1 method, m2 method) bool {
	for _, v1 := range m1.SelfVarAccessed {
		for _, v2 := range m2.SelfVarAccessed {
			if v1.right == v2.right {
				return true
			}
		}
	}

	return false
}

func ATFD(c class) int {
	uniqList := uniqeSelectors{}

	for _, m := range c.Methods {
		for _, v := range m.OthersVarAccessed {
			if !uniqList.exists(v) {
				uniqList.add(v)
			}
		}
	}

	atfd := len(uniqList.selectors)
	return atfd
}

func TCC(c class) float32 {
	if c.NP == 0 {
		return 99999
	}
	return float32(c.NDC) / float32(c.NP)
}

func GodStruct(c class) bool {
	if c.WMC > 47 && c.TCC < 0.3 && c.ATFD > 5 {
		return true
	}

	return false
}

func DemiGodStruct(c class) bool {
	if GodStruct(c) {
		return false
	}

	demiGodEligibleCounter := 0

	if c.WMC > 47 {
		demiGodEligibleCounter++
	}

	if c.TCC < 0.3 {
		demiGodEligibleCounter++
	}

	if c.ATFD > 5 {
		demiGodEligibleCounter++
	}

	if demiGodEligibleCounter == 2 {
		return true
	}

	return false
}
