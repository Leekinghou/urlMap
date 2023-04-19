package main

var keyChar = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

func genKey(n int) string {
	if n == 0 {
		return string(keyChar[0])
	}
	l := len(keyChar)
	// make中参数意义：make([]T, len, cap)
	s := make([]byte, 0, n)
	i := len(s)
	for n > 0 && i >= 0 {
		i--
		j := n % l
		n = (n - j) / l
		s[i] = keyChar[j]
	}
	return string(s[i:])
}
