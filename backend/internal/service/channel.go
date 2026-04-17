package service

import "time"

type Channel struct {
	ID             int64
	Status         string
	FeaturesConfig map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (c *Channel) IsActive() bool {
	return c != nil && c.Status == StatusActive
}

func (c *Channel) Clone() *Channel {
	if c == nil {
		return nil
	}
	cp := *c
	if c.FeaturesConfig != nil {
		cp.FeaturesConfig = deepCopyFeaturesConfig(c.FeaturesConfig)
	}
	return &cp
}

func (c *Channel) IsWebSearchEmulationEnabled(platform string) bool {
	if c == nil || c.FeaturesConfig == nil {
		return false
	}
	raw, ok := c.FeaturesConfig[featureKeyWebSearchEmulation]
	if !ok {
		return false
	}
	config, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	enabled, ok := config[platform].(bool)
	return ok && enabled
}

func deepCopyFeaturesConfig(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		if inner, ok := v.(map[string]any); ok {
			dst[k] = deepCopyFeaturesConfig(inner)
			continue
		}
		dst[k] = v
	}
	return dst
}
