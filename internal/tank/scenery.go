package tank

// placeShells lays the shells on the sand for the Tank's layout: one about
// a third of the way across and one about three quarters, each nudged to the
// nearest column clear of the seaweed. They stay put until the next resize.
func (t *Tank) placeShells() {
	t.shells = t.shells[:0]
	for _, want := range []int{t.width/3 + 4, t.width*3/4 + 2} {
		for d := 0; d < 2*t.width; d++ {
			x := want + (d+1)/2*(1-2*(d%2)) // want, want+1, want-1, want+2, ...
			if t.shellFits(x) {
				t.shells = append(t.shells, x)
				break
			}
		}
	}
}

// shellFits reports whether a shell can rest at column x: inside the Tank,
// on a column of its own, and ShellClearance clear of the SeaweedWidth
// columns every stalk stands and sways in.
func (t *Tank) shellFits(x int) bool {
	if x < 0 || x >= t.width {
		return false
	}
	for _, s := range t.shells {
		if s == x {
			return false
		}
	}
	for _, st := range t.seaweed {
		if x >= st.col-ShellClearance && x <= st.col+SeaweedWidth-1+ShellClearance {
			return false
		}
	}
	return true
}
