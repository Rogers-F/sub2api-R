package handler

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

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
		UpdatedAt: time.Date(2026, 1, 2, 3, 4, 5, 600000000, time.UTC),
		ID:        123,
	}
	token := encodeConversationCursor(cur)
	require.NotEmpty(t, token)

	decoded, err := decodeConversationCursor(token)
	require.NoError(t, err)
	require.NotNil(t, decoded)
	require.Equal(t, cur.ID, decoded.ID)
	require.True(t, cur.UpdatedAt.Equal(decoded.UpdatedAt))

	// Empty token -> nil cursor (first page).
	none, err := decodeConversationCursor("")
	require.NoError(t, err)
	require.Nil(t, none)

	// Malformed base64 -> error.
	_, err = decodeConversationCursor("!!!not-base64!!!")
	require.Error(t, err)
}
