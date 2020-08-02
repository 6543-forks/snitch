package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"

	"gopkg.in/go-ini/ini.v1"
)

// GiteaCredentials contains PersonalToken for GitLab API authorization
// and Host for possibly implementing support for self-hosted instances
type GiteaCredentials struct {
	Host          string
	PersonalToken string
}

func (creds GiteaCredentials) query(method, url string) (map[string]interface{}, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("TOKEN", creds.PersonalToken)

	return QueryHTTP(req)
}

func (creds GiteaCredentials) getIssue(repo string, todo Todo) (map[string]interface{}, error) {
	json, err := creds.query(
		"GET",
		"http://"+creds.Host+"/api/v1/repos/"+url.QueryEscape(repo)+"/issues/"+(*todo.ID)[1:]) // self-hosted

	if err != nil {
		return nil, err
	}

	return json, nil
}

func (creds GiteaCredentials) postIssue(repo string, todo Todo, body string) (Todo, error) {
	params := url.Values{}
	params.Add("title", todo.Title)
	params.Add("description", body)

	json, err := creds.query(
		"POST",
		"https://"+creds.Host+"/api/v1/repos/"+url.QueryEscape(repo)+"/issues?"+params.Encode()) // self-hosted
	if err != nil {
		return todo, err
	}

	id := "#" + strconv.Itoa(int(json["iid"].(float64)))
	todo.ID = &id

	return todo, err
}

func (creds GiteaCredentials) getHost() string {
	return creds.Host
}

// GiteaCredentialsFromFile gets GiteaCredentials from a filepath
func GiteaCredentialsFromFile(filepath string) []GiteaCredentials {
	credentials := []GiteaCredentials{}

	cfg, err := ini.Load(filepath)
	if err != nil {
		return credentials
	}

	for _, section := range cfg.Sections()[1:] {
		credentials = append(credentials, GiteaCredentials{
			Host:          section.Name(),
			PersonalToken: section.Key("personal_token").String(),
		})
	}

	return credentials
}

// GiteaCredentialsFromToken returns a GiteaCredentials from a string token
func GiteaCredentialsFromToken(token string) (GiteaCredentials, error) {
	credentials := strings.Split(token, ":")

	switch len(credentials) {
	case 1:
		return GiteaCredentials{
			Host:          "gitlab.com",
			PersonalToken: credentials[0],
		}, nil
	case 2:
		return GiteaCredentials{
			Host:          credentials[0],
			PersonalToken: credentials[1],
		}, nil
	default:
		return GiteaCredentials{},
			fmt.Errorf("Couldn't parse GitLab credentials from ENV: %s", token)
	}

}

func getGiteaCredentials(creds []IssueAPI) []IssueAPI {
	tokenEnvar := os.Getenv("GITLAB_PERSONAL_TOKEN")
	xdgEnvar := os.Getenv("XDG_CONFIG_HOME")
	usr, err := user.Current()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if len(tokenEnvar) != 0 {
		for _, credential := range strings.Split(tokenEnvar, ",") {
			parsedCredentials, err := GiteaCredentialsFromToken(credential)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			creds = append(creds, parsedCredentials)
		}
	}

	// custom XDG_CONFIG_HOME
	if len(xdgEnvar) != 0 {
		filePath := path.Join(xdgEnvar, "snitch/gitlab.ini")
		if _, err := os.Stat(filePath); err == nil {
			for _, cred := range GiteaCredentialsFromFile(filePath) {
				creds = append(creds, cred)
			}
		}
	}

	// default XDG_CONFIG_HOME
	if len(xdgEnvar) == 0 {
		filePath := path.Join(usr.HomeDir, ".config/snitch/gitlab.ini")
		if _, err := os.Stat(filePath); err == nil {
			for _, cred := range GiteaCredentialsFromFile(filePath) {
				creds = append(creds, cred)
			}
		}
	}

	filePath := path.Join(usr.HomeDir, ".snitch/gitlab.ini")
	if _, err := os.Stat(filePath); err == nil {
		for _, cred := range GiteaCredentialsFromFile(filePath) {
			creds = append(creds, cred)
		}
	}

	return creds
}
