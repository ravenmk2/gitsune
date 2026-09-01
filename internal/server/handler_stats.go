package server

import (
	"github.com/gin-gonic/gin"

	"gitsune/internal/model"
	"gitsune/internal/store"
)

type statsOverviewResponse struct {
	TotalRepos  int               `json:"total_repos"`
	TotalStars  int               `json:"total_stars"`
	RecentAdded int               `json:"recent_added"`
	Platforms   []store.NameCount `json:"platforms"`
	Languages   []store.NameCount `json:"languages"`
	LatestRepos []model.Repo      `json:"latest_repos"`
	TopRepos    []model.Repo      `json:"top_repos"`
}

// statsOverview POST /api/stats/overview：首页概览统计（登录即可）。
func (s *Server) statsOverview(c *gin.Context) {
	total, totalStars, err := s.store.RepoTotals()
	if err != nil {
		fail(c, CodeInternalError, "failed to query stats")
		return
	}
	recent, err := s.store.CountReposSince(store.RecentCutoff(7))
	if err != nil {
		fail(c, CodeInternalError, "failed to query stats")
		return
	}
	platforms, err := s.store.CountReposByPlatform()
	if err != nil {
		fail(c, CodeInternalError, "failed to query stats")
		return
	}
	languages, err := s.store.CountReposByLanguage(5)
	if err != nil {
		fail(c, CodeInternalError, "failed to query stats")
		return
	}
	latest, err := s.store.ListLatestRepos(8)
	if err != nil {
		fail(c, CodeInternalError, "failed to query stats")
		return
	}
	top, err := s.store.ListTopRepos(10)
	if err != nil {
		fail(c, CodeInternalError, "failed to query stats")
		return
	}
	ok(c, statsOverviewResponse{
		TotalRepos:  total,
		TotalStars:  totalStars,
		RecentAdded: recent,
		Platforms:   platforms,
		Languages:   languages,
		LatestRepos: latest,
		TopRepos:    top,
	})
}
