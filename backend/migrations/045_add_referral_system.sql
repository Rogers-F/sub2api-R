-- 邀请系统迁移：为 users 表添加邀请相关字段，并创建奖励记录表

-- 为 users 表添加邀请相关字段
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS referrer_id BIGINT REFERENCES users(id),
  ADD COLUMN IF NOT EXISTS referral_code VARCHAR(16);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_users_referrer_id ON users(referrer_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_referral_code_unique ON users(referral_code) WHERE deleted_at IS NULL AND referral_code IS NOT NULL;

COMMENT ON COLUMN users.referrer_id IS '邀请人用户ID';
COMMENT ON COLUMN users.referral_code IS '用户的邀请码（唯一）';

-- 创建邀请奖励记录表
CREATE TABLE IF NOT EXISTS referral_rewards (
    id BIGSERIAL PRIMARY KEY,
    referrer_id BIGINT NOT NULL REFERENCES users(id),
    referee_id BIGINT NOT NULL REFERENCES users(id),
    reward_type VARCHAR(20) NOT NULL,          -- 'register' (注册奖励) | 'commission' (返利)
    source_type VARCHAR(20),                   -- 'redeem_code' (来自兑换码)
    source_id BIGINT,                          -- 来源ID（如兑换码ID）
    source_amount DECIMAL(20,8) DEFAULT 0,     -- 来源金额
    reward_amount DECIMAL(20,8) NOT NULL,      -- 奖励金额
    commission_rate DECIMAL(5,4),              -- 返利比例（仅 commission 类型）
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_referral_rewards_referrer_id ON referral_rewards(referrer_id);
CREATE INDEX IF NOT EXISTS idx_referral_rewards_referee_id ON referral_rewards(referee_id);
CREATE INDEX IF NOT EXISTS idx_referral_rewards_reward_type ON referral_rewards(reward_type);
CREATE INDEX IF NOT EXISTS idx_referral_rewards_created_at ON referral_rewards(created_at);

COMMENT ON TABLE referral_rewards IS '邀请奖励记录表';
COMMENT ON COLUMN referral_rewards.referrer_id IS '邀请人ID';
COMMENT ON COLUMN referral_rewards.referee_id IS '被邀请人ID';
COMMENT ON COLUMN referral_rewards.reward_type IS '奖励类型：register=注册奖励，commission=返利';
COMMENT ON COLUMN referral_rewards.source_type IS '来源类型：redeem_code=兑换码';
COMMENT ON COLUMN referral_rewards.source_id IS '来源ID';
COMMENT ON COLUMN referral_rewards.source_amount IS '来源金额';
COMMENT ON COLUMN referral_rewards.reward_amount IS '奖励金额';
COMMENT ON COLUMN referral_rewards.commission_rate IS '返利比例';
