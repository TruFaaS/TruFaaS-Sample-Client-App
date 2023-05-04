package main

import (
	"os"
	"os/exec"
)

func main() {
	fnName := os.Args[1]
	if fnName == "" {
		fnName = "sample_fn"
	}

	fileName := os.Args[2]
	if fileName == "" {
		fileName = "sample_fn.js"
	}

	// Command 1: fission fn create --name test --env nodejs --code sample_fn.js
	cmd1 := exec.Command("fission", "fn", "create", "--name", fnName, "--env", "nodejs", "--code", fileName)
	cmd1.Dir = "."
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err1 := cmd1.Run()

	if err1 != nil {
		panic(err1)
	}
	//Command 2: fission route create --name test --function test --url test
	cmd2 := exec.Command("fission", "route", "create", "--name", fnName, "--function", fnName, "--url", fnName)
	cmd2.Dir = "."
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		panic(err2)
	}
}
