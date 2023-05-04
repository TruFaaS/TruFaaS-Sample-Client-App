# TruFaaS Client Application

This artifact contains a sample application for function creation and function invocation.
This README will guid you through using this application locally.
Prior to using this application, ensure that you have installed the TruFaaS version of Fission 
and the TruFaaS external component.

### Prerequisites:
1. Build and deploy the TruFaaS external component on your local machine on port 8080.
2. Build and deploy the TruFaaS version of Fission.
3. Run the command ```kubectl port-forward svc/router 31314:80 -n fission``` on a terminal. Make sure that it continues to run while you use this application.
4. Create the Fission environment relevant to the programming language of the function.
   - For JS (which is used in the sample application), run ```fission env create --name nodejs --image fission/node-env``` in the terminal.

### Function Creation
1. Open a terminal inside the source folder.
2. Run the command ```go run fn_create.go {functionName} {functionSourceCode}```. 
   - Replace ```{functionName}``` with what you would like your function to be named.
   - Replace ```{functionSourceCode}``` with the file path of your function source code. A sample function ```sample_fn.js``` has been provided here.
   - For example, ```go run fn_invoke.go sample_function sample_fn.js``` creates a function named sample_function from the ```sample_fn.js``` file in the client app base folder.
3. If the function was created successfully, you should get the response
    ```bash
      [TruFaaS] Function Trust Value Generated.
      function 'sample_function' created
      trigger 'sample_function' created
    ```

### Function Invocation
1. First, create a function. Refer to the function creation section.
2. Open a terminal inside the source folder.
3. Run the command ```go run fn_invoke.go {URL of the function}```.
    - If you have run the function locally, the URL is ```http://localhost:31314/{functionName}```
4. If the function was invoked successfully, you should get the response
    ```bash
      MAC tag verification succeeded
      [TruFaaS] Trust verification value received:  true
      Function invocation result: // results
    ```
5. If you want to test a case where trust verification fails, delete the ```tree.gob``` file in the TruFaaS external component and run the function invocation command again. If run successfully, you should get the response
    ```bash
      MAC tag verification succeeded
      Trust verification value received from TruFaaS:  false
      Function invocation result:  [TruFaaS] Function Invocation Stopped as Function Trust Verification Failed.
    ```