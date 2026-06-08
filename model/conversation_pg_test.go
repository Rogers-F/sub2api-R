package model

import (
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// openConvPG points the package's global DB at a real Postgres for the duration
// of one test, so the FOR UPDATE row lock is actually exercised (SQLite cannot).
// It is gated on TEST_POSTGRES_DSN and refuses to touch a DB that already has the
// conversation tables (anti-clobber). The returned func restores the prior
// (SQLite) globals and drops the test tables.
func openConvPG(t *testing.T) func() {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping Postgres concurrency gate")
	}
	pg, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true}), &gorm.Config{})
	require.NoError(t, err)
	if pg.Migrator().HasTable(&Conversation{}) || pg.Migrator().HasTable(&ConversationMessage{}) {
		if sqlDB, e := pg.DB(); e == nil {
			_ = sqlDB.Close()
		}
		t.Skip("target Postgres already has conversation tables; refusing to clobber")
	}

	oldDB, oldLog := DB, LOG_DB
	oldSQLite, oldPG := common.UsingSQLite, common.UsingPostgreSQL
	common.UsingSQLite = false
	common.UsingPostgreSQL = true
	DB, LOG_DB = pg, pg

	require.NoError(t, pg.AutoMigrate(&Conversation{}, &ConversationMessage{}))
	require.NoError(t, migrateConversationConstraints(pg))

	return func() {
		_ = pg.Migrator().DropTable(&ConversationMessage{}, &Conversation{})
		if sqlDB, e := pg.DB(); e == nil {
			_ = sqlDB.Close()
		}
		DB, LOG_DB = oldDB, oldLog
		common.UsingSQLite, common.UsingPostgreSQL = oldSQLite, oldPG
	}
}

func maxMessageCreatedAt(t *testing.T, conversationID int) int64 {
	t.Helper()
	var v int64
	require.NoError(t, DB.Model(&ConversationMessage{}).
		Where("conversation_id = ?", conversationID).
		Select("COALESCE(MAX(created_at), 0)").Scan(&v).Error)
	return v
}

// TestPG_AppendSerializedUnderRowLock: N concurrent single-message appends each
// with a distinct client id all succeed, the count is exact, and the invariant
// last_message_at == MAX(message.created_at) holds.
func TestPG_AppendSerializedUnderRowLock(t *testing.T) {
	defer openConvPG(t)()
	c, err := CreateConversation(1, "pg-serial", "", "")
	require.NoError(t, err)

	const N = 8
	var wg sync.WaitGroup
	errs := make([]error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = AppendMessages(1, c.Id, []MessageInput{assistantMsg("m"+strconv.Itoa(i), "k-"+strconv.Itoa(i))})
		}(i)
	}
	wg.Wait()
	for i := 0; i < N; i++ {
		require.NoError(t, errs[i])
	}

	var n int64
	require.NoError(t, DB.Model(&ConversationMessage{}).Where("conversation_id = ?", c.Id).Count(&n).Error)
	require.Equal(t, int64(N), n)

	conv, err := GetConversation(1, c.Id)
	require.NoError(t, err)
	require.Equal(t, maxMessageCreatedAt(t, c.Id), conv.LastMessageAt)
}

// TestPG_ConcurrentSameClientID_NoDuplicate: N concurrent appends of the SAME
// client_message_id yield exactly one row (lock + read-then-insert + unique).
func TestPG_ConcurrentSameClientID_NoDuplicate(t *testing.T) {
	defer openConvPG(t)()
	c, err := CreateConversation(1, "pg-dup", "", "")
	require.NoError(t, err)

	const N = 6
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = AppendMessages(1, c.Id, []MessageInput{assistantMsg("same", "dup")})
		}()
	}
	wg.Wait()

	var n int64
	require.NoError(t, DB.Model(&ConversationMessage{}).
		Where("conversation_id = ? AND client_message_id = ?", c.Id, "dup").Count(&n).Error)
	require.Equal(t, int64(1), n)
}

// TestPG_ReplaceConcurrentOnlyOneWins: N concurrent replaces from the same cutoff
// — exactly one succeeds (cutoff must equal MAX(id)); the rest get 409.
func TestPG_ReplaceConcurrentOnlyOneWins(t *testing.T) {
	defer openConvPG(t)()
	c, err := CreateConversation(1, "pg-rep", "", "")
	require.NoError(t, err)
	_, err = AppendMessages(1, c.Id, []MessageInput{userMsg("q", "u1"), assistantMsg("a1", "as1")})
	require.NoError(t, err)

	const N = 5
	var wg sync.WaitGroup
	results := make([]error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, results[i] = ReplaceMessageFrom(1, c.Id, 0, "as1", assistantMsg("a2-"+strconv.Itoa(i), "as2-"+strconv.Itoa(i)))
		}(i)
	}
	wg.Wait()

	wins := 0
	for i := 0; i < N; i++ {
		if results[i] == nil {
			wins++
		}
	}
	require.Equal(t, 1, wins)

	var n int64
	require.NoError(t, DB.Model(&ConversationMessage{}).Where("conversation_id = ?", c.Id).Count(&n).Error)
	require.Equal(t, int64(2), n) // u1 + the single winning replacement
}
