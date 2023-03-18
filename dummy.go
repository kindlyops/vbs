//go:build neverbuild
// +build neverbuild

package main

import "fmt"

func main() {
	fmt.Println("This file should not be used. There must be an error in packaging. Check the bazel build.")
}
