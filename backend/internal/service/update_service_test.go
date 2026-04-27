package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type updateCacheStub struct {
	data string
	err  error
}

func (s *updateCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.data, nil
}

func (s *updateCacheStub) SetUpdateInfo(context.Context, string, time.Duration) error {
	return nil
}

type githubReleaseClientStub struct {
	release *GitHubRelease
	err     error
	repo    string
}

func (s *githubReleaseClientStub) FetchLatestRelease(_ context.Context, repo string) (*GitHubRelease, error) {
	s.repo = repo
	if s.err != nil {
		return nil, s.err
	}
	return s.release, nil
}

func (s *githubReleaseClientStub) DownloadFile(context.Context, string, string, int64) error {
	return nil
}

func (s *githubReleaseClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	return nil, nil
}

func TestCheckUpdateFetchFailureDoesNotReportCurrentAsLatest(t *testing.T) {
	cache := &updateCacheStub{err: errors.New("cache miss")}
	client := &githubReleaseClientStub{err: errors.New("GitHub API returned 404")}
	svc := NewUpdateService(cache, client, "0.2.120", "release")

	info, err := svc.CheckUpdate(context.Background(), true)
	if err != nil {
		t.Fatalf("CheckUpdate returned error: %v", err)
	}

	if info.CurrentVersion != "0.2.120" {
		t.Fatalf("CurrentVersion = %q, want 0.2.120", info.CurrentVersion)
	}
	if info.LatestVersion != "" {
		t.Fatalf("LatestVersion = %q, want empty when latest release cannot be fetched", info.LatestVersion)
	}
	if info.HasUpdate {
		t.Fatal("HasUpdate = true, want false when latest release cannot be fetched")
	}
	if info.Warning == "" {
		t.Fatal("Warning is empty, want fetch error warning")
	}
}

func TestFetchLatestReleaseUsesConfiguredRepo(t *testing.T) {
	cache := &updateCacheStub{err: errors.New("cache miss")}
	client := &githubReleaseClientStub{release: &GitHubRelease{TagName: "v0.2.121"}}
	svc := NewUpdateServiceWithRepo(cache, client, "0.2.120", "release", "Rogers-F/sub2api-R")

	_, err := svc.CheckUpdate(context.Background(), true)
	if err != nil {
		t.Fatalf("CheckUpdate returned error: %v", err)
	}

	if client.repo != "Rogers-F/sub2api-R" {
		t.Fatalf("repo = %q, want Rogers-F/sub2api-R", client.repo)
	}
}
