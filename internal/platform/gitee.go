package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
)

// GiteeCollector Gitee 采集器。
type GiteeCollector struct{}

// NewGitee 创建 Gitee 采集器。
func NewGitee() *GiteeCollector {
	return &GiteeCollector{}
}

// Name 平台名。
func (c *GiteeCollector) Name() string { return "gitee" }

// Match 判断是否 gitee.com 链接。
func (c *GiteeCollector) Match(rawurl string) bool {
	return matchHost(rawurl, "gitee.com")
}

type giteeRepoJSON struct {
	Path        string `json:"path"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stargazers_count"`
	Forks       int    `json:"forks_count"`
	License     string `json:"license"`
	HTMLURL     string `json:"html_url"`
	Namespace   struct {
		Path string `json:"path"`
	} `json:"namespace"`
}

// FetchRepo 调用 Gitee API 抓取仓库详情。
func (c *GiteeCollector) FetchRepo(ctx context.Context, owner, name string) (*RepoInfo, error) {
	req, err := newRequest(ctx, fmt.Sprintf("https://api.gitee.com/api/v5/repos/%s/%s", owner, name))
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gitee api returned %d", resp.StatusCode)
	}
	var data giteeRepoJSON
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	info := &RepoInfo{
		Platform:    c.Name(),
		Owner:       data.Namespace.Path,
		Name:        data.Path,
		URL:         data.HTMLURL,
		Description: data.Description,
		Language:    data.Language,
		Stars:       data.Stars,
		Forks:       data.Forks,
		License:     data.License,
	}
	if info.Owner == "" || info.Name == "" {
		info.Owner, info.Name = owner, name
	}
	info.URL = strings.TrimSuffix(info.URL, ".git")
	if info.URL == "" {
		info.URL = fmt.Sprintf("https://gitee.com/%s/%s", info.Owner, info.Name)
	}
	return info, nil
}

// repoLinkPattern 匹配 /owner/name 形式的站内链接。
var repoLinkPattern = regexp.MustCompile(`^/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+/?$`)

// giteeNonRepoSegments 非仓库路径的首段黑名单。
var giteeNonRepoSegments = map[string]bool{
	"explore": true, "features": true, "help": true, "login": true, "logout": true,
	"signup": true, "join": true, "about": true, "api": true, "oauth": true,
	"search": true, "dashboard": true, "notifications": true, "settings": true,
	"profile": true, "enterprises": true, "premium": true, "jobs": true,
	"terms": true, "privacy": true, "static": true, "assets": true, "organizations": true,
}

// FetchGVP 解析 gitee.com/explore/gvp 页面得到 GVP 仓库列表，
// 再逐个调用 Gitee API 校验并补全详情。
func (c *GiteeCollector) FetchGVP(ctx context.Context) ([]*RepoInfo, error) {
	req, err := newRequest(ctx, "https://gitee.com/explore/gvp")
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gitee gvp page returned %d", resp.StatusCode)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	// 页面结构可能调整，采用启发式：收集所有 /owner/name 形式的站内链接并去重
	type repoRef struct{ owner, name string }
	seen := map[repoRef]bool{}
	var candidates []repoRef
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if !repoLinkPattern.MatchString(href) {
			return
		}
		parts := strings.Split(strings.Trim(href, "/"), "/")
		if giteeNonRepoSegments[strings.ToLower(parts[0])] {
			return
		}
		ref := repoRef{owner: parts[0], name: parts[1]}
		if !seen[ref] {
			seen[ref] = true
			candidates = append(candidates, ref)
		}
	})
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no repos parsed from gitee gvp page (possibly blocked by anti-crawler)")
	}
	var repos []*RepoInfo
	for _, ref := range candidates {
		info, err := c.FetchRepo(ctx, ref.owner, ref.name)
		if err != nil {
			logrus.WithError(err).Warnf("gitee gvp: skipping %s/%s", ref.owner, ref.name)
			continue
		}
		repos = append(repos, info)
	}
	if len(repos) == 0 {
		return nil, fmt.Errorf("gitee gvp: none of %d candidate links passed api validation", len(candidates))
	}
	return repos, nil
}
