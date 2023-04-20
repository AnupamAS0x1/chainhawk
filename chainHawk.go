package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/github"
)

func main() {
	fmt.Println("Enter the name of a GitHub organization:")

	var orgName string
	_, err := fmt.Scan(&orgName)

	if err != nil {
		fmt.Println("Error reading input:", err)
		os.Exit(1)
	}

	client := github.NewClient(nil)
	org, _, err := client.Organizations.Get(context.Background(), orgName)

	if err != nil {
		fmt.Println("Error fetching organization data:", err)
		os.Exit(1)
	}

	fmt.Printf("\nOrganization details for %s:\n", orgName)
	fmt.Println("ID:", org.GetID())
	fmt.Println("Name:", org.GetName())
	fmt.Println("Description:", org.GetDescription())
	fmt.Println("Public repos:", org.GetPublicRepos())
}
