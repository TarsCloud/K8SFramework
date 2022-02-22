package scaffold

import "math/rand"

var CheckLeftInRight = func(l, r map[string]string) bool {
	if len(l) > len(r) {
		return false
	}
	for lk, lv := range l {
		if rv, ok := r[lk]; !ok || rv != lv {
			return false
		}
	}
	return true
}

func RandStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
