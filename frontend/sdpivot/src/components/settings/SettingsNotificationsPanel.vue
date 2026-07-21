<template>
  <div class="settings-panel muted-panel">
    <div class="panel-head">
      <div>
        <p class="panel-kicker">Notification roadmap</p>
        <h2>通知与提醒</h2>
      </div>
      <t-tag variant="outline">规划中</t-tag>
    </div>
    <div class="placeholder-grid">
      <div class="placeholder-card">
        <strong>系统通知</strong>
        <p>预留空间状态、导入任务、生成任务等提醒订阅配置。</p>
      </div>
      <div class="placeholder-card">
        <strong>工作流提醒</strong>
        <p>未来可承接写作、问答、运营侧任务事件通知。</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-panel {
  padding: 22px;
  background: var(--surface-elevated);
  border: 1px solid var(--border-soft);
  border-radius: 12px;
  box-shadow: var(--shadow-soft);
}
.muted-panel {
  background: linear-gradient(180deg, color-mix(in srgb, var(--surface-elevated) 98%, transparent), color-mix(in srgb, var(--sdp-surface-soft) 94%, transparent));
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
.placeholder-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}
.placeholder-card {
  border-radius: 12px;
  padding: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: color-mix(in srgb, var(--surface-elevated) 84%, transparent);
}
.placeholder-card strong {
  display: block;
  margin-bottom: 8px;
  color: var(--text-primary);
}
.placeholder-card p {
  color: var(--text-secondary);
  line-height: 1.6;
}
@media (max-width: 1080px) {
  .placeholder-grid {
    grid-template-columns: 1fr;
  }
}
</style>
