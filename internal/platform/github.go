package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// GitHubCollector GitHub 采集器。
type GitHubCollector struct {
	// TokenFunc 动态获取 token（设置页可改），返回空串则匿名访问。
	TokenFunc func() string
}

// NewGitHub 创建 GitHub 采集器。
func NewGitHub(tokenFunc func() string) *GitHubCollector {
	return &GitHubCollector{TokenFunc: tokenFunc}
}

// Name 平台名。
func (c *GitHubCollector) Name() string { return "github" }

// Match 判断是否 github.com 链接。
func (c *GitHubCollector) Match(rawurl string) bool {
	return matchHost(rawurl, "github.com")
}

type githubRepoJSON struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stargazers_count"`
	Forks       int    `json:"forks_count"`
	HTMLURL     string `json:"html_url"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
	License *struct {
		SpdxID string `json:"spdx_id"`
	} `json:"license"`
}

// FetchRepo 调用 GitHub API 抓取仓库详情。
func (c *GitHubCollector) FetchRepo(ctx context.Context, owner, name string) (*RepoInfo, error) {
	req, err := newRequest(ctx, fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, name))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.TokenFunc != nil {
		if token := c.TokenFunc(); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github api returned %d", resp.StatusCode)
	}
	var data githubRepoJSON
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	info := &RepoInfo{
		Platform:    c.Name(),
		Owner:       data.Owner.Login,
		Name:        data.Name,
		URL:         data.HTMLURL,
		Description: data.Description,
		Language:    data.Language,
		Stars:       data.Stars,
		Forks:       data.Forks,
	}
	if info.Owner == "" || info.Name == "" {
		info.Owner, info.Name = owner, name
	}
	if info.URL == "" {
		info.URL = fmt.Sprintf("https://github.com/%s/%s", info.Owner, info.Name)
	}
	if data.License != nil && data.License.SpdxID != "NOASSERTION" {
		info.License = data.License.SpdxID
	}
	return info, nil
}

// FetchTrending 解析 github.com/trending 页面，since ∈ daily/weekly/monthly。
func (c *GitHubCollector) FetchTrending(ctx context.Context, since string) ([]*RepoInfo, error) {
	req, err := newRequest(ctx, "https://github.com/trending?since="+since)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github trending returned %d", resp.StatusCode)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	var repos []*RepoInfo
	doc.Find("article.Box-row").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Find("h2 a").First().Attr("href")
		parts := strings.Split(strings.Trim(href, "/"), "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return
		}
		repos = append(repos, &RepoInfo{
			Platform:    c.Name(),
			Owner:       parts[0],
			Name:        parts[1],
			URL:         "https://github.com/" + parts[0] + "/" + parts[1],
			Description: strings.TrimSpace(s.Find("p.col-9").First().Text()),
			Language:    strings.TrimSpace(s.Find(`span[itemprop="programmingLanguage"]`).First().Text()),
			Stars:       parseNumber(s.Find(`a[href$="/stargazers"]`).First().Text()),
			Forks:       parseNumber(s.Find(`a[href$="/forks"]`).First().Text()),
		})
	})
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repos parsed from github trending page (since=%s)", since)
	}
	return repos, nil
}
