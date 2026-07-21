<template>
  <div class="profile-panel-stack">
    <div class="settings-panel">
      <div class="panel-head">
        <div>
          <p class="panel-kicker">Account profile</p>
          <h2>账户资料</h2>
        </div>
        <t-tag theme="success" variant="light">已同步</t-tag>
      </div>

      <div class="profile-grid">
        <div class="profile-card focus-card">
          <span class="field-label">用户名</span>
          <strong>{{ user?.username || '--' }}</strong>
          <small>用于系统内展示与身份识别</small>
        </div>
        <div class="profile-card focus-card">
          <span class="field-label">邮箱</span>
          <strong>{{ user?.email || '--' }}</strong>
          <small>登录账号与通知接收地址</small>
        </div>
        <div class="profile-card focus-card">
          <span class="field-label">租户 ID</span>
          <strong>{{ user?.tenant_id || '--' }}</strong>
          <small>当前数据隔离上下文</small>
        </div>
      </div>
    </div>

    <div class="settings-panel muted-panel">
      <div class="panel-head">
        <div>
          <p class="panel-kicker">Security roadmap</p>
          <h2>安全与登录</h2>
        </div>
      </div>
      <div class="placeholder-grid">
        <div class="placeholder-card">
          <strong>密码更新</strong>
          <p>下一阶段接入修改密码与安全校验流程。</p>
        </div>
        <div class="placeholder-card">
          <strong>设备记录</strong>
          <p>预留最近登录设备与会话管理能力。</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  user?: Record<string, any> | null
}>()
</script>

<style scoped>
.profile-panel-stack {
  display: grid;
  gap: 18px;
}
.settings-panel {
  padding: 22px;
  background: var(--surface-elevated);
  border: 1px solid var(--border-soft);
  border-radius: 12px;
  box-shadow: var(--shadow-soft);
}
.panel-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 14px;
  margin-bottom: 18px;
}
.panel-kicker {
  margin: 0 0 8px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--brand-primary);
}
.panel-head h2 {
  margin: 0;
  font-size: 28px;
  line-height: 1.1;
  color: var(--text-primary);
}
.profile-grid,
.placeholder-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}
.profile-card,
.placeholder-card {
  border-radius: 12px;
  padding: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: color-mix(in srgb, var(--surface-elevated) 84%, transparent);
}
.field-label {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
}
.profile-card strong {
  display: block;
  margin-top: 8px;
  font-size: 20px;
  color: var(--text-primary);
}
.profile-card small,
.placeholder-card p {
  color: var(--text-secondary);
  line-height: 1.6;
}
.placeholder-card strong {
  display: block;
  margin-bottom: 8px;
  color: var(--text-primary);
}
.focus-card {
  min-height: 132px;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  gap: 10px;
}
.muted-panel {
  margin-top: 18px;
  background: linear-gradient(180deg, color-mix(in srgb, var(--surface-elevated) 98%, transparent), color-mix(in srgb, var(--sdp-surface-soft) 94%, transparent));
}
@media (max-width: 1080px) {
  .profile-grid,
  .placeholder-grid {
    grid-template-columns: 1fr;
  }
}
</style>
