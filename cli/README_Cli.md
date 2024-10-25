## Using the CLI

If you have already completed the steps for [Create Account](https://storage.titannet.io/login) and [Set API Key](../doc/access_key.jpg), your CLI and Titan Storage are already configured. To use the CLI, you can follow the steps below:

### 1. Compile the Command-Line File Using the Go Build Tool
First, ensure that Go is installed in your environment. Use the following command to compile the CLI file:

```bash
go build -o cli main.go
```

This will generate an executable file named cli, which you can use to interact with the API.

### 2. Initialize Environment Variables
Before using the CLI, you need to set up environment variables using the API key you registered in the browser:

```bash
export TITAN_URL=<your_titan_url>
export API_KEY=<your_api_key>
```
- TITAN_URL: The URL of the Titan service.
- API_KEY: The API key you obtained from the Titan platform.


### 3. Execute CLI Methods
Once the environment variables are set, you can use the CLI to execute API methods. The basic format to execute a command is:

```bash
./cli <api_method> <arguments>
```


### Methods

* [ completion](_completion.md)	 - Generate the autocompletion script for the specified shell
* [ delete](_delete.md)	 - delete file
* [ folder](_folder.md)	 - Manage folders
* [ gendoc](_gendoc.md)	 - Generate markdown documentation
* [ get](_get.md)	 - get file
* [ list](_list.md)	 - list files
* [ upload](_upload.md)	 - upload file
* [ url](_url.md)	 - get file url by cid
* [ version](_version.md)	 - Print the version number




