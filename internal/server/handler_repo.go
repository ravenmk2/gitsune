package server

import (
	"database/sql"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitsune/internal/model"
	"gitsune/internal/platform"
)

type collectRepoRequest struct {
	URL string `json:"url"`
}

// collectRepo POST /api/repo/collect：识别平台抓取详情后 upsert（source=manual）。
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
	Page     int    `json:"page"`
	Size     int    `json:"size"`
	Platform string `json:"platform"`
	Keyword  string `json:"keyword"`
	Language string `json:"language"`
	Source   string `json:"source"`
}

// listRepos POST /api/repo/list：分页 + 筛选，按 stars 倒序。
func (s *Server) listRepos(c *gin.Context) {
	var req listReposRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	pageNum, size := normalizePage(req.Page, req.Size)
	items, total, err := s.store.ListRepos(pageNum, size, req.Platform, req.Keyword, req.Language, req.Source)
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
