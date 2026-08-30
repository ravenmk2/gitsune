// Package platform 平台采集器：按 URL 识别平台并抓取仓库详情。
package platform

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RepoInfo 采集到的仓库信息，字段与 repo 表对应。
type RepoInfo struct {
	Platform    string
	Owner       string
	Name        string
	URL         string
	Description string
	Language    string
	Stars       int
	Forks       int
	License     string
}

// Collector 平台采集器接口，新平台实现并注册即可扩展。
type Collector interface {
	Name() string
	Match(rawurl string) bool
	FetchRepo(ctx context.Context, owner, name string) (*RepoInfo, error)
}

// Registry 采集器注册表。
type Registry struct {
	mu         sync.RWMutex
	collectors []Collector
}

// NewRegistry 创建注册表。
func NewRegistry() *Registry {
	return &Registry{}
}

// Register 注册采集器。
func (r *Registry) Register(c Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors = append(r.collectors, c)
}

// Get 按名称取采集器，不存在返回 nil。
func (r *Registry) Get(name string) Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.collectors {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// Match 返回能处理该 URL 的采集器，无匹配返回 nil。
func (r *Registry) Match(rawurl string) Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.collectors {
		if c.Match(rawurl) {
			return c
		}
	}
	return nil
}

// httpClient 统一 30s 超时。
var httpClient = &http.Client{Timeout: 30 * time.Second}

// newRequest 构造带 UA 的 GET 请求。
func newRequest(ctx context.Context, rawurl string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawurl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gitsune")
	req.Header.Set("Accept", "*/*")
	return req, nil
}

// parseURLHost 解析 URL 并返回小写 host。
func parseURLHost(rawurl string) (string, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid url: %s", rawurl)
	}
	return strings.ToLower(u.Hostname()), nil
}

// ParseRepoPath 从 URL path 解析 owner/name（忽略多余段与 .git 后缀）。
func ParseRepoPath(rawurl string) (owner, name string, err error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return "", "", err
	}
	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segs) < 2 {
		return "", "", fmt.Errorf("missing owner/name in url: %s", rawurl)
	}
	owner, name = segs[0], strings.TrimSuffix(segs[1], ".git")
	if owner == "" || name == "" {
		return "", "", fmt.Errorf("missing owner/name in url: %s", rawurl)
	}
	return owner, name, nil
}

// matchHost 判断 URL host 是否属于给定域名（含 www. 前缀）。
func matchHost(rawurl string, hosts ...string) bool {
	h, err := parseURLHost(rawurl)
	if err != nil {
		return false
	}
	for _, host := range hosts {
		if h == host || h == "www."+host {
			return true
		}
	}
	return false
}

// parseNumber 解析带千分位逗号或 K/M 后缀的数字文本，如 "31,729"、"2.1K"。
func parseNumber(s string) int {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return 0
	}
	mult := 1.0
	last := s[len(s)-1]
	switch last {
	case 'K', 'k':
		mult = 1000
		s = s[:len(s)-1]
	case 'M', 'm':
		mult = 1000000
		s = s[:len(s)-1]
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int(f * mult)
}
