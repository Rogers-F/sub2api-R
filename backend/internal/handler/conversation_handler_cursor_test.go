package handler

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// listMessagesRequest drives ListMessages with the given raw query string,
// returning the HTTP status. The 400 guards under test return before the
// service is touched, so a nil service is safe here.
func listMessagesRequest(t *testing.T, rawQuery string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewConversationHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1})
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/conversations/1/messages?"+rawQuery, nil)
	h.ListMessages(c)
	return w.Code
}

func TestListMessages_PaginationParamGuards(t *testing.T) {
	// cursor and before_id are mutually exclusive (presence-based) -> 400.
	require.Equal(t, http.StatusBadRequest, listMessagesRequest(t, "cursor=5&before_id=3"))
	// Both present even with an empty before_id value -> 400.
	require.Equal(t, http.StatusBadRequest, listMessagesRequest(t, "cursor=5&before_id="))
	// before_id present but empty/malformed -> 400 (not a silent fallback).
	require.Equal(t, http.StatusBadRequest, listMessagesRequest(t, "before_id="))
	require.Equal(t, http.StatusBadRequest, listMessagesRequest(t, "before_id=abc"))
	require.Equal(t, http.StatusBadRequest, listMessagesRequest(t, "before_id=-1"))
}

func TestParseAfterIDQuery(t *testing.T) {
	// Empty -> first page, no error.
	v, err := parseAfterIDQuery("")
	require.NoError(t, err)
	require.Equal(t, int64(0), v)

	// Valid positive id.
	v, err = parseAfterIDQuery(" 42 ")
	require.NoError(t, err)
	require.Equal(t, int64(42), v)

	// Malformed -> error (handler maps to 400).
	_, err = parseAfterIDQuery("abc")
	require.Error(t, err)

	// Negative -> error.
	_, err = parseAfterIDQuery("-1")
	require.Error(t, err)
}

func TestConversationCursorRoundTrip(t *testing.T) {
	cur := &service.ConversationCursor{
		LastMessageAt: time.Date(2026, 1, 2, 3, 4, 5, 600000000, time.UTC),
		ID:            123,
	}
	token := encodeConversationCursor(cur)
	require.NotEmpty(t, token)

	decoded, err := decodeConversationCursor(token)
	require.NoError(t, err)
	require.NotNil(t, decoded)
	require.Equal(t, cur.ID, decoded.ID)
	require.True(t, cur.LastMessageAt.Equal(decoded.LastMessageAt))

	// Empty token -> nil cursor (first page).
	none, err := decodeConversationCursor("")
	require.NoError(t, err)
	require.Nil(t, none)

	// Malformed base64 -> error.
	_, err = decodeConversationCursor("!!!not-base64!!!")
	require.Error(t, err)

	// A legacy v1 token (no version prefix: "<nanos>:<id>") must be rejected so
	// it is never silently misinterpreted as a v2 (last_message_at) cursor.
	v1 := base64.RawURLEncoding.EncodeToString(
		[]byte(strconv.FormatInt(cur.LastMessageAt.UTC().UnixNano(), 10) + ":" + strconv.FormatInt(cur.ID, 10)),
	)
	_, err = decodeConversationCursor(v1)
	require.Error(t, err)
}
