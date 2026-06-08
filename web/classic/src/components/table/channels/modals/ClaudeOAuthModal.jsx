/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Modal,
  Button,
  Space,
  Typography,
  Input,
  Banner,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../../helpers';

const { Text } = Typography;

// ClaudeOAuthModal lets an admin paste an Anthropic OAuth subscription refresh_token.
// It is live-validated server-side (a real token refresh against platform.claude.com),
// and on success the resulting credential JSON is filled into the channel key field.
// The channel must then be saved with auth_mode=oauth (the parent handles that).
const ClaudeOAuthModal = ({ visible, onCancel, onSuccess, defaultProxy }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [refreshToken, setRefreshToken] = useState('');
  const [proxy, setProxy] = useState('');

  const submit = async () => {
    if (!refreshToken || !refreshToken.trim()) {
      showError(t('请先粘贴 refresh_token'));
      return;
    }
    setLoading(true);
    try {
      const res = await API.post(
        '/api/channel/claude/oauth/validate',
        { refresh_token: refreshToken.trim(), proxy: (proxy || '').trim() },
        { skipErrorHandler: true },
      );
      if (!res?.data?.success) {
        throw new Error(res?.data?.message || t('校验失败'));
      }
      const key = res?.data?.data?.key || '';
      if (!key) {
        throw new Error(t('响应缺少凭据'));
      }
      onSuccess && onSuccess(key);
      showSuccess(t('已校验并填入 OAuth 订阅凭据'));
      onCancel && onCancel();
    } catch (error) {
      showError(error?.message || t('校验失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!visible) return;
    setRefreshToken('');
    setProxy(defaultProxy || '');
  }, [visible, defaultProxy]);

  return (
    <Modal
      title={t('导入 Claude OAuth 订阅')}
      visible={visible}
      onCancel={onCancel}
      maskClosable={false}
      closeOnEsc
      width={720}
      footer={
        <Space>
          <Button theme='borderless' onClick={onCancel} disabled={loading}>
            {t('取消')}
          </Button>
          <Button
            theme='solid'
            type='primary'
            onClick={submit}
            loading={loading}
          >
            {t('校验并填入')}
          </Button>
        </Space>
      }
    >
      <Space vertical spacing='tight' style={{ width: '100%' }}>
        <Banner
          type='info'
          description={t(
            '粘贴 Claude.ai / Anthropic OAuth 订阅的 refresh_token，系统将向官方刷新一次以校验并自动获取账号信息，随后把凭据填入下方密钥框。保存渠道前请确保已选择 OAuth 认证模式，并建议先禁用渠道再保存、最后启用，避免与其他系统争抢刷新。',
          )}
        />

        <Input
          value={refreshToken}
          onChange={(value) => setRefreshToken(value)}
          placeholder={t('请粘贴 refresh_token')}
          showClear
        />

        <Input
          value={proxy}
          onChange={(value) => setProxy(value)}
          placeholder={t('可选：刷新使用的代理，例如 socks5://user:pass@host:port')}
          showClear
        />

        <Text type='tertiary' size='small'>
          {t(
            '说明：校验通过后会把包含 access_token / refresh_token / account_uuid 的 JSON 填入密钥框；refresh_token 仅用于一次性校验，不会被回显。',
          )}
        </Text>
      </Space>
    </Modal>
  );
};

export default ClaudeOAuthModal;
