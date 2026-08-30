package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// GitLabCollector GitLab 采集器。
type GitLabCollector struct{}

// NewGitLab 创建 GitLab 采集器。
func NewGitLab() *GitLabCollector {
	return &GitLabCollector{}
}

// Name 平台名。
func (c *GitLabCollector) Name() string { return "gitlab" }

// Match 判断是否 gitlab.com 链接。
func (c *GitLabCollector) Match(rawurl string) bool {
	return matchHost(rawurl, "gitlab.com")
}

type gitlabProjectJSON struct {
	PathWithNamespace string `json:"path_with_namespace"`
	Description       string `json:"description"`
	Stars             int    `json:"star_count"`
	Forks             int    `json:"forks_count"`
	WebURL            string `json:"web_url"`
	License           *struct {
		Name string `json:"name"`
	} `json:"license"`
}

// FetchRepo 调用 GitLab API v4 抓取仓库详情。
func (c *GitLabCollector) FetchRepo(ctx context.Context, owner, name string) (*RepoInfo, error) {
	projectPath := url.PathEscape(owner + "/" + name)
	req, err := newRequest(ctx, "https://gitlab.com/api/v4/projects/"+projectPath)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gitlab api returned %d", resp.StatusCode)
	}
	var data gitlabProjectJSON
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	fullPath := data.PathWithNamespace
	if fullPath == "" {
		fullPath = owner + "/" + name
	}
	o, n := owner, name
	if idx := strings.LastIndex(fullPath, "/"); idx > 0 {
		o, n = fullPath[:idx], fullPath[idx+1:]
	}
	info := &RepoInfo{
		Platform:    c.Name(),
		Owner:       o,
		Name:        n,
		URL:         data.WebURL,
		Description: data.Description,
		Stars:       data.Stars,
		Forks:       data.Forks,
	}
	if info.URL == "" {
		info.URL = "https://gitlab.com/" + fullPath
	}
	if data.License != nil {
		info.License = data.License.Name
	}
	return info, nil
}
