package server

import (
	"database/sql"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitsune/internal/model"
	"gitsune/internal/platform"
)

type collectRepoRequest struct {
	URL string `json:"url"`
}

// collectRepo POST /api/repo/collect：识别平台抓取详情后录入（source=manual）；已存在的仓库不覆盖，直接返回现有记录。
func (s *Server) collectRepo(c *gin.Context) {
	var req collectRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		fail(c, CodeValidationError, "url is required")
		return
	}
	collector := s.registry.Match(req.URL)
	if collector == nil {
		fail(c, CodePlatformNotSupport, "unsupported repository platform")
		return
	}
	owner, name, err := platform.ParseRepoPath(req.URL)
	if err != nil {
		fail(c, CodeValidationError, "cannot parse owner/name from url")
		return
	}
	// 已存在的仓库不覆盖，直接返回现有记录（数据更新只走 repo/refresh 手动刷新）
	if existing, err := s.store.GetRepoByKey(collector.Name(), owner, name); err == nil {
		ok(c, existing)
		return
	} else if err != sql.ErrNoRows {
		logrus.WithError(err).Error("repo/collect: failed to query repo")
		fail(c, CodeInternalError, "failed to query repos")
		return
	}
	repo, err := s.fetchAndUpsert(c, collector, owner, name, model.SourceManual)
	if err != nil {
		logrus.WithError(err).Warnf("repo/collect: failed to fetch repo details: %s", req.URL)
		fail(c, CodeInternalError, "failed to fetch repo details: "+err.Error())
		return
	}
	ok(c, repo)
}

// fetchAndUpsert 抓详情、upsert 并返回最新 repo 记录。
func (s *Server) fetchAndUpsert(c *gin.Context, collector platform.Collector, owner, name, source string) (*model.Repo, error) {
	info, err := collector.FetchRepo(c.Request.Context(), owner, name)
	if err != nil {
		return nil, err
	}
	id, _, err := s.store.UpsertRepo(&model.Repo{
		Platform:    info.Platform,
		Owner:       info.Owner,
		Name:        info.Name,
		URL:         info.URL,
		Description: info.Description,
		Language:    info.Language,
		Stars:       info.Stars,
		Forks:       info.Forks,
		License:     info.License,
		Source:      source,
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetRepoByID(id)
}

type listReposRequest struct {
	Page      int    `json:"page"`
	Size      int    `json:"size"`
	Platform  string `json:"platform"`
	Keyword   string `json:"keyword"`
	Language  string `json:"language"`
	Source    string `json:"source"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"`
}

// listRepos POST /api/repo/list：分页 + 筛选 + 排序（sort_by: id/stars/forks，sort_order: asc/desc，默认 id 倒序）。
func (s *Server) listRepos(c *gin.Context) {
	var req listReposRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	pageNum, size := normalizePage(req.Page, req.Size)
	items, total, err := s.store.ListRepos(pageNum, size, req.Platform, req.Keyword, req.Language, req.Source, req.SortBy, req.SortOrder)
	if err != nil {
		fail(c, CodeInternalError, "failed to query repos")
		return
	}
	page(c, items, pageNum, size, total)
}

type repoIDRequest struct {
	ID int64 `json:"id"`
}

// getRepo POST /api/repo/get
func (s *Server) getRepo(c *gin.Context) {
	var req repoIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	repo, err := s.store.GetRepoByID(req.ID)
	if err == sql.ErrNoRows {
		fail(c, CodeRepoNotFound, "repo not found")
		return
	}
	if err != nil {
		fail(c, CodeInternalError, "failed to query repos")
		return
	}
	ok(c, repo)
}

// refreshRepo POST /api/repo/refresh（admin）：重新抓取详情更新。
func (s *Server) refreshRepo(c *gin.Context) {
	var req repoIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	repo, err := s.store.GetRepoByID(req.ID)
	if err == sql.ErrNoRows {
		fail(c, CodeRepoNotFound, "repo not found")
		return
	}
	if err != nil {
		fail(c, CodeInternalError, "failed to query repos")
		return
	}
	collector := s.registry.Get(repo.Platform)
	if collector == nil {
		fail(c, CodePlatformNotSupport, "unsupported repository platform")
		return
	}
	updated, err := s.fetchAndUpsert(c, collector, repo.Owner, repo.Name, repo.Source)
	if err != nil {
		logrus.WithError(err).Warnf("repo/refresh: failed to fetch repo details: %s/%s", repo.Owner, repo.Name)
		fail(c, CodeInternalError, "failed to fetch repo details: "+err.Error())
		return
	}
	ok(c, updated)
}

// deleteRepo POST /api/repo/delete（admin）
func (s *Server) deleteRepo(c *gin.Context) {
	var req repoIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	if _, err := s.store.GetRepoByID(req.ID); err == sql.ErrNoRows {
		fail(c, CodeRepoNotFound, "repo not found")
		return
	} else if err != nil {
		fail(c, CodeInternalError, "failed to delete repo")
		return
	}
	if err := s.store.DeleteRepo(req.ID); err != nil {
		fail(c, CodeInternalError, "failed to delete repo")
		return
	}
	ok(c, nil)
}

// repoTransferItem 导出/导入共用的仓库条目（不含 id/platform/created_at）。
type repoTransferItem struct {
	Owner        string `json:"owner"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	Description  string `json:"description"`
	Language     string `json:"language"`
	Stars        int    `json:"stars"`
	Forks        int    `json:"forks"`
	License      string `json:"license"`
	Source       string `json:"source"`
	LastSyncedAt string `json:"last_synced_at"`
}

func toTransferItem(r *model.Repo) repoTransferItem {
	return repoTransferItem{
		Owner:        r.Owner,
		Name:         r.Name,
		URL:          r.URL,
		Description:  r.Description,
		Language:     r.Language,
		Stars:        r.Stars,
		Forks:        r.Forks,
		License:      r.License,
		Source:       r.Source,
		LastSyncedAt: r.LastSyncedAt,
	}
}

type exportReposRequest struct {
	Platforms []string `json:"platforms"`
}

type exportReposResponse struct {
	Version    int                            `json:"version"`
	ExportedAt string                         `json:"exported_at"`
	Platforms  map[string][]repoTransferItem `json:"platforms"`
}

// exportRepos POST /api/repo/export：导出仓库为按平台分组的 JSON 文档。
func (s *Server) exportRepos(c *gin.Context) {
	var req exportReposRequest
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	repos, err := s.store.ListAll(req.Platforms)
	if err != nil {
		fail(c, CodeInternalError, "failed to query repos")
		return
	}
	// 被请求的平台都作为 key 出现（没有数据则空数组）；缺省时输出全部已注册平台
	keys := req.Platforms
	if len(keys) == 0 {
		keys = s.registry.Names()
	}
	groups := make(map[string][]repoTransferItem, len(keys))
	for _, p := range keys {
		groups[p] = []repoTransferItem{}
	}
	for i := range repos {
		groups[repos[i].Platform] = append(groups[repos[i].Platform], toTransferItem(&repos[i]))
	}
	ok(c, exportReposResponse{Version: 1, ExportedAt: model.NowUTC(), Platforms: groups})
}

// 导入模式
const (
	importModeIncremental = "incremental"
	importModeOverwrite   = "overwrite"
)

type importReposRequest struct {
	Mode      string                        `json:"mode"`
	Platforms map[string][]repoTransferItem `json:"platforms"`
}

type importFailedItem struct {
	Platform string `json:"platform"`
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	Reason   string `json:"reason"`
}

type importReposResponse struct {
	Added   int                `json:"added"`
	Updated int                `json:"updated"`
	Failed  []importFailedItem `json:"failed"`
}

// importRepos POST /api/repo/import（admin）：按模式增量 upsert 或覆盖重建仓库表。
func (s *Server) importRepos(c *gin.Context) {
	var req importReposRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	if req.Mode != importModeIncremental && req.Mode != importModeOverwrite {
		fail(c, CodeValidationError, "mode must be incremental or overwrite")
		return
	}
	repos := make([]model.Repo, 0)
	failed := make([]importFailedItem, 0)
	for platformName, items := range req.Platforms {
		if s.registry.Get(platformName) == nil {
			for _, item := range items {
				failed = append(failed, importFailedItem{
					Platform: platformName, Owner: item.Owner, Name: item.Name, Reason: "unknown platform",
				})
			}
			continue
		}
		for _, item := range items {
			owner := strings.TrimSpace(item.Owner)
			name := strings.TrimSpace(item.Name)
			if owner == "" || name == "" {
				failed = append(failed, importFailedItem{
					Platform: platformName, Owner: item.Owner, Name: item.Name, Reason: "owner and name are required",
				})
				continue
			}
			source := strings.TrimSpace(item.Source)
			if source == "" {
				source = model.SourceManual
			}
			repos = append(repos, model.Repo{
				Platform:     platformName,
				Owner:        owner,
				Name:         name,
				URL:          item.URL,
				Description:  item.Description,
				Language:     item.Language,
				Stars:        item.Stars,
				Forks:        item.Forks,
				License:      item.License,
				Source:       source,
				LastSyncedAt: item.LastSyncedAt,
			})
		}
	}

	var added, updated int
	var err error
	if req.Mode == importModeOverwrite {
		added, err = s.store.ReplaceAll(repos)
	} else {
		added, updated, err = s.store.UpsertMany(repos)
	}
	if err != nil {
		logrus.WithError(err).Error("repo/import: failed to write repos")
		fail(c, CodeInternalError, "failed to import repos")
		return
	}
	logrus.Infof("repo/import: mode=%s added=%d updated=%d failed=%d", req.Mode, added, updated, len(failed))
	ok(c, importReposResponse{Added: added, Updated: updated, Failed: failed})
}
