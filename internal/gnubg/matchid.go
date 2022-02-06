package gnubg

/*
 * Calculate log2 of Cube value.
 *
 * Input:
 *   n: cube value
 *
 * Returns:
 *   log(n)/log(2)
 *
 */
func logCube(n int) int {
	i := 0

	for n = n >> 1; n > 0; n = n >> 1 {
		i++
	}

	return i
}
