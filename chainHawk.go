package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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

	// Replace "your_token_here" with your actual token
	token := ""

	repos, err := fetchRepos(org, token)
	if err != nil {
		fmt.Printf("Error fetching repositories: %v\n", err)
		os.Exit(1)
	}

	//... rest of the main function

	for _, repo := range repos {
		fmt.Printf("\nChecking repository: %s/%s\n", org, repo.Name)

		checkNPMRepo(org, repo.Name)
		checkRubyGemsRepo(org, repo.Name)
		checkPythonPipRepo(org, repo.Name)

		// fmt.Println("Checking for leaked API keys:")
		// checkForLeakedAPIKeys(org, repo.Name)
	}
}

// Add checkNPMRepo function
func checkNPMRepo(org, repoName string) {
	fmt.Println("Checking for package.json...")

	packageJSON, err := fetchPackageJSON(org, repoName)
	if err != nil {
		fmt.Printf("Error fetching package.json: %v\n", err)
		return
	}

	if packageJSON == nil {
		fmt.Println("package.json not found")
	} else {
		fmt.Println("package.json found")
		checkDependencies(packageJSON)
	}
}

// Add checkRubyGemsRepo function
func checkRubyGemsRepo(org, repoName string) {
	fmt.Println("Checking for Gemfile...")

	gemfile, err := fetchFileContent(org, repoName, "Gemfile")
	if err != nil {
		fmt.Printf("Error fetching Gemfile: %v\n", err)
		return
	}

	if gemfile == "" {
		fmt.Println("Gemfile not found")
	} else {
		fmt.Println("Gemfile found")
		checkRubyGemsDependencies(gemfile)
	}
}

// Add checkPythonPipRepo function
func checkPythonPipRepo(org, repoName string) {
	fmt.Println("Checking for requirements.txt...")

	requirements, err := fetchFileContent(org, repoName, "requirements.txt")
	if err != nil {
		fmt.Printf("Error fetching requirements.txt: %v\n", err)
		return
	}

	if requirements == "" {
		fmt.Println("requirements.txt not found")
	} else {
		fmt.Println("requirements.txt found")
		checkPythonPipDependencies(requirements)
	}
}

func fetchRepos(org string, token string) ([]Repo, error) {
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos", org)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
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
		available, err := isNpmPackageAvailable(packageName)
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

func checkRubyGemsDependencies(gemfile string) {
	gems := parseGemfile(gemfile)
	for _, gem := range gems {
		available, err := isRubyGemAvailable(gem)
		if err != nil {
			fmt.Printf("Error checking gem '%s': %v\n", gem, err)
			continue
		}

		if available {
			fmt.Printf("Gem '%s' is available\n", gem)
		} else {
			fmt.Printf("Gem '%s' is not available\n", gem)
		}
	}
}

func checkPythonPipDependencies(requirements string) {
	packages := parseRequirements(requirements)
	for _, packageName := range packages {
		available, err := isPypiPackageAvailable(packageName)
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

func parseGemfile(gemfile string) []string {
	lines := strings.Split(gemfile, "\n")
	gems := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gem ") {
			parts := strings.Split(line, " ")
			if len(parts) >= 2 {
				gemName := strings.Trim(parts[1], `'""`)
				gems = append(gems, gemName)
			}
		}
	}

	return gems
}

func parseRequirements(requirements string) []string {
	lines := strings.Split(requirements, "\n")
	packages := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			parts := strings.Split(line, "==")
			packages = append(packages, parts[0])
		}
	}

	return packages
}

func isNpmPackageAvailable(packageName string) (bool, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", packageName)

	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func isRubyGemAvailable(gemName string) (bool, error) {
	url := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s.json", gemName)

	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func isPypiPackageAvailable(packageName string) (bool, error) {
	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", packageName)

	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func fetchFileContent(org, repoName, fileName string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/%s", org, repoName, fileName)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// func checkForLeakedAPIKeys(org, token string) {
// 	patterns := []string{
// 		"(?i)\\/\\*\\s*TODO",
// 		"(?i)//\\s*TODO",
// 		"(?i)#\\s*TODO",
// 	}

// 	for _, pattern := range patterns {
// 		query := fmt.Sprintf("%s org:%s", pattern, org)
// 		resp, err := searchGitHub(query, token)

// 		if err != nil {
// 			fmt.Printf("Error fetching search results for pattern '%s': %v\n", pattern, err)
// 			continue
// 		}

// 		if resp.StatusCode != http.StatusOK {
// 			fmt.Printf("Error fetching search results for pattern '%s': status code: %d\n", pattern, resp.StatusCode)
// 			continue
// 		}

// 		// Process the search results
// 		// ...
// 	}
// }

func searchLeakedAPIKeys(org, repoName, pattern string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/search/code?q=%s+in:file+repo:%s/%s", url.QueryEscape(pattern), org, repoName)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error fetching search results, status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var searchResult struct {
		Items []struct {
			Path string `json:"path"`
		} `json:"items"`
	}

	err = json.Unmarshal(body, &searchResult)
	if err != nil {
		return nil, err
	}

	leakedKeys := make([]string, 0, len(searchResult.Items))
	for _, item := range searchResult.Items {
		leakedKeys = append(leakedKeys, item.Path)
	}

	return leakedKeys, nil
}

// func searchGitHub(query, token string) (*http.Response, error) {
// 	url := fmt.Sprintf("https://api.github.com/search/code?q=%s", url.QueryEscape(query))

// 	client := &http.Client{}
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	req.Header.Set("Accept", "application/vnd.github+json")
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return resp, nil

//}
