package ghbackup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
)

// Get repositories from Github.
// Follow all "next" links.
func fetch(account, secret, api string, doer Doer) ([]repo, error) {
	var allRepos []repo

	currentURL, err := getURL(account, secret, api, doer)
	if err != nil {
		return allRepos, err
	}

	// Go through all pages
	for {
		req, err := http.NewRequest("GET", currentURL, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot create request: %v", err)
		}
		if secret != "" {
			// For token authentication `account` will be ignored
			req.SetBasicAuth(account, secret)
		}
		res, err := doer.Do(req)
		if err != nil {
			return nil, fmt.Errorf("cannot get repos: %v", err)
		}
		defer func() {
			_ = res.Body.Close()
		}()
		if res.StatusCode >= 300 {
			return nil, fmt.Errorf("bad response from %s: %v", res.Request.URL, res.Status)
		}

		var repos []repo
		err = json.NewDecoder(res.Body).Decode(&repos)
		if err != nil {
			return nil, fmt.Errorf("cannot decode JSON response: %v", err)
		}

		allRepos = append(allRepos, selectRepos(repos, account)...)

		// Set url for next iteration
		currentURL = getNextURL(res.Header)

		// Done if no next URL
		if currentURL == "" {
			return allRepos, nil
		}
	}
}

func getURL(account, secret, api string, doer Doer) (string, error) {
	user := "user"
	if secret == "" {
		category, err := getCategory(account, api, doer)
		if err != nil {
			return "", err
		}
		user = category + "/" + account
	}
	url := api + "/" + user + "/repos?per_page=100"
	return url, nil
}

// Returns "users" or "orgs" depending on type of account
func getCategory(account, api string, doer Doer) (string, error) {
	req, err := http.NewRequest("GET", strings.Join([]string{api, "users", account}, "/"), nil)
	if err != nil {
		return "", fmt.Errorf("cannot create HTTP request: %v", err)
	}
	res, err := doer.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot get user info: %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("bad response from %s: %v", res.Request.URL, res.Status)
	}

	var a struct {
		Type string
	}
	err = json.NewDecoder(res.Body).Decode(&a)
	if err != nil {
		return "", fmt.Errorf("cannot decode JSON response: %v", err)
	}

	if a.Type == "User" {
		return "users", nil
	}
	if a.Type == "Organization" {
		return "orgs", nil
	}
	return "", fmt.Errorf("unknown type of account %s for %s", a.Type, account)
}

func selectRepos(repos []repo, account string) []repo {
	if account == "" {
		return repos
	}
	var res []repo
	for _, r := range repos {
		if path.Dir(r.Path) == account {
			res = append(res, r)
		}
	}
	return res
}

func getNextURL(header http.Header) string {
	linkHeader := header["Link"]
	if len(linkHeader) == 0 {
		return ""
	}
	parts := strings.Split(linkHeader[0], ",")
	if len(parts) == 0 {
		return ""
	}
	firstLink := parts[0]
	if !strings.Contains(firstLink, "rel=\"next\"") {
		return ""
	}
	parts = strings.Split(firstLink, ";")
	if len(parts) == 0 {
		return ""
	}
	urlInBrackets := parts[0]
	if len(urlInBrackets) < 3 {
		return ""
	}
	return urlInBrackets[1 : len(urlInBrackets)-1]
}
