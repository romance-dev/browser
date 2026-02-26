package main

import (
	u "net/url"
)

func checkCommand(args []string, cmd string) (bool, string) {
	found := false
	var url string

	for _, s := range args {
		if url == "" && isValidURL(s) {
			url = s
			continue
		}

		if s == cmd {
			found = true
		}
	}
	return found, url
}

func isValidURL(url string) bool {
	u, err := u.Parse(url)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return true
}
