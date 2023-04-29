package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {

	// Port forwarding must be manually run --> kubectl port-forward svc/router 31314:80 -n fission
	deployFunction()
	invokeFunction()
	deleteFunction()

}

func deployFunction() {
	// Command 1: fission fn create --name test --env nodejs --code hello.js
	cmd1 := exec.Command("fission", "fn", "create", "--name", "test", "--env", "nodejs", "--code", "./functions/hello.js")
	cmd1.Dir = "."
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err1 := cmd1.Run()
	if err1 != nil {
		panic(err1)
	}

	// Command 2: fission route create --name test --function test --url test
	cmd2 := exec.Command("fission", "route", "create", "--name", "test", "--function", "test", "--url", "test")
	cmd2.Dir = "."
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		panic(err2)
	}
}

func invokeFunction() {
	// Command: curl http://localhost:31314/test
	cmd := exec.Command("curl", "http://localhost:31314/test")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("error invoking:", err)
		return
	}
	fmt.Println("function invoked. result:", string(output))
}

func deleteFunction() {
	// Command 1: fission fn delete --name test
	cmd1 := exec.Command("fission", "fn", "delete", "--name", "test")
	cmd1.Dir = "."
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err1 := cmd1.Run()
	if err1 != nil {
		panic(err1)
	}

	// Command 2: fission delete create --name test
	cmd2 := exec.Command("fission", "route", "delete", "--name", "test")
	cmd2.Dir = "."
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		panic(err2)
	}
}
