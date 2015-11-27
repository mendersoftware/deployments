package main

import (
	"fmt"

	"github.com/mendersoftware/artifacts/utils/uuid"
)

func Test() {
	fmt.Println("Test")
}

func main() {
	fmt.Println("Hello Artifacts!")

	fmt.Println("Commit =", Commit)
	fmt.Println("Tag =", Tag)
	fmt.Println("Branch =", Branch)
	fmt.Println("BuildNumber =", BuildNumber)

	id, _ := uuid.MakeUUID()
	fmt.Println("UUID=", id)
}
