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

func completeAssistantMessage(clientMessageID, content string) service.MessageInput {
	return service.MessageInput{
		Role:            service.MessageRoleAssistant,
		Content:         content,
		Status:          service.MessageStatusComplete,
		ClientMessageID: clientMessageID,
	}
}

// appendTurn appends a [user, assistant] pair and returns the assistant message.
func appendTurn(t *testing.T, ctx context.Context, svc *service.ConversationService, userID, convID int64, suffix, userText, asstText string) service.Message {
	t.Helper()
	out, err := svc.AppendMessages(ctx, userID, convID, []service.MessageInput{
		completeUserMessage("u-"+suffix, userText),
		completeAssistantMessage("a-"+suffix, asstText),
	})
	require.NoError(t, err)
	require.Len(t, out, 2)
	return out[1]
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

	// Pin last_message_at (the ordering key) to a known value, then issue a pure
	// idempotent retry.
	pinned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, convRepo.Touch(ctx, userID, conv.ID, pinned))

	_, err = svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{
		completeUserMessage("msg-1", "hello"),
	})
	require.NoError(t, err)

	after, err := svc.GetConversation(ctx, userID, conv.ID)
	require.NoError(t, err)
	require.WithinDuration(t, pinned, after.LastMessageAt, time.Second,
		"pure idempotent retry must not advance last_message_at")
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

// ==================== Phase 2: last_message_at / rename ordering ====================

func TestConversationService_RenameDoesNotReorder(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "rename@test.com")
	convRepo := NewConversationRepository(client)

	convA, err := svc.CreateConversation(ctx, userID, "a", "A", "m")
	require.NoError(t, err)
	convB, err := svc.CreateConversation(ctx, userID, "b", "B", "m")
	require.NoError(t, err)

	// B has the more recent last_message_at, so it sorts ahead of A.
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, convRepo.Touch(ctx, userID, convA.ID, base.Add(1*time.Hour)))
	require.NoError(t, convRepo.Touch(ctx, userID, convB.ID, base.Add(2*time.Hour)))

	page, err := svc.ListConversations(ctx, userID, nil, 10)
	require.NoError(t, err)
	require.Equal(t, convB.ID, page.Items[0].ID, "B should be first before rename")

	// Renaming A bumps its updated_at but MUST NOT advance last_message_at, so
	// the order is unchanged (B still first).
	_, err = svc.UpdateTitle(ctx, userID, convA.ID, "A renamed")
	require.NoError(t, err)

	page2, err := svc.ListConversations(ctx, userID, nil, 10)
	require.NoError(t, err)
	require.Equal(t, convB.ID, page2.Items[0].ID, "rename must not move A to the top")
}

func TestConversationService_EmptyConversationLastMessageAtEqualsCreatedAt(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "emptyinv@test.com")

	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)

	got, err := svc.GetConversation(ctx, userID, conv.ID)
	require.NoError(t, err)
	require.True(t, got.LastMessageAt.Equal(got.CreatedAt),
		"a conversation with no messages must have last_message_at == created_at")
}

func TestConversationService_LastMessageAtTracksNewestMessage(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "lma@test.com")

	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)
	asst := appendTurn(t, ctx, svc, userID, conv.ID, "1", "hi", "hello there")

	got, err := svc.GetConversation(ctx, userID, conv.ID)
	require.NoError(t, err)
	require.WithinDuration(t, asst.CreatedAt, got.LastMessageAt, time.Second,
		"last_message_at must equal the newest message's created_at")
}

// ==================== Phase 2: backward (newest-first) message pagination ====================

func TestConversationService_ListMessagesBefore(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "before@test.com")

	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)

	// 5 messages, ids ascending in insertion order.
	inserted, err := svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{
		completeUserMessage("m1", "1"),
		completeUserMessage("m2", "2"),
		completeUserMessage("m3", "3"),
		completeUserMessage("m4", "4"),
		completeUserMessage("m5", "5"),
	})
	require.NoError(t, err)
	require.Len(t, inserted, 5)

	// Latest page (before_id == 0), limit 2 -> newest two (m4, m5) in id ASC, more older.
	latest, err := svc.ListMessagesBefore(ctx, userID, conv.ID, 0, 2)
	require.NoError(t, err)
	require.Len(t, latest.Items, 2)
	require.True(t, latest.HasMore)
	require.Equal(t, inserted[3].ID, latest.Items[0].ID)
	require.Equal(t, inserted[4].ID, latest.Items[1].ID)

	// Older page before the smallest id of the latest page (m4) -> m2, m3.
	older, err := svc.ListMessagesBefore(ctx, userID, conv.ID, latest.Items[0].ID, 2)
	require.NoError(t, err)
	require.Len(t, older.Items, 2)
	require.True(t, older.HasMore)
	require.Equal(t, inserted[1].ID, older.Items[0].ID)
	require.Equal(t, inserted[2].ID, older.Items[1].ID)

	// Final older page -> just m1, no more.
	last, err := svc.ListMessagesBefore(ctx, userID, conv.ID, older.Items[0].ID, 2)
	require.NoError(t, err)
	require.Len(t, last.Items, 1)
	require.False(t, last.HasMore)
	require.Equal(t, inserted[0].ID, last.Items[0].ID)

	// Empty conversation -> empty page, no more.
	empty, err := svc.CreateConversation(ctx, userID, "empty", "E", "m")
	require.NoError(t, err)
	page, err := svc.ListMessagesBefore(ctx, userID, empty.ID, 0, 10)
	require.NoError(t, err)
	require.Len(t, page.Items, 0)
	require.False(t, page.HasMore)

	// Negative before_id -> bad request.
	_, err = svc.ListMessagesBefore(ctx, userID, conv.ID, -1, 10)
	require.True(t, infraerrors.IsBadRequest(err))

	// Foreign owner -> 404.
	bob := mustCreateConvUser(t, ctx, client, "before-bob@test.com")
	_, err = svc.ListMessagesBefore(ctx, bob, conv.ID, 0, 10)
	require.True(t, infraerrors.IsNotFound(err))
}

// ==================== Phase 2: regenerate / atomic replace ====================

func TestConversationService_ReplaceSuccess(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "rep@test.com")

	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)
	asst := appendTurn(t, ctx, svc, userID, conv.ID, "1", "question", "first answer")

	newMsg := completeAssistantMessage("a-regen", "regenerated answer")
	replaced, err := svc.ReplaceMessageFrom(ctx, userID, conv.ID, asst.ID, "", newMsg)
	require.NoError(t, err)
	require.Equal(t, "regenerated answer", replaced.Content)
	require.Greater(t, replaced.ID, asst.ID, "replacement gets a fresh, larger id")

	// Old assistant is gone; the conversation now holds [user, regenerated].
	all, err := svc.ListMessagesBefore(ctx, userID, conv.ID, 0, 100)
	require.NoError(t, err)
	require.Len(t, all.Items, 2)
	require.Equal(t, service.MessageRoleUser, all.Items[0].Role)
	require.Equal(t, replaced.ID, all.Items[1].ID)

	// last_message_at tracks the new message.
	got, err := svc.GetConversation(ctx, userID, conv.ID)
	require.NoError(t, err)
	require.WithinDuration(t, replaced.CreatedAt, got.LastMessageAt, time.Second)
}

func TestConversationService_ReplaceResolvesByClientMessageID(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "repcid@test.com")

	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)
	_ = appendTurn(t, ctx, svc, userID, conv.ID, "1", "q", "a")

	// Resolve the cutoff by its client_message_id ("a-1"), without a server id.
	newMsg := completeAssistantMessage("a-regen", "regen via client id")
	replaced, err := svc.ReplaceMessageFrom(ctx, userID, conv.ID, 0, "a-1", newMsg)
	require.NoError(t, err)
	require.Equal(t, "regen via client id", replaced.Content)
}

func TestConversationService_ReplaceIDOR(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	alice := mustCreateConvUser(t, ctx, client, "rep-alice@test.com")
	bob := mustCreateConvUser(t, ctx, client, "rep-bob@test.com")

	conv, err := svc.CreateConversation(ctx, alice, "c", "T", "m")
	require.NoError(t, err)
	asst := appendTurn(t, ctx, svc, alice, conv.ID, "1", "q", "a")

	// Bob cannot replace within Alice's conversation.
	_, err = svc.ReplaceMessageFrom(ctx, bob, conv.ID, asst.ID, "", completeAssistantMessage("x", "evil"))
	require.True(t, infraerrors.IsNotFound(err))

	// Alice's assistant is untouched.
	all, err := svc.ListMessagesBefore(ctx, alice, conv.ID, 0, 100)
	require.NoError(t, err)
	require.Len(t, all.Items, 2)
	require.Equal(t, asst.ID, all.Items[1].ID)
}

func TestConversationService_ReplaceForeignCutoffRejected(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "repforeign@test.com")

	conv1, err := svc.CreateConversation(ctx, userID, "c1", "T", "m")
	require.NoError(t, err)
	conv2, err := svc.CreateConversation(ctx, userID, "c2", "T", "m")
	require.NoError(t, err)
	_ = appendTurn(t, ctx, svc, userID, conv1.ID, "1", "q1", "a1")
	other := appendTurn(t, ctx, svc, userID, conv2.ID, "2", "q2", "a2")

	// A cutoff id belonging to conv2 must not resolve within conv1 (no cross-conv
	// truncation, no accidental whole-conversation wipe).
	_, err = svc.ReplaceMessageFrom(ctx, userID, conv1.ID, other.ID, "", completeAssistantMessage("x", "y"))
	require.True(t, infraerrors.IsNotFound(err))
}

func TestConversationService_ReplaceCutoffMustBeLastAssistant(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "replast@test.com")

	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)
	asst1 := appendTurn(t, ctx, svc, userID, conv.ID, "1", "q1", "a1")
	_ = appendTurn(t, ctx, svc, userID, conv.ID, "2", "q2", "a2") // a2 is now last

	// Replacing a1 (no longer the tail) is refused with a conflict.
	_, err = svc.ReplaceMessageFrom(ctx, userID, conv.ID, asst1.ID, "", completeAssistantMessage("x", "y"))
	require.True(t, infraerrors.IsConflict(err))

	// A user message as cutoff is also refused (not an assistant).
	out, err := svc.AppendMessages(ctx, userID, conv.ID, []service.MessageInput{completeUserMessage("u3", "q3")})
	require.NoError(t, err)
	_, err = svc.ReplaceMessageFrom(ctx, userID, conv.ID, out[0].ID, "", completeAssistantMessage("z", "w"))
	require.True(t, infraerrors.IsConflict(err))
}

func TestConversationService_ReplaceRejectsBadCutoffArgs(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "repargs@test.com")
	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)
	asst := appendTurn(t, ctx, svc, userID, conv.ID, "1", "q", "a")

	// Both identifiers -> 400.
	_, err = svc.ReplaceMessageFrom(ctx, userID, conv.ID, asst.ID, "a-1", completeAssistantMessage("x", "y"))
	require.True(t, infraerrors.IsBadRequest(err))

	// Neither identifier -> 400.
	_, err = svc.ReplaceMessageFrom(ctx, userID, conv.ID, 0, "", completeAssistantMessage("x", "y"))
	require.True(t, infraerrors.IsBadRequest(err))

	// A non-assistant replacement message -> 400.
	_, err = svc.ReplaceMessageFrom(ctx, userID, conv.ID, asst.ID, "", completeUserMessage("x", "y"))
	require.True(t, infraerrors.IsBadRequest(err))
}

func TestConversationService_ReplaceIdempotentRetry(t *testing.T) {
	svc, client := newConversationStack(t)
	ctx := context.Background()
	userID := mustCreateConvUser(t, ctx, client, "repidem@test.com")

	conv, err := svc.CreateConversation(ctx, userID, "c", "T", "m")
	require.NoError(t, err)
	asst := appendTurn(t, ctx, svc, userID, conv.ID, "1", "q", "a")

	newMsg := completeAssistantMessage("a-regen", "regen")
	first, err := svc.ReplaceMessageFrom(ctx, userID, conv.ID, asst.ID, "", newMsg)
	require.NoError(t, err)

	// Retry with the same cutoff (now deleted) + same new message -> returns the
	// existing replacement (success), no duplicate, no error.
	again, err := svc.ReplaceMessageFrom(ctx, userID, conv.ID, asst.ID, "", newMsg)
	require.NoError(t, err)
	require.Equal(t, first.ID, again.ID)

	all, err := svc.ListMessagesBefore(ctx, userID, conv.ID, 0, 100)
	require.NoError(t, err)
	require.Len(t, all.Items, 2, "retry must not create a duplicate")

	// Same new client_message_id but a DIFFERENT payload -> conflict.
	_, err = svc.ReplaceMessageFrom(ctx, userID, conv.ID, asst.ID, "", completeAssistantMessage("a-regen", "tampered"))
	require.True(t, infraerrors.IsConflict(err))
}
