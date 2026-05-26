package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type handlerBatchAPIKeyRepoStub struct {
	service.APIKeyRepository
	ownedIDs       []int64
	keysByID       map[int64]string
	affected       int64
	updatedIDs     []int64
	updatedGroupID int64
}

func (s *handlerBatchAPIKeyRepoStub) VerifyOwnership(context.Context, int64, []int64) ([]int64, error) {
	return append([]int64(nil), s.ownedIDs...), nil
}

func (s *handlerBatchAPIKeyRepoStub) ListKeysByUserAndIDs(_ context.Context, _ int64, ids []int64) ([]string, error) {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		if key, ok := s.keysByID[id]; ok {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *handlerBatchAPIKeyRepoStub) BatchUpdateGroupIDByUserAndIDs(_ context.Context, _ int64, ids []int64, groupID int64) (int64, error) {
	s.updatedIDs = append([]int64(nil), ids...)
	s.updatedGroupID = groupID
	return s.affected, nil
}

type handlerBatchUserRepoStub struct {
	service.UserRepository
	user *service.User
}

func (s *handlerBatchUserRepoStub) GetByID(context.Context, int64) (*service.User, error) {
	clone := *s.user
	return &clone, nil
}

type handlerBatchGroupRepoStub struct {
	service.GroupRepository
	group *service.Group
}

func (s *handlerBatchGroupRepoStub) GetByID(context.Context, int64) (*service.Group, error) {
	clone := *s.group
	return &clone, nil
}

func setupBatchAPIKeyHandlerTest() (*gin.Engine, *handlerBatchAPIKeyRepoStub) {
	gin.SetMode(gin.TestMode)
	repo := &handlerBatchAPIKeyRepoStub{
		ownedIDs: []int64{1, 2},
		keysByID: map[int64]string{1: "sk-one", 2: "sk-two"},
		affected: 2,
	}
	apiKeyService := service.NewAPIKeyService(
		repo,
		&handlerBatchUserRepoStub{user: &service.User{ID: 7, AllowedGroups: []int64{10}}},
		&handlerBatchGroupRepoStub{group: &service.Group{ID: 10, Status: service.StatusActive}},
		nil,
		nil,
		nil,
		nil,
	)
	handler := NewAPIKeyHandler(apiKeyService)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
		c.Next()
	})
	router.POST("/api/v1/keys/batch/group", handler.BatchUpdateGroup)
	return router, repo
}

func TestAPIKeyHandler_BatchUpdateGroup_Success(t *testing.T) {
	router, repo := setupBatchAPIKeyHandlerTest()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys/batch/group", bytes.NewBufferString(`{"ids":[1,2],"group_id":10}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{1, 2}, repo.updatedIDs)
	require.Equal(t, int64(10), repo.updatedGroupID)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Updated int64 `json:"updated"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(2), resp.Data.Updated)
}

func TestAPIKeyHandler_BatchUpdateGroup_InvalidJSON(t *testing.T) {
	router, _ := setupBatchAPIKeyHandlerTest()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys/batch/group", bytes.NewBufferString(`{bad json`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Invalid request")
}
