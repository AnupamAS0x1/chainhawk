package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type PackageJSON struct {
	Dependencies map[string]string `json:"dependencies"`
}

type PackageStatus struct {
	PackageName string
	Version     string
	Available   bool
}

type Repo struct {
	Name string `json:"name"`
}

type RepoReport struct {
	RepoName           string
	PackageJSONExists  bool
	NpmPackages        []PackageStatus
	GemfileExists      bool
	RubyGems           []PackageStatus
	RequirementsExists bool
	PythonPackages     []PackageStatus
	LeakedAPIKeys      []string
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter the GitHub organization name: ")
	org, _ := reader.ReadString('\n')
	org = strings.TrimSpace(org)

	// Replace "your_token_here" with your actual token
	token := "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"

	repos, err := fetchRepos(org, token)
	if err != nil {
		fmt.Printf("Error fetching repositories: %v\n", err)
		os.Exit(1)
	}
	var reports []RepoReport

	for _, repo := range repos {
		fmt.Printf("\nChecking repository: %s/%s\n", org, repo.Name)

		report := RepoReport{
			RepoName: repo.Name,
		}

		report.PackageJSONExists = checkNPMRepo(org, repo.Name, &report)
		report.GemfileExists = checkRubyGemsRepo(org, repo.Name, &report)
		report.RequirementsExists = checkPythonPipRepo(org, repo.Name, &report)
		report.LeakedAPIKeys, _ = searchGithub(org, token)

		reports = append(reports, report)
	}

	generateReport(reports)
}

func searchGithub(org, token string) ([]string, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	opts := &github.SearchOptions{Sort: "indexed"}
	fmt.Println("Searching for leaks...")
	queries := []string{
		"api in:file org:" + org,
		"secrets in:file org:" + org,
		"aws_key in:file org:" + org,
	}

	var leakedKeys []string
	for _, query := range queries {
		results, _, err := client.Search.Code(ctx, query, opts)
		if err != nil {
			return nil, err
		}

		for _, result := range results.CodeResults {
			leakedKeys = append(leakedKeys, result.GetHTMLURL())
		}
	}

	return leakedKeys, nil
}

func checkNPMRepo(org, repoName string, report *RepoReport) bool {
	fmt.Println("Checking for package.json...")

	packageJSON, err := fetchPackageJSON(org, repoName)
	if err != nil {
		fmt.Printf("Error fetching package.json: %v\n", err)
		return false
	}

	if packageJSON == nil {
		fmt.Println("package.json not found")
		return false
	} else {
		fmt.Println("package.json found")
		report.PackageJSONExists = true
		checkDependencies(packageJSON, &report.NpmPackages)
		return true
	}
}

func checkRubyGemsRepo(org, repoName string, report *RepoReport) bool {
	fmt.Println("Checking for Gemfile...")

	gemfile, err := fetchFileContent(org, repoName, "Gemfile")
	if err != nil {
		fmt.Printf("Error fetching Gemfile: %v\n", err)
		return false
	}

	if gemfile == "" {
		fmt.Println("Gemfile not found")
		return false
	} else {
		fmt.Println("Gemfile found")
		report.GemfileExists = true
		checkRubyGemsDependencies(gemfile, &report.RubyGems)
		return true
	}
}

func checkPythonPipRepo(org, repoName string, report *RepoReport) bool {
	fmt.Println("Checking for requirements.txt...")

	requirements, err := fetchFileContent(org, repoName, "requirements.txt")
	if err != nil {
		fmt.Printf("Error fetching requirements.txt: %v\n", err)
		return false
	}

	if requirements == "" {
		fmt.Println("requirements.txt not found")
		return false
	} else {
		fmt.Println("requirements.txt found")
		report.RequirementsExists = true
		checkPythonPipDependencies(requirements, &report.PythonPackages)
		return true
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

func checkDependencies(packageJSON *PackageJSON, packages *[]PackageStatus) {
	for packageName, version := range packageJSON.Dependencies {
		available, err := isNpmPackageAvailable(packageName)
		if err != nil {
			fmt.Printf("Error checking package '%s': %v\n", packageName, err)
			continue
		}
		*packages = append(*packages, PackageStatus{
			PackageName: packageName,
			Version:     version,
			Available:   available,
		})
	}
}

func checkRubyGemsDependencies(gemfile string, gems *[]PackageStatus) {
	gemNames := parseGemfile(gemfile)
	for _, gem := range gemNames {
		available, err := isRubyGemAvailable(gem)
		if err != nil {
			fmt.Printf("Error checking gem '%s': %v\n", gem, err)
			continue
		}
		*gems = append(*gems, PackageStatus{
			PackageName: gem,
			Available:   available,
		})
	}
}

func checkPythonPipDependencies(requirements string, packages *[]PackageStatus) {
	packageNames := parseRequirements(requirements)
	for _, packageName := range packageNames {
		available, err := isPypiPackageAvailable(packageName)
		if err != nil {
			fmt.Printf("Error checking package '%s': %v\n", packageName, err)
			continue
		}
		*packages = append(*packages, PackageStatus{
			PackageName: packageName,
			Available:   available,
		})
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

func generateReport(reports []RepoReport) {
	for _, report := range reports {
		fmt.Println("=======================================")
		fmt.Printf("Repo: %s\n", report.RepoName)
		fmt.Println("=======================================")

		fmt.Printf("package.json exists: %v\n", report.PackageJSONExists)
		if report.PackageJSONExists {
			fmt.Println("NPM packages:")
			for _, pkg := range report.NpmPackages {
				status := "Not Available"
				if pkg.Available {
					status = "Available"
				}
				fmt.Printf("- %s (Version: %s): %s\n", pkg.PackageName, pkg.Version, status)
			}
		}

		fmt.Printf("\nGemfile exists: %v\n", report.GemfileExists)
		if report.GemfileExists {
			fmt.Println("Ruby gems:")
			for _, pkg := range report.RubyGems {
				status := "Not Available"
				if pkg.Available {
					status = "Available"
				}
				fmt.Printf("- %s: %s\n", pkg.PackageName, status)
			}
		}

		fmt.Printf("\nrequirements.txt exists: %v\n", report.RequirementsExists)
		if report.RequirementsExists {
			fmt.Println("Python packages:")
			for _, pkg := range report.PythonPackages {
				status := "Not Available"
				if pkg.Available {
					status = "Available"
				}
				fmt.Printf("- %s: %s\n", pkg.PackageName, status)
			}
		}

		fmt.Printf("\nLeaked API keys: %v\n", len(report.LeakedAPIKeys) > 0)
		if len(report.LeakedAPIKeys) > 0 {
			fmt.Println("Leaked keys:")
			for _, key := range report.LeakedAPIKeys {
				fmt.Printf("- %s\n", key)
			}
		}
		fmt.Println("\n")
	}
}
