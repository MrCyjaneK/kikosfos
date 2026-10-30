package main

import "C"

//export KIKOAPI_hello
func KIKOAPI_hello() *C.char {
	return C.CString("hello world")
}

func main() {}
