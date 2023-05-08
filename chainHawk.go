package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type PackageJSON struct {
	Dependencies map[string]string `json:"dependencies"`
}

type Repo struct {
	Name string `json:"name"`
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter the GitHub organization name: ")
	org, _ := reader.ReadString('\n')
	org = strings.TrimSpace(org)

	repos, err := fetchRepos(org)
	if err != nil {
		fmt.Printf("Error fetching repositories: %v\n", err)
		os.Exit(1)
	}

	for _, repo := range repos {
		fmt.Printf("\nChecking repository: %s/%s\n", org, repo.Name)

		packageJSON, err := fetchPackageJSON(org, repo.Name)
		if err != nil {
			fmt.Printf("Error fetching package.json: %v\n", err)
			continue
		}

		if packageJSON == nil {
			fmt.Println("package.json not found")
		} else {
			fmt.Println("package.json found")
			checkDependencies(packageJSON)
		}
	}
}

func fetchRepos(org string) ([]Repo, error) {
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos", org)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching repos, status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var repos []Repo
	err = json.Unmarshal(body, &repos)
	if err != nil {
		return nil, err
	}

	return repos, nil
}

func fetchPackageJSON(org, repoName string) (*PackageJSON, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/package.json", org, repoName)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var packageJSON PackageJSON
	err = json.Unmarshal(body, &packageJSON)
	if err != nil {
		return nil, err
	}

	return &packageJSON, nil
}

func checkDependencies(packageJSON *PackageJSON) {
	for packageName := range packageJSON.Dependencies {
		available, err := isPackageAvailable(packageName)
		if err != nil {
			fmt.Printf("Error checking package '%s': %v\n", packageName, err)
			continue
		}

		if available {
			fmt.Printf("Package '%s' is available\n", packageName)
		} else {
			fmt.Printf("Package '%s' is not available\n", packageName)
		}
	}
}

func isPackageAvailable(packageName string) (bool, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", packageName)

	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
