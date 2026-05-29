//go:build unit

// Unit tests for the persisted multi-conversation chat feature.
//
// These exercise the real ConversationService on top of the real Ent-backed
// repositories using an in-memory SQLite database (via enttest). They cover:
//   - IDOR: a user cannot Get/Update/Delete/ListMessages/Append another user's
//     conversation (expect not-found 404).
//   - Idempotency: same client_message_id + same payload returns the existing
//     message; a different payload yields a 409 conflict.
//   - Ownership 404 on a missing/foreign id.
//   - Pagination cursor correctness.
//   - Rejection of role='system'.
//   - Rejection of a complete message with blank content.
//
// NOTE on CHECK constraints: enttest derives the SQLite schema from the Ent
// schema, which does not include the SQL CHECK constraints (role/status/content
// rules). Those rules are validated in the service layer (and additionally
// enforced by the authoritative SQL migration in Postgres), so the
// service-level validation tests below assert the service behavior directly.

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newConversationStack(t *testing.T) (*service.ConversationService, *dbent.Client) {
	t.Helper()

	// Unique in-memory DB name per test to avoid shared-cache cross-talk.
	dsn := fmt.Sprintf("file:conv_%s?mode=memory&cache=shared", t.Name())
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	// Pin to a single connection so the in-memory database is stable across the
	// schema migration and subsequent reads/writes (avoids per-connection DBs).
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	convRepo := NewConversationRepository(client)
	msgRepo := NewMessageRepository(client)
	svc := service.NewConversationService(client, convRepo, msgRepo)
	return svc, client
}

func mustCreateConvUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) int64 {
	t.Helper()
	u, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("test-password-hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)
	return u.ID
}

func completeUserMessage(clientMessageID, content string) service.MessageInput {
	return service.MessageInput{
		Role:            service.MessageRoleUser,
		Content:         content,
		Status:          service.MessageStatusComplete,
		ClientMessageID: clientMessageID,
	}
}

func TestConversationService_CreateIsIdempotent(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "create@test.com")

	first, err := svc.CreateConversation(ctx, userID, "client-1", "Hello", "model-x")
	require.NoError(t, err)
	require.NotZero(t, first.ID)

	// Same client_conversation_id returns the existing record (same id).
	second, err := svc.CreateConversation(ctx, userID, "client-1", "Different title", "model-y")
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, "Hello", second.Title) // original preserved
}

func TestConversationService_IDOR(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	alice := mustCreateConvUser(t, ctx, client, "alice@test.com")
	bob := mustCreateConvUser(t, ctx, client, "bob@test.com")

	owned, err := svc.CreateConversation(ctx, alice, "alice-conv", "Alice", "m")
	require.NoError(t, err)

	// Bob cannot Get Alice's conversation.
	_, err = svc.GetConversation(ctx, bob, owned.ID)
	require.True(t, infraerrors.IsNotFound(err), "Get should be 404 for foreign owner")

	// Bob cannot Update title.
	_, err = svc.UpdateTitle(ctx, bob, owned.ID, "hacked")
	require.True(t, infraerrors.IsNotFound(err), "Update should be 404 for foreign owner")

	// Bob cannot Delete.
	err = svc.DeleteConversation(ctx, bob, owned.ID)
	require.True(t, infraerrors.IsNotFound(err), "Delete should be 404 for foreign owner")

	// Bob cannot ListMessages.
	_, err = svc.ListMessages(ctx, bob, owned.ID, 0, 50)
	require.True(t, infraerrors.IsNotFound(err), "ListMessages should be 404 for foreign owner")

	// Bob cannot Append.
	_, err = svc.AppendMessages(ctx, bob, owned.ID, []service.MessageInput{
		completeUserMessage("m1", "hi"),
	})
	require.True(t, infraerrors.IsNotFound(err), "Append should be 404 for foreign owner")

	// Alice's data is untouched.
	got, err := svc.GetConversation(ctx, alice, owned.ID)
	require.NoError(t, err)
	require.Equal(t, "Alice", got.Title)
}

func TestConversationService_OwnershipNotFound(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "nf@test.com")

	_, err := svc.GetConversation(ctx, userID, 999999)
	require.True(t, infraerrors.IsNotFound(err))

	_, err = svc.ListMessages(ctx, userID, 999999, 0, 50)
	require.True(t, infraerrors.IsNotFound(err))

	_, err = svc.AppendMessages(ctx, userID, 999999, []service.MessageInput{
		completeUserMessage("m1", "hi"),
	})
	require.True(t, infraerrors.IsNotFound(err))
}

func TestConversationService_AppendIdempotency(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "idem@test.com")

	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)

	first, err := svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{
		completeUserMessage("msg-1", "hello world"),
	})
	require.NoError(t, err)
	require.Len(t, first, 1)
	firstID := first[0].ID

	// Same client_message_id + same payload -> returns existing (no error, same id).
	again, err := svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{
		completeUserMessage("msg-1", "hello world"),
	})
	require.NoError(t, err)
	require.Len(t, again, 1)
	require.Equal(t, firstID, again[0].ID)

	// Same client_message_id + different payload -> 409 conflict.
	_, err = svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{
		completeUserMessage("msg-1", "different content"),
	})
	require.True(t, infraerrors.IsConflict(err), "expected 409 conflict on differing payload")
}

func TestConversationService_IdempotentRetryDoesNotTouch(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "notouch@test.com")
	convRepo := NewConversationRepository(client)

	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)
	_, err = svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{
		completeUserMessage("msg-1", "hello"),
	})
	require.NoError(t, err)

	// Pin updated_at to a known value, then issue a pure idempotent retry.
	pinned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, convRepo.Touch(ctx, userID, conv.ID, pinned))

	_, err = svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{
		completeUserMessage("msg-1", "hello"),
	})
	require.NoError(t, err)

	after, err := svc.GetConversation(ctx, userID, conv.ID)
	require.NoError(t, err)
	require.WithinDuration(t, pinned, after.UpdatedAt, time.Second,
		"pure idempotent retry must not bump updated_at")
}

func TestConversationService_DeleteCascadesMessages(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "cascade@test.com")

	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)
	_, err = svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{
		completeUserMessage("m1", "a"),
		completeUserMessage("m2", "b"),
	})
	require.NoError(t, err)

	// Sanity: messages exist before delete.
	before, err := client.ConversationMessage.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, before, 2)

	require.NoError(t, svc.DeleteConversation(ctx, userID, conv.ID))

	// Conversation is gone.
	_, err = svc.GetConversation(ctx, userID, conv.ID)
	require.True(t, infraerrors.IsNotFound(err))

	// Messages were cascade-deleted.
	after, err := client.ConversationMessage.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 0, "ON DELETE CASCADE should remove child messages")
}

func TestConversationService_RejectDuplicateInBatch(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "batch@test.com")
	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)

	_, err = svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{
		completeUserMessage("dup", "a"),
		completeUserMessage("dup", "b"),
	})
	require.True(t, infraerrors.IsBadRequest(err), "expected 400 on duplicate client_message_id in batch")
}

func TestConversationService_RejectSystemRole(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "sys@test.com")
	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)

	_, err = svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{
		{
			Role:            "system",
			Content:         "you are helpful",
			Status:          service.MessageStatusComplete,
			ClientMessageID: "sys-1",
		},
	})
	require.True(t, infraerrors.IsBadRequest(err), "expected 400 rejecting role=system")
}

func TestConversationService_RejectBlankCompleteContent(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "blank@test.com")
	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)

	_, err = svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{
		{
			Role:            service.MessageRoleAssistant,
			Content:         "   ",
			Status:          service.MessageStatusComplete,
			ClientMessageID: "blank-1",
		},
	})
	require.True(t, infraerrors.IsBadRequest(err), "expected 400 for blank complete content")
}

func TestConversationService_RejectReportedTokenOverflow(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "tok@test.com")
	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)

	over := math.MaxInt32 + 1
	msg := completeUserMessage("tok-1", "hello")
	msg.ReportedInputTokens = &over

	_, err = svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{msg})
	require.True(t, infraerrors.IsBadRequest(err),
		"expected 400 when reported token exceeds int4 range")

	// MaxInt32 is accepted (boundary).
	atMax := math.MaxInt32
	ok := completeUserMessage("tok-2", "hello")
	ok.ReportedOutputTokens = &atMax
	_, err = svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{ok})
	require.NoError(t, err)
}

func TestConversationService_ListPaginationCursor(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "page@test.com")

	// Create 5 conversations and assign each a distinct, well-separated
	// updated_at. SQLite stores timestamps with coarse precision, so creating in
	// a tight loop yields near-identical values; assigning explicit, spaced
	// timestamps makes the (updated_at DESC, id DESC) ordering unambiguous and
	// lets the composite-cursor logic be validated deterministically.
	convRepo := NewConversationRepository(client)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ids := make([]int64, 0, 5)
	for i := 0; i < 5; i++ {
		conv, err := svc.CreateConversation(ctx, userID, fmt.Sprintf("c-%d", i), "T", "m")
		require.NoError(t, err)
		require.NoError(t, convRepo.Touch(ctx, userID, conv.ID, base.Add(time.Duration(i)*time.Hour)))
		ids = append(ids, conv.ID)
	}

	// First page of 2.
	page1, err := svc.ListConversations(ctx, userID, nil, 2)
	require.NoError(t, err)
	require.Len(t, page1.Items, 2)
	require.NotNil(t, page1.NextCursor, "expected a next cursor for page 1")

	// Second page of 2 using the cursor.
	page2, err := svc.ListConversations(ctx, userID, page1.NextCursor, 2)
	require.NoError(t, err)
	require.Len(t, page2.Items, 2)
	require.NotNil(t, page2.NextCursor)

	// Third page: 1 remaining, no further cursor.
	page3, err := svc.ListConversations(ctx, userID, page2.NextCursor, 2)
	require.NoError(t, err)
	require.Len(t, page3.Items, 1)
	require.Nil(t, page3.NextCursor, "expected no cursor on the final page")

	// All ids are distinct and there are exactly 5 across pages.
	seen := map[int64]struct{}{}
	for _, pg := range [][]service.Conversation{page1.Items, page2.Items, page3.Items} {
		for i := range pg {
			seen[pg[i].ID] = struct{}{}
		}
	}
	require.Len(t, seen, 5)
}
