package robots

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

type RobotsChecker struct {
	userAgent string
	cache     map[string]*robotstxt.Group
	client    *http.Client
	mu        sync.RWMutex
}

func NewRobotsChecker(userAgent string, timeout time.Duration) *RobotsChecker {
	return &RobotsChecker{
		userAgent: userAgent,
		cache:     make(map[string]*robotstxt.Group),
		client:    &http.Client{Timeout: timeout},
	}
}

func (r *RobotsChecker) IsAllowed(targetURL string) bool {
	u, err := url.Parse(targetURL)
	if err != nil {
		fmt.Println("Error: Unable to parse targetURL")
		return false
	}

	domain := u.Host
	scheme := u.Scheme

	r.mu.RLock()
	group, exists := r.cache[domain]
	r.mu.RUnlock()

	if exists {
		if group == nil {
			return true
		}
		return group.Test(u.Path)
	}

	group = r.fetchRobotsTxt(scheme, domain)

	r.mu.Lock()
	r.cache[domain] = group
	r.mu.Unlock()

	if group == nil {
		return true
	}
	return group.Test(u.Path)
}

func (r *RobotsChecker) fetchRobotsTxt(scheme, domain string) *robotstxt.Group {
	robotsURL := scheme + "://" + domain + "/robots.txt"
	resp, err := r.client.Get(robotsURL)
	if err != nil {
		fmt.Println("No robots.txt found")
		return nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	data, err := robotstxt.FromResponse(resp)
	if err != nil {
		return nil
	}

	return data.FindGroup(r.userAgent)
}
