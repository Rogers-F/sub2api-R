-- 公告系统迁移：创建公告表和已读记录表

-- 创建公告表
CREATE TABLE IF NOT EXISTS announcements (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    content_type VARCHAR(20) DEFAULT 'markdown',   -- markdown/html/url
    priority INT DEFAULT 0,                        -- 排序优先级，越大越靠前
    status VARCHAR(20) DEFAULT 'active',           -- active/inactive
    published_at TIMESTAMPTZ,                      -- 发布时间（NULL=立即发布）
    expires_at TIMESTAMPTZ,                        -- 过期时间（NULL=永不过期）
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_announcements_status ON announcements(status);
CREATE INDEX IF NOT EXISTS idx_announcements_priority ON announcements(priority DESC);
CREATE INDEX IF NOT EXISTS idx_announcements_published_at ON announcements(published_at);
CREATE INDEX IF NOT EXISTS idx_announcements_expires_at ON announcements(expires_at);

COMMENT ON TABLE announcements IS '系统公告表';
COMMENT ON COLUMN announcements.title IS '公告标题';
COMMENT ON COLUMN announcements.content IS '公告内容';
COMMENT ON COLUMN announcements.content_type IS '内容类型：markdown/html/url';
COMMENT ON COLUMN announcements.priority IS '排序优先级，越大越靠前';
COMMENT ON COLUMN announcements.status IS '状态：active=启用，inactive=禁用';
COMMENT ON COLUMN announcements.published_at IS '发布时间，NULL表示立即发布';
COMMENT ON COLUMN announcements.expires_at IS '过期时间，NULL表示永不过期';

-- 创建公告已读记录表
CREATE TABLE IF NOT EXISTS announcement_reads (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    announcement_id BIGINT NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, announcement_id)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_announcement_reads_user_id ON announcement_reads(user_id);
CREATE INDEX IF NOT EXISTS idx_announcement_reads_announcement_id ON announcement_reads(announcement_id);

COMMENT ON TABLE announcement_reads IS '公告已读记录表';
COMMENT ON COLUMN announcement_reads.user_id IS '用户ID';
COMMENT ON COLUMN announcement_reads.announcement_id IS '公告ID';
COMMENT ON COLUMN announcement_reads.read_at IS '阅读时间';
