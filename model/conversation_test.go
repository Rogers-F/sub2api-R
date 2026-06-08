package model

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetConversations clears the two conversation tables before and after a test
// (the package shares one in-memory SQLite DB via TestMain).
func resetConversations(t *testing.T) {
	t.Helper()
	clear := func() {
		DB.Exec("DELETE FROM conversation_messages")
		DB.Exec("DELETE FROM conversations")
	}
	clear()
	t.Cleanup(clear)
}

func userMsg(content, clientID string) MessageInput {
	return MessageInput{Role: MessageRoleUser, Content: content, Status: MessageStatusComplete, ClientMessageId: clientID}
}

func assistantMsg(content, clientID string) MessageInput {
	return MessageInput{Role: MessageRoleAssistant, Content: content, Status: MessageStatusComplete, ClientMessageId: clientID}
}

func TestConversation_AutoMigrated(t *testing.T) {
	require.True(t, DB.Migrator().HasTable(&Conversation{}))
	require.True(t, DB.Migrator().HasTable(&ConversationMessage{}))
}

func TestValidateMessageInput(t *testing.T) {
	// valid
	in := userMsg("hi", "c1")
	require.NoError(t, validateMessageInput(&in))

	// bad role (system rejected)
	bad := MessageInput{Role: "system", Content: "x", Status: MessageStatusComplete, ClientMessageId: "c2"}
	require.ErrorIs(t, validateMessageInput(&bad), ErrMessageInvalid)

	// user must be complete
	uErr := MessageInput{Role: MessageRoleUser, Content: "x", Status: MessageStatusError, ClientMessageId: "c3"}
	require.ErrorIs(t, validateMessageInput(&uErr), ErrMessageInvalid)

	// complete must have content
	empty := MessageInput{Role: MessageRoleAssistant, Content: "  ", Status: MessageStatusComplete, ClientMessageId: "c4"}
	require.ErrorIs(t, validateMessageInput(&empty), ErrMessageInvalid)

	// >512KB content
	big := MessageInput{Role: MessageRoleAssistant, Content: string(make([]byte, MaxMessageContentBytes+1)), Status: MessageStatusError, ClientMessageId: "c5"}
	require.ErrorIs(t, validateMessageInput(&big), ErrMessageInvalid)

	// empty client id
	noID := MessageInput{Role: MessageRoleAssistant, Content: "x", Status: MessageStatusComplete, ClientMessageId: "  "}
	require.ErrorIs(t, validateMessageInput(&noID), ErrMessageInvalid)

	// negative reported tokens
	neg := -1
	negTok := MessageInput{Role: MessageRoleAssistant, Content: "x", Status: MessageStatusComplete, ClientMessageId: "c6", ReportedInputTokens: &neg}
	require.ErrorIs(t, validateMessageInput(&negTok), ErrMessageInvalid)
}

func TestCursorRoundTrip(t *testing.T) {
	enc := EncodeConvCursor(&ConvCursor{LastMessageAt: 1700000000, ID: 42})
	got, err := DecodeConvCursor(enc)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(1700000000), got.LastMessageAt)
	assert.Equal(t, 42, got.ID)

	// empty -> nil, no error (first page)
	nilCur, err := DecodeConvCursor("")
	require.NoError(t, err)
	assert.Nil(t, nilCur)

	// malformed
	_, err = DecodeConvCursor("not-base64!!")
	require.Error(t, err)
}

func TestCreateConversation_Idempotent(t *testing.T) {
	resetConversations(t)
	c1, err := CreateConversation(1, "cc-1", "Title A", "m")
	require.NoError(t, err)
	require.Equal(t, c1.CreatedAt, c1.LastMessageAt) // no messages -> last_message_at == created_at

	// same client id -> existing returned, old title kept
	c2, err := CreateConversation(1, "cc-1", "Title B", "m")
	require.NoError(t, err)
	assert.Equal(t, c1.Id, c2.Id)
	assert.Equal(t, "Title A", c2.Title)
}

func TestCreateConversation_Validation(t *testing.T) {
	resetConversations(t)
	_, err := CreateConversation(0, "cc", "", "")
	require.ErrorIs(t, err, ErrConversationInvalid)
	_, err = CreateConversation(1, "   ", "", "")
	require.ErrorIs(t, err, ErrConversationInvalid)
}

func TestGetConversation_IDOR(t *testing.T) {
	resetConversations(t)
	c, err := CreateConversation(1, "cc-idor", "t", "")
	require.NoError(t, err)
	// other user cannot read it
	_, err = GetConversation(2, c.Id)
	require.ErrorIs(t, err, ErrConversationNotFound)
	// owner can
	got, err := GetConversation(1, c.Id)
	require.NoError(t, err)
	assert.Equal(t, c.Id, got.Id)
}

func TestUpdateTitle_404AndNoTouch(t *testing.T) {
	resetConversations(t)
	c, err := CreateConversation(1, "cc-rename", "old", "")
	require.NoError(t, err)
	require.NoError(t, touchConversationAt(DB, 1, c.Id, 500))

	updated, err := UpdateConversationTitle(1, c.Id, "new")
	require.NoError(t, err)
	assert.Equal(t, "new", updated.Title)
	assert.Equal(t, int64(500), updated.LastMessageAt) // rename must NOT advance last_message_at

	// missing id
	_, err = UpdateConversationTitle(1, 999999, "x")
	require.ErrorIs(t, err, ErrConversationNotFound)
	// other user's id (IDOR)
	_, err = UpdateConversationTitle(2, c.Id, "x")
	require.ErrorIs(t, err, ErrConversationNotFound)
}

func TestDeleteConversation(t *testing.T) {
	resetConversations(t)
	c, err := CreateConversation(1, "cc-del", "t", "")
	require.NoError(t, err)
	_, err = AppendMessages(1, c.Id, []MessageInput{userMsg("hi", "m1")})
	require.NoError(t, err)

	require.NoError(t, DeleteConversation(1, c.Id))
	_, err = GetConversation(1, c.Id)
	require.ErrorIs(t, err, ErrConversationNotFound)
	// children removed (app-level cascade on sqlite)
	var n int64
	DB.Model(&ConversationMessage{}).Where("conversation_id = ?", c.Id).Count(&n)
	assert.Equal(t, int64(0), n)
	// second delete -> not found
	require.ErrorIs(t, DeleteConversation(1, c.Id), ErrConversationNotFound)
}

func TestListConversations_KeysetOrder(t *testing.T) {
	resetConversations(t)
	a, _ := CreateConversation(1, "a", "", "")
	b, _ := CreateConversation(1, "b", "", "")
	cc, _ := CreateConversation(1, "c", "", "")
	require.NoError(t, touchConversationAt(DB, 1, a.Id, 100))
	require.NoError(t, touchConversationAt(DB, 1, b.Id, 200))
	require.NoError(t, touchConversationAt(DB, 1, cc.Id, 300))

	page1, next, err := ListConversations(1, nil, 2)
	require.NoError(t, err)
	require.Len(t, page1, 2)
	assert.Equal(t, cc.Id, page1[0].Id) // 300 first
	assert.Equal(t, b.Id, page1[1].Id)  // 200
	require.NotNil(t, next)

	page2, next2, err := ListConversations(1, next, 2)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.Equal(t, a.Id, page2[0].Id) // 100
	assert.Nil(t, next2)
}

func TestAppend_IdempotentAndTouch(t *testing.T) {
	resetConversations(t)
	c, err := CreateConversation(1, "cc-app", "", "")
	require.NoError(t, err)

	msgs := []MessageInput{userMsg("q", "u1"), assistantMsg("a", "a1")}
	out, err := AppendMessages(1, c.Id, msgs)
	require.NoError(t, err)
	require.Len(t, out, 2)

	maxCreated := out[0].CreatedAt
	if out[1].CreatedAt > maxCreated {
		maxCreated = out[1].CreatedAt
	}
	conv, _ := GetConversation(1, c.Id)
	assert.Equal(t, maxCreated, conv.LastMessageAt) // last_message_at == MAX(message.created_at)

	// idempotent replay: same payload -> count stays 2, same ids
	out2, err := AppendMessages(1, c.Id, msgs)
	require.NoError(t, err)
	require.Len(t, out2, 2)
	assert.Equal(t, out[0].Id, out2[0].Id)
	var n int64
	DB.Model(&ConversationMessage{}).Where("conversation_id = ?", c.Id).Count(&n)
	assert.Equal(t, int64(2), n)
}

func TestAppend_ConflictDifferentFingerprint(t *testing.T) {
	resetConversations(t)
	c, _ := CreateConversation(1, "cc-conf", "", "")
	_, err := AppendMessages(1, c.Id, []MessageInput{assistantMsg("first", "x1")})
	require.NoError(t, err)
	// same client id, different content -> conflict
	_, err = AppendMessages(1, c.Id, []MessageInput{assistantMsg("second", "x1")})
	require.ErrorIs(t, err, ErrMessageConflict)
}

func TestAppend_DupInBatch(t *testing.T) {
	resetConversations(t)
	c, _ := CreateConversation(1, "cc-dup", "", "")
	_, err := AppendMessages(1, c.Id, []MessageInput{assistantMsg("a", "d1"), assistantMsg("b", "d1")})
	require.ErrorIs(t, err, ErrMessageDuplicateInBatch)
}

func TestAppend_BatchLimit(t *testing.T) {
	resetConversations(t)
	c, _ := CreateConversation(1, "cc-lim", "", "")
	big := make([]MessageInput, maxMessageBatchSize+1)
	for i := range big {
		big[i] = assistantMsg("x", "lim-"+strconv.Itoa(i))
	}
	_, err := AppendMessages(1, c.Id, big)
	require.ErrorIs(t, err, ErrMessageInvalid)
}

func TestListMessages_BeforeAndForward(t *testing.T) {
	resetConversations(t)
	c, _ := CreateConversation(1, "cc-msgs", "", "")
	batch := make([]MessageInput, 0, 5)
	for i := 0; i < 5; i++ {
		batch = append(batch, assistantMsg("m"+strconv.Itoa(i), "k-"+strconv.Itoa(i)))
	}
	_, err := AppendMessages(1, c.Id, batch)
	require.NoError(t, err)

	// backward: newest page, limit 2 -> id ASC, hasMore
	page, hasMore, err := ListMessagesBefore(1, c.Id, 0, 2)
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.True(t, hasMore)
	assert.Less(t, page[0].Id, page[1].Id) // id ASC

	// older page via before_id = smallest id of this page
	older, _, err := ListMessagesBefore(1, c.Id, page[0].Id, 2)
	require.NoError(t, err)
	require.Len(t, older, 2)
	assert.Less(t, older[1].Id, page[0].Id)

	// forward: afterID 0 -> from start id ASC
	fwd, fwdMore, err := ListMessages(1, c.Id, 0, 3)
	require.NoError(t, err)
	require.Len(t, fwd, 3)
	assert.True(t, fwdMore)
	assert.Less(t, fwd[0].Id, fwd[2].Id)

	// IDOR: other user sees not-found
	_, _, err = ListMessagesBefore(2, c.Id, 0, 5)
	require.ErrorIs(t, err, ErrConversationNotFound)
}

func TestReplaceMessageFrom(t *testing.T) {
	resetConversations(t)
	c, _ := CreateConversation(1, "cc-rep", "", "")
	_, err := AppendMessages(1, c.Id, []MessageInput{userMsg("q", "u1"), assistantMsg("a1", "as1")})
	require.NoError(t, err)

	// replace trailing assistant a1 -> a2
	rep, err := ReplaceMessageFrom(1, c.Id, 0, "as1", assistantMsg("a2", "as2"))
	require.NoError(t, err)
	assert.Equal(t, "a2", rep.Content)
	var n int64
	DB.Model(&ConversationMessage{}).Where("conversation_id = ?", c.Id).Count(&n)
	assert.Equal(t, int64(2), n) // u1 + a2

	// using the user message (not the last, and not assistant) as cutoff -> conflict
	_, err = ReplaceMessageFrom(1, c.Id, 0, "u1", assistantMsg("a3", "as3"))
	require.ErrorIs(t, err, ErrMessageConflict)
}

func TestReplace_RequiresExactlyOneCutoff(t *testing.T) {
	resetConversations(t)
	c, _ := CreateConversation(1, "cc-rep2", "", "")
	// neither cutoff
	_, err := ReplaceMessageFrom(1, c.Id, 0, "", assistantMsg("x", "y"))
	require.ErrorIs(t, err, ErrMessageInvalid)
	// both cutoffs
	_, err = ReplaceMessageFrom(1, c.Id, 5, "cid", assistantMsg("x", "y"))
	require.ErrorIs(t, err, ErrMessageInvalid)
}
