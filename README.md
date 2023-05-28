# ChainHawk

ChainHawk is a tool written in Go that helps analyze repositories within a GitHub organization. It provides various functionalities to check dependencies, search for leaked API keys, and verify the availability of packages from external registries.

## Features

- Fetch and analyze repositories within a GitHub organization
- Check for `package.json` in repositories and verify package dependencies
- Check for `Gemfile` in repositories and verify Ruby gem dependencies
- Check for `requirements.txt` in repositories and verify Python package dependencies
- Search for leaked API keys in repository files
- Verify the availability of packages from external registries (npm, RubyGems, PyPI)

## Usage

1. Make sure you have Go installed on your system.
2. Clone this repository and navigate to the project directory.
3. Replace the github token by using any text editor
    
    `vi chainHawk.go`
    
    ```token := "your_token"```
   
4. Open the terminal and build the project using the following command:
    `go build` 

5. Run the executable file with the desired options. For example, to analyze repositories within a GitHub organization, use the following command:
    `./chainhawl`


6. Follow the instructions provided by the tool to input the GitHub organization name and the required authentication token.

## Dependencies

The tool relies on the following dependencies:

- Go 1.16 or later
- `fmt` package
- `net/http` package
- `encoding/json` package

## License

This project is licensed under the [MIT License](LICENSE).

Feel free to contribute, report issues, or suggest improvements.


