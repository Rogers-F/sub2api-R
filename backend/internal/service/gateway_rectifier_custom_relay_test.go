//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type rectifierSettingsRepoStub struct {
	value string
	err   error
}

func (s *rectifierSettingsRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *rectifierSettingsRepoStub) GetValue(context.Context, string) (string, error) {
	return s.value, s.err
}

func (s *rectifierSettingsRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *rectifierSettingsRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *rectifierSettingsRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *rectifierSettingsRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *rectifierSettingsRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestGatewayService_BuildCustomRelayURL(t *testing.T) {
	proxyID := int64(1)
	account := &Account{
		ProxyID: &proxyID,
		Proxy: &Proxy{
			Protocol: "http",
			Host:     "proxy.example.com",
			Port:     8080,
		},
	}

	got := (&GatewayService{}).buildCustomRelayURL("https://relay.example.com/", "/v1/messages", account)
	require.Equal(t, "https://relay.example.com/v1/messages?beta=true&proxy=http%3A%2F%2Fproxy.example.com%3A8080", got)
}

func TestGatewayService_ShouldRectifySignatureError_APIKeyCustomPattern(t *testing.T) {
	repo := &rectifierSettingsRepoStub{
		value: `{"enabled":true,"thinking_signature_enabled":true,"thinking_budget_enabled":true,"apikey_signature_enabled":true,"apikey_signature_patterns":["custom_sig_error"]}`,
	}
	svc := &GatewayService{
		settingService: &SettingService{settingRepo: repo},
	}

	got := svc.shouldRectifySignatureError(context.Background(), &Account{Type: AccountTypeAPIKey}, []byte(`{"error":{"message":"CUSTOM_SIG_ERROR happened"}}`))
	require.True(t, got)
}

func TestGatewayService_ShouldRectifySignatureError_APIKeyDisabled(t *testing.T) {
	repo := &rectifierSettingsRepoStub{
		value: `{"enabled":true,"thinking_signature_enabled":true,"thinking_budget_enabled":true,"apikey_signature_enabled":false,"apikey_signature_patterns":["custom_sig_error"]}`,
	}
	svc := &GatewayService{
		settingService: &SettingService{settingRepo: repo},
	}

	got := svc.shouldRectifySignatureError(context.Background(), &Account{Type: AccountTypeAPIKey}, []byte(`{"error":{"message":"custom_sig_error happened"}}`))
	require.False(t, got)
}

func TestGatewayService_ShouldRectifySignatureError_GroupOverrideWhenGlobalDisabled(t *testing.T) {
	repo := &rectifierSettingsRepoStub{
		value: `{"enabled":false,"thinking_signature_enabled":false,"thinking_budget_enabled":true,"apikey_signature_enabled":false}`,
	}
	svc := &GatewayService{
		settingService: &SettingService{settingRepo: repo},
	}

	ctx := context.WithValue(context.Background(), ctxkey.Group, &Group{
		ID:                             99,
		Name:                           "mixed-channel-group",
		Platform:                       PlatformAnthropic,
		Status:                         StatusActive,
		Hydrated:                       true,
		ThinkingSignatureCompatEnabled: true,
	})

	got := svc.shouldRectifySignatureError(ctx, &Account{Type: AccountTypeOAuth}, []byte(`{"error":{"message":"thinking block signatures in the conversation history are no longer valid. (request id: upstream-1)"}}`))
	require.True(t, got)
}
