package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
