package slices

func PopFirst[T any](s []T) (T, []T) {
	return s[0], s[1:]
}

func PopLast[T any](s []T) (T, []T) {
	return s[len(s)-1], s[:len(s)-1]
}
