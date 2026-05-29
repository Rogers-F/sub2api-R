package service

import (
	"context"
	"errors"
	"testing"
)

// TestChannelMonitorCreate_EncryptionKeyGate 验证未配置固定加密主密钥时，
// 创建监控会在写入 api_key 密文前被拒绝（避免写入重启后不可解密的数据）。
//
// 用例构造：endpoint 用公网 IP 字面量（validateEndpoint 走 IP 字面量分支，无 DNS、无网络），
// 校验通过后命中加密门禁。门禁在 encryptor.Encrypt / repo.Create 之前返回，
// 因此 repo/encryptor 传 nil 不会被解引用。
func TestChannelMonitorCreate_EncryptionKeyGate(t *testing.T) {
	svc := NewChannelMonitorService(nil, nil, false) // encryptionKeyConfigured=false

	_, err := svc.Create(context.Background(), ChannelMonitorCreateParams{
		Name:            "gate-test",
		Provider:        MonitorProviderOpenAI,
		Endpoint:        "https://8.8.8.8", // 公网 IP 字面量，validateEndpoint 不做 DNS
		APIKey:          "sk-should-not-be-stored",
		PrimaryModel:    "gpt-4",
		IntervalSeconds: 60,
	})

	if !errors.Is(err, ErrChannelMonitorEncryptionKeyNotConfigured) {
		t.Fatalf("expected ErrChannelMonitorEncryptionKeyNotConfigured, got %v", err)
	}
}

// TestChannelMonitorUpdateAPIKey_EncryptionKeyGate 验证未配置固定加密主密钥时，
// 通过 applyAPIKeyUpdate 更新 api_key 同样被拒绝；而"不更新 key"（raw 为空）应放行（返回 updated=false）。
func TestChannelMonitorUpdateAPIKey_EncryptionKeyGate(t *testing.T) {
	svc := NewChannelMonitorService(nil, nil, false)

	// 提供新 key → 命中门禁
	newKey := "sk-new-key"
	if _, _, err := svc.applyAPIKeyUpdate(&ChannelMonitor{}, &newKey); !errors.Is(err, ErrChannelMonitorEncryptionKeyNotConfigured) {
		t.Fatalf("update with new key: expected gate error, got %v", err)
	}

	// 不改 key（nil / 空白）→ 不触门禁，updated=false
	if _, updated, err := svc.applyAPIKeyUpdate(&ChannelMonitor{}, nil); err != nil || updated {
		t.Fatalf("update without key: expected (updated=false, nil err), got updated=%v err=%v", updated, err)
	}
	blank := "   "
	if _, updated, err := svc.applyAPIKeyUpdate(&ChannelMonitor{}, &blank); err != nil || updated {
		t.Fatalf("update with blank key: expected (updated=false, nil err), got updated=%v err=%v", updated, err)
	}
}

// TestChannelMonitorCreate_EncryptionKeyConfigured_PassesGate 验证已配置密钥时门禁放行：
// 继续走到 encrypt。让 stub encryptor 返回错误，使 Create 在 Encrypt 之后、repo.Create 之前返回，
// 从而无需依赖 nil-repo panic 即可断言"门禁已放行且 Encrypt 被调用"。
func TestChannelMonitorCreate_EncryptionKeyConfigured_PassesGate(t *testing.T) {
	enc := &recordingEncryptor{err: errors.New("boom")}
	svc := NewChannelMonitorService(nil, enc, true) // configured=true

	_, err := svc.Create(context.Background(), ChannelMonitorCreateParams{
		Name:            "gate-pass",
		Provider:        MonitorProviderOpenAI,
		Endpoint:        "https://8.8.8.8",
		APIKey:          "sk-ok",
		PrimaryModel:    "gpt-4",
		IntervalSeconds: 60,
	})

	if !enc.called {
		t.Fatalf("expected encryptor.Encrypt to be called (gate passed when key configured)")
	}
	// 应是 encrypt 阶段的错误，而非加密门禁错误（证明门禁已放行）。
	if errors.Is(err, ErrChannelMonitorEncryptionKeyNotConfigured) {
		t.Fatalf("gate should have passed when key configured, got gate error: %v", err)
	}
	if err == nil {
		t.Fatalf("expected encrypt error to propagate, got nil")
	}
}

type recordingEncryptor struct {
	called bool
	err    error
}

func (e *recordingEncryptor) Encrypt(plaintext string) (string, error) {
	e.called = true
	if e.err != nil {
		return "", e.err
	}
	return "ciphertext", nil
}

func (e *recordingEncryptor) Decrypt(ciphertext string) (string, error) {
	return "", nil
}
