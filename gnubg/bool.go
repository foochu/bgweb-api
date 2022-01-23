package gnubg

func itob(x int) bool {
	return x != 0
}

func btoi(x bool) int {
	if x {
		return 1
	}
	return 0
}
