package main

import (
	"context"
	"fmt"
	"time"

	"github.com/anggasct/httpio/internal/client"
)

func main() {
	client := client.New().
		WithBaseURL("https://api.github.com").
		WithTimeout(10*time.Second).
		WithHeader("Accept", "application/vnd.github.v3+json")

	ctx := context.Background()

	fmt.Println("=== HTTP GET Example ===")
	getUserInfo(ctx, client)

	fmt.Println("\n=== Query Parameters Example ===")
	getUserRepositories(ctx, client)

	fmt.Println("\n=== Multiple Query Parameters Example ===")
	searchRepositories(ctx, client)

	fmt.Println("\n=== POST Request Example ===")
	createGist(ctx, client)

	fmt.Println("\n=== PUT Request Example ===")
	updateGist(ctx, client)

	fmt.Println("\n=== PATCH Request Example ===")
	patchGist(ctx, client)

	fmt.Println("\n=== DELETE Request Example ===")
	deleteGist(ctx, client)

	fmt.Println("\n=== HEAD Request Example ===")
	checkResource(ctx, client)

	fmt.Println("\n=== OPTIONS Request Example ===")
	getAPIOptions(ctx, client)
}

func getUserInfo(ctx context.Context, client *client.Client) {
	resp, err := client.GET(ctx, "/users/octocat")
	if err != nil {
		fmt.Println("Request error:", err)
		return
	}
	defer resp.Close()

	if !resp.IsSuccess() {
		fmt.Println("Request failed:", resp.Status)
		return
	}

	var user struct {
		Login       string `json:"login"`
		Name        string `json:"name"`
		Company     string `json:"company"`
		PublicRepos int    `json:"public_repos"`
	}
	if err := resp.JSON(&user); err == nil {
		fmt.Println("User:", user.Login, "- Name:", user.Name)
		fmt.Println("Company:", user.Company)
		fmt.Println("Public Repos:", user.PublicRepos)
	} else {
		fmt.Println("Failed to parse JSON:", err)
	}
}

func getUserRepositories(ctx context.Context, client *client.Client) {
	repoResp, err := client.NewRequest("GET", "/users/octocat/repos").
		WithQuery("sort", "updated").
		WithQuery("per_page", "3").
		Do(ctx)

	if err != nil {
		fmt.Println("Request error:", err)
		return
	}
	defer repoResp.Close()

	if !repoResp.IsSuccess() {
		fmt.Println("Request failed:", repoResp.Status)
		return
	}

	var repositories []struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		StargazersCount int    `json:"stargazers_count"`
	}

	if err := repoResp.JSON(&repositories); err == nil {
		for _, repo := range repositories {
			fmt.Println("Repo:", repo.Name, "- Stars:", repo.StargazersCount)
			if repo.Description != "" {
				fmt.Println("Description:", repo.Description)
			}
		}
	} else {
		fmt.Println("Failed to parse JSON:", err)
	}
}

func searchRepositories(ctx context.Context, client *client.Client) {
	queryParams := map[string]string{
		"q":        "language:go",
		"sort":     "stars",
		"order":    "desc",
		"per_page": "3",
	}

	searchResp, err := client.NewRequest("GET", "/search/repositories").
		WithQueryMap(queryParams).
		Do(ctx)

	if err != nil {
		fmt.Println("Request error:", err)
		return
	}
	defer searchResp.Close()

	if !searchResp.IsSuccess() {
		fmt.Println("Request failed:", searchResp.Status)
		return
	}

	var searchResult struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			Name            string `json:"name"`
			FullName        string `json:"full_name"`
			StargazersCount int    `json:"stargazers_count"`
		} `json:"items"`
	}

	if err := searchResp.JSON(&searchResult); err == nil {
		fmt.Println("Total Go repositories:", searchResult.TotalCount)
		for _, repo := range searchResult.Items {
			fmt.Println("Repo:", repo.FullName, "- Stars:", repo.StargazersCount)
		}
	} else {
		fmt.Println("Failed to parse JSON:", err)
	}
}

func createGist(ctx context.Context, client *client.Client) {
	gistContent := map[string]interface{}{
		"description": "Example gist created with httpio client",
		"public":      true,
		"files": map[string]interface{}{
			"example.txt": map[string]string{
				"content": "This is an example file content",
			},
		},
	}

	resp, err := client.POST(ctx, "/gists", gistContent)
	if err != nil {
		fmt.Println("Request error:", err)
		return
	}
	defer resp.Close()

	fmt.Println("POST request status:", resp.Status)
	fmt.Println("Note: Actual gist creation would require authentication")
}

func updateGist(ctx context.Context, client *client.Client) {
	gistID := "exampleGistId"

	updateContent := map[string]interface{}{
		"description": "Updated example gist",
		"files": map[string]interface{}{
			"example.txt": map[string]string{
				"content": "This content has been updated",
			},
			"newfile.txt": map[string]string{
				"content": "This is a newly added file",
			},
		},
	}

	resp, err := client.PUT(ctx, "/gists/"+gistID, updateContent)
	if err != nil {
		fmt.Println("Request error:", err)
		return
	}
	defer resp.Close()

	fmt.Println("PUT request status:", resp.Status)
	fmt.Println("Note: Actual gist update would require authentication")
}

func patchGist(ctx context.Context, client *client.Client) {
	gistID := "exampleGistId"
	patchContent := map[string]interface{}{
		"description": "Partially updated description",
	}

	resp, err := client.PATCH(ctx, "/gists/"+gistID, patchContent)
	if err != nil {
		fmt.Println("Request error:", err)
		return
	}
	defer resp.Close()

	fmt.Println("PATCH request status:", resp.Status)
	fmt.Println("Note: Actual gist patching would require authentication")
}

func deleteGist(ctx context.Context, client *client.Client) {
	gistID := "exampleGistId"

	resp, err := client.DELETE(ctx, "/gists/"+gistID)
	if err != nil {
		fmt.Println("Request error:", err)
		return
	}
	defer resp.Close()

	fmt.Println("DELETE request status:", resp.Status)
	fmt.Println("Note: Actual gist deletion would require authentication")
}

func checkResource(ctx context.Context, client *client.Client) {
	resp, err := client.HEAD(ctx, "/users/octocat")
	if err != nil {
		fmt.Println("Request error:", err)
		return
	}
	defer resp.Close()

	fmt.Println("HEAD request status:", resp.Status)
	fmt.Println("Rate limit remaining:", resp.Header.Get("X-RateLimit-Remaining"))
}

func getAPIOptions(ctx context.Context, client *client.Client) {
	resp, err := client.OPTIONS(ctx, "/")
	if err != nil {
		fmt.Println("Request error:", err)
		return
	}
	defer resp.Close()

	fmt.Println("OPTIONS request status:", resp.Status)
	fmt.Println("Allowed methods:", resp.Header.Get("Allow"))
}
