<template>
  <div class="settings-panel muted-panel">
    <div class="panel-head">
      <div>
        <p class="panel-kicker">Appearance controls</p>
        <h2>外观与主题</h2>
      </div>
      <t-tag variant="outline">即时生效</t-tag>
    </div>
    <div class="theme-grid">
      <button type="button" class="theme-option" :class="{ active: themeMode === 'light' }" @click="setTheme('light')">
        <div class="theme-preview light-preview"></div>
        <strong>浅色模式</strong>
        <span>适合日间办公与默认阅读场景</span>
      </button>
      <button type="button" class="theme-option" :class="{ active: themeMode === 'dark' }" @click="setTheme('dark')">
        <div class="theme-preview dark-preview"></div>
        <strong>深色模式</strong>
        <span>降低夜间阅读眩光，强化沉浸感</span>
      </button>
      <button type="button" class="theme-option" :class="{ active: themeMode === 'system' }" @click="setTheme('system')">
        <div class="theme-preview system-preview">
          <div></div>
          <div></div>
        </div>
        <strong>跟随系统</strong>
        <span>{{ currentThemeLabel }}</span>
      </button>
    </div>

    <div class="placeholder-grid columns-3 extra-top">
      <div class="placeholder-card emphasis-card">
        <strong>主题切换</strong>
        <p>已接入 `light / dark / system` 三态，并持久化到本地存储。</p>
      </div>
      <div class="placeholder-card">
        <strong>阅读偏好</strong>
        <p>预留字号、密度、动效强度等个性化配置入口。</p>
      </div>
      <div class="placeholder-card">
        <strong>系统联动</strong>
        <p>当浏览器系统主题变化时，跟随系统模式会自动同步。</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  themeMode: 'light' | 'dark' | 'system'
  currentThemeLabel: string
  setTheme: (mode: 'light' | 'dark' | 'system') => void
}>()
</script>

<style scoped>
.settings-panel {
  padding: 24px;
  background: var(--surface-elevated);
  border: 1px solid var(--border-soft);
  border-radius: 24px;
  box-shadow: var(--shadow-soft);
}
.muted-panel {
  background: linear-gradient(180deg, color-mix(in srgb, var(--surface-elevated) 98%, transparent), color-mix(in srgb, var(--sk-surface-soft) 94%, transparent));
}
.panel-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 20px;
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
  font-size: 30px;
  line-height: 1.1;
  color: var(--text-primary);
}
.theme-grid,
.placeholder-grid,
.columns-3 {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}
.theme-option,
.placeholder-card {
  border-radius: 20px;
  padding: 20px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: color-mix(in srgb, var(--surface-elevated) 84%, transparent);
}
.theme-option {
  text-align: left;
  cursor: pointer;
  transition: transform 0.24s ease, border-color 0.24s ease, box-shadow 0.24s ease;
}
.theme-option:hover,
.theme-option.active {
  transform: translateY(-2px);
  border-color: rgba(0, 185, 107, 0.3);
  box-shadow: inset 0 0 0 1px rgba(0, 185, 107, 0.18);
}
.theme-option strong,
.placeholder-card strong {
  display: block;
  margin: 12px 0 8px;
  color: var(--text-primary);
}
.theme-option span,
.placeholder-card p {
  display: block;
  color: var(--text-secondary);
  line-height: 1.6;
}
.theme-preview {
  height: 110px;
  border-radius: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
}
.light-preview {
  background: linear-gradient(180deg, color-mix(in srgb, var(--sk-surface-soft) 92%, white) 0%, color-mix(in srgb, var(--surface-elevated) 96%, white) 100%);
}
.dark-preview {
  background: linear-gradient(180deg, #1f2937 0%, #0f172a 100%);
}
.system-preview {
  display: grid;
  grid-template-columns: 1fr 1fr;
  overflow: hidden;
}
.system-preview div:first-child {
  background: linear-gradient(180deg, color-mix(in srgb, var(--sk-surface-soft) 92%, white) 0%, color-mix(in srgb, var(--surface-elevated) 96%, white) 100%);
}
.system-preview div:last-child {
  background: linear-gradient(180deg, #1f2937 0%, #0f172a 100%);
}
.extra-top {
  margin-top: 18px;
}
.emphasis-card {
  background: linear-gradient(135deg, color-mix(in srgb, var(--brand-primary) 12%, transparent), color-mix(in srgb, var(--surface-elevated) 96%, transparent));
}
@media (max-width: 1080px) {
  .theme-grid,
  .placeholder-grid,
  .columns-3 {
    grid-template-columns: 1fr;
  }
}
</style>
