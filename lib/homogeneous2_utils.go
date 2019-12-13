package lib

func sorted(a, b, c portalData) (portalData, portalData, portalData) {
	if a.Index < b.Index {
		if a.Index < c.Index {
			if b.Index < c.Index {
				return a, b, c
			}
			return a, c, b
		}
		return c, a, b
	}
	if a.Index < c.Index {
		return b, a, c
	}
	if b.Index < c.Index {
		return b, c, a
	}
	return c, b, a
}

func sortedIndices(a, b, c portalIndex) (portalIndex, portalIndex, portalIndex) {
	if a < b {
		if a < c {
			if b < c {
				return a, b, c
			}
			return a, c, b
		}
		return c, a, b
	}
	if a < c {
		return b, a, c
	}
	if b < c {
		return b, c, a
	}
	return c, b, a
}

func ordering(p0, p1, p2 portalData, index int) (portalData, portalData, portalData) {
	switch index {
	case 2:
		return p0, p1, p2
	case 3:
		return p0, p2, p1
	case 4:
		return p1, p0, p2
	case 5:
		return p1, p2, p0
	case 6:
		return p2, p0, p1
	default:
		return p2, p1, p0
	}
}
func indexOrdering(p0, p1, p2 portalIndex, index int) (portalIndex, portalIndex, portalIndex) {
	switch index {
	case 2:
		return p0, p1, p2
	case 3:
		return p0, p2, p1
	case 4:
		return p1, p0, p2
	case 5:
		return p1, p2, p0
	case 6:
		return p2, p0, p1
	default:
		return p2, p1, p0
	}
}
