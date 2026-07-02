<template>
  <div class="settings-page">
    <h1>个人设置</h1>
    <t-card class="settings-card">
      <t-form label-align="top">
        <t-form-item label="用户名">
          <t-input :value="authStore.user?.username || ''" readonly />
        </t-form-item>
        <t-form-item label="邮箱">
          <t-input :value="authStore.user?.email || ''" readonly />
        </t-form-item>
        <t-form-item label="租户ID">
          <t-input :value="String(authStore.user?.tenant_id || '')" readonly />
        </t-form-item>
      </t-form>
    </t-card>
  </div>
</template>
<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
const authStore = useAuthStore()
</script>
<style scoped>
.settings-page { max-width: 600px; margin: 0 auto; }
.settings-page h1 { font-size: 24px; font-weight: 600; margin-bottom: 24px; }
</style>
