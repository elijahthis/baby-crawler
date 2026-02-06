package shared

import (
	"net/url"
	"strings"
)

func CleanKey(key string) string {
	key = strings.TrimPrefix(key, "http://")
	key = strings.TrimPrefix(key, "https://")

	if strings.HasSuffix(key, "/") {
		key += "index.html"
	} else if !strings.Contains(key, ".") {
		key += "/index.html"
	}
	return key
}

func GetDomain(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	return u.Host, nil
}

func CompareDomains(parent, child string) (bool, error) {
	parentDomain, err := GetDomain(parent)
	if err != nil {
		return false, err
	}
	childDomain, err := GetDomain(child)
	if err != nil {
		return false, err
	}

	return parentDomain == childDomain, nil
}

func ResolveURL(parent, link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", nil
	}

	base, err := url.Parse(parent)
	if err != nil {
		return "", nil
	}

	urlObj := base.ResolveReference(u)
	urlObj.Fragment = ""

	return urlObj.String(), nil
}
