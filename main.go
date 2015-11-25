package main

import "fmt"

var (
	Commit      string
	Tag         string
	Branch      string
	BuildNumber string
)

func PrintBuildInfo() {
	fmt.Println("Commit =", Commit)
	fmt.Println("Tag =", Tag)
	fmt.Println("Branch =", Branch)
	fmt.Println("BuildNumber =", BuildNumber)
}

func main() {
	fmt.Println("Hello Artifacts!")

	PrintBuildInfo()
}
