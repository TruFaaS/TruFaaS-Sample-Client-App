package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/exec"
)

func main() {

	// Port forwarding must be manually run --> kubectl port-forward svc/router 31314:80 -n fission

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Access the variables from the .env file
	fnEnv := os.Getenv("FN_ENV")
	fnName := os.Getenv("FN_NAME")
	fnFile := os.Getenv("FN_FILE")
	fnFileType := os.Getenv("FN_FILE_TYPE")

	deployFunction(fnName, fnEnv, fnFile, fnFileType)
	invokeFunction(fnName)
	deleteFunction(fnName)

}

func deployFunction(fnName string, fnEnv string, fnFile string, fnFileType string) {
	// Command 1: fission fn create --name test --env nodejs --code hello.js
	cmd1 := exec.Command("fission", "fn", "create", "--name", fnName, "--env", fnEnv, fnFileType, fnFile)
	cmd1.Dir = "."
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err1 := cmd1.Run()
	if err1 != nil {
		panic(err1)
	}

	// Command 2: fission route create --name test --function test --url test
	cmd2 := exec.Command("fission", "route", "create", "--name", fnName, "--function", fnName, "--url", fnName)
	cmd2.Dir = "."
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		panic(err2)
	}
}

func invokeFunction(fnName string) {
	// Command: curl http://localhost:31314/test
	cmd := exec.Command("curl", "http://localhost:31314/"+fnName)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("error invoking:", err)
		return
	}
	fmt.Println("function invoked. result:", string(output))
}

func deleteFunction(fnName string) {
	// Command 1: fission fn delete --name test
	cmd1 := exec.Command("fission", "fn", "delete", "--name", fnName)
	cmd1.Dir = "."
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err1 := cmd1.Run()
	if err1 != nil {
		panic(err1)
	}

	// Command 2: fission route delete --name test
	cmd2 := exec.Command("fission", "route", "delete", "--name", fnName)
	cmd2.Dir = "."
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		panic(err2)
	}

	// Command 2: fission pkg delete --orphan=true
	cmd3 := exec.Command("fission", "pkg", "delete", "--orphan=true")
	cmd3.Dir = "."
	cmd3.Stdout = os.Stdout
	cmd3.Stderr = os.Stderr
	err3 := cmd3.Run()
	if err3 != nil {
		panic(err2)
	}
}
