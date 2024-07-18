package kloudinary

func Map[T, V comparable](data []T, fn func(v T) V) []V {
	var result []V
	for _, dt := range data {
		result = append(result, fn(dt))
	}
	return result
}

func ForEach[T any](data []T, fn func(v T)) {
	for _, dt := range data {
		fn(dt)
	}
}

func ExtractFromMap[K, V comparable](data, result map[K]V, fn func(value V) bool) {
	for k, v := range data {
		if fn(v) {
			result[k] = v
		}
	}
}

func Filter[T comparable](data []T, fn func(v T) bool) []T {
	var result []T
	for _, dt := range data {
		if fn(dt) {
			result = append(result, dt)
		}
	}
	return result
}
