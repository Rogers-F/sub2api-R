package service

import "strconv"

// OpenAITokenCacheKey 生成 OpenAI OAuth 账号的缓存键
// 格式: "openai:account:{account_id}"
func OpenAITokenCacheKey(account *Account) string {
	return "openai:account:" + strconv.FormatInt(account.ID, 10)
}

// ClaudeTokenCacheKey 生成 Claude (Anthropic) OAuth 账号的缓存键
// 格式: "claude:account:{account_id}"
func ClaudeTokenCacheKey(account *Account) string {
	return "claude:account:" + strconv.FormatInt(account.ID, 10)
}

// TokenCacheKeyForAccount 根据账号平台返回对应的缓存键
// 用于后台刷新服务获取与请求侧一致的锁键
func TokenCacheKeyForAccount(account *Account) string {
	switch account.Platform {
	case PlatformAnthropic:
		return ClaudeTokenCacheKey(account)
	case PlatformOpenAI:
		return OpenAITokenCacheKey(account)
	case PlatformGemini:
		return GeminiTokenCacheKey(account)
	case PlatformAntigravity:
		return AntigravityTokenCacheKey(account)
	default:
		return "unknown:account:" + strconv.FormatInt(account.ID, 10)
	}
}
