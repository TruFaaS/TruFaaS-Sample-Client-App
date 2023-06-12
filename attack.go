package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
)

func main() {
	fnName := os.Args[1]
	if fnName == "" {
		fnName = "sample_fn"
	}
	// Generate a random integer between 1 and 100
	functionTimeout := rand.Intn(500) + 1

	// Convert the random integer to a string
	spec := `{"spec": {"functionTimeout": ` + strconv.Itoa(functionTimeout) + `}}`

	cmd := exec.Command("kubectl", "patch", "function", fnName, "-p", spec, "--type=merge")
	cmd.Dir = "."
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	fmt.Println("Simulated Attack Scenario")
}
