<template>
  <SdpSidebarLayout mode="admin">
    <main class="token-page">
      <div class="shell">
        <header><div><p>API Access</p><h1>API Token 管理</h1><span>为 SDPivot·文枢 CLI、MCP 与自动化集成签发可撤销凭据。</span></div><SdpButton @click="showForm = !showForm">{{ showForm ? '取消生成' : '生成 Token' }}</SdpButton></header>
        <SdpNotice v-if="notice" :type="notice.type" :title="notice.text" />
        <section v-if="revealedToken" class="secret" aria-live="polite"><div><strong>请立即保存 Token</strong><p>该明文仅显示一次，关闭后无法再次查看。</p></div><code>{{ revealedToken }}</code><SdpButton variant="secondary" size="sm" @click="copyToken">复制</SdpButton></section>
        <form v-if="showForm" class="form" @submit.prevent="createToken"><label><span>名称</span><input v-model.trim="draft.name" required maxlength="100" placeholder="例如：生产环境 MCP" /></label><label><span>Scopes</span><input v-model="draft.scopes" placeholder="knowledge.read, knowledge.write" /></label><label><span>过期时间</span><input v-model="draft.expiresAt" type="datetime-local" /></label><SdpButton :loading="creating" @click="createToken">确认生成</SdpButton></form>
        <section class="panel"><div class="panel-head"><div><p>Credentials</p><h2>已签发 Token</h2></div><SdpButton variant="secondary" size="sm" :loading="loading" @click="load">刷新</SdpButton></div>
          <div class="table-wrap"><table><thead><tr><th>名称</th><th>前缀</th><th>Scopes</th><th>过期时间</th><th>最后使用</th><th>状态</th><th>操作</th></tr></thead><tbody><tr v-for="token in tokens" :key="token.id"><td><input v-if="editing === token.id" v-model="editName" /><strong v-else>{{ token.name }}</strong></td><td><code>{{ token.prefix }}••••</code></td><td>{{ scopeText(token.scopes) }}</td><td>{{ formatTime(token.expires_at) }}</td><td>{{ formatTime(token.last_used_at) }}</td><td><span :class="['status', status(token)]">{{ statusLabel(token) }}</span></td><td class="actions"><template v-if="editing === token.id"><SdpButton size="sm" :loading="working === token.id" @click="saveEdit(token)">保存</SdpButton><SdpButton size="sm" variant="ghost" @click="editing = ''">取消</SdpButton></template><template v-else><SdpButton size="sm" variant="ghost" :disabled="!!token.revoked_at" @click="startEdit(token)">编辑</SdpButton><SdpButton size="sm" variant="secondary" :loading="working === token.id" @click="regenerate(token)">重生成</SdpButton><SdpButton size="sm" variant="danger" :disabled="!!token.revoked_at" :loading="working === token.id" @click="revoke(token)">撤销</SdpButton></template></td></tr><tr v-if="!loading && tokens.length === 0"><td colspan="7" class="empty">暂无 API Token</td></tr></tbody></table></div>
        </section>
      </div>
    </main>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { createAdminApiToken, listAdminApiTokens, regenerateAdminApiToken, revokeAdminApiToken, updateAdminApiToken, type AdminApiToken } from '@/api/admin'
import { SdpButton, SdpNotice } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

const tokens = ref<AdminApiToken[]>([]), loading = ref(true), creating = ref(false), showForm = ref(false), revealedToken = ref(''), editing = ref(''), editName = ref(''), working = ref('')
const draft = reactive({ name: '', scopes: 'knowledge.read', expiresAt: '' })
const notice = ref<{ type: 'success' | 'danger'; text: string } | null>(null)
async function load(){loading.value=true;try{tokens.value=(await listAdminApiTokens()).data.tokens||[]}catch(e){fail(e,'Token 列表加载失败')}finally{loading.value=false}}
async function createToken(){if(!draft.name||creating.value)return;creating.value=true;try{const response=await createAdminApiToken({name:draft.name,scopes:parseScopes(draft.scopes),expires_at:draft.expiresAt?new Date(draft.expiresAt).toISOString():null});revealedToken.value=response.data.token;showForm.value=false;draft.name='';await load()}catch(e){fail(e,'Token 生成失败')}finally{creating.value=false}}
function startEdit(token:AdminApiToken){editing.value=token.id;editName.value=token.name}
async function saveEdit(token:AdminApiToken){working.value=token.id;try{await updateAdminApiToken(token.id,{name:editName.value,scopes:normalizeScopes(token.scopes)});editing.value='';await load()}catch(e){fail(e,'Token 更新失败')}finally{working.value=''}}
async function regenerate(token:AdminApiToken){working.value=token.id;try{revealedToken.value=(await regenerateAdminApiToken(token.id)).data.token;await load()}catch(e){fail(e,'Token 重生成失败')}finally{working.value=''}}
async function revoke(token:AdminApiToken){if(!window.confirm(`确定撤销“${token.name}”吗？`))return;working.value=token.id;try{await revokeAdminApiToken(token.id);await load()}catch(e){fail(e,'Token 撤销失败')}finally{working.value=''}}
async function copyToken(){await navigator.clipboard.writeText(revealedToken.value);notice.value={type:'success',text:'Token 已复制到剪贴板'}}
function parseScopes(value:string){return value.split(',').map(v=>v.trim()).filter(Boolean)}
function normalizeScopes(value:unknown){if(Array.isArray(value))return value.map(String);if(typeof value==='string'){try{const parsed=JSON.parse(value);return Array.isArray(parsed)?parsed.map(String):parseScopes(value)}catch{return parseScopes(value)}}return []}
function scopeText(value:unknown){return normalizeScopes(value).join(', ')||'无'}
function formatTime(value?:string|null){return value?new Date(value).toLocaleString('zh-CN'):'永不过期 / 未使用'}
function status(token:AdminApiToken){if(token.revoked_at)return'revoked';if(token.expires_at&&new Date(token.expires_at)<=new Date())return'expired';return'active'}
function statusLabel(token:AdminApiToken){return status(token)==='active'?'有效':status(token)==='revoked'?'已撤销':'已过期'}
function fail(error:unknown,fallback:string){const value=error as {response?:{data?:{error?:string}}};notice.value={type:'danger',text:value.response?.data?.error||fallback}}
onMounted(load)
</script>

<style scoped>
.token-page{min-height:100dvh;background:var(--ink-100);color:var(--ink-900);font-family:var(--font-body)}.shell{display:grid;gap:var(--space-5);max-width:var(--content-max-width);margin:auto;padding:var(--space-8)}header,.panel-head,.secret,.form,.actions{display:flex;align-items:center}header,.panel-head{justify-content:space-between;gap:var(--space-5)}header p,.panel-head p{color:var(--brand-700);font:var(--font-weight-semibold) var(--text-xs)/1 var(--font-mono);letter-spacing:.08em;text-transform:uppercase}h1{margin-top:var(--space-1);font:var(--font-weight-bold) var(--text-4xl)/1.1 var(--font-display)}header span{display:block;margin-top:var(--space-2);color:var(--ink-600)}.secret,.form,.panel{padding:var(--space-5);border:1px solid var(--ink-200);border-radius:var(--radius-lg);background:var(--ink-50)}.secret{gap:var(--space-4);border-color:var(--brand-400);background:var(--brand-50)}.secret div{flex:1}.secret p{margin-top:var(--space-1);color:var(--ink-600);font-size:var(--text-sm)}.secret code{max-width:45%;overflow:auto;padding:var(--space-3);background:var(--ink-950);color:var(--brand-300);border-radius:var(--radius-sm)}.form{align-items:end;gap:var(--space-4)}label{display:grid;flex:1;gap:var(--space-2);font-size:var(--text-sm);font-weight:var(--font-weight-semibold)}input{min-width:0;padding:var(--space-3);border:1px solid var(--ink-300);border-radius:var(--radius-sm);background:var(--ink-50);font:inherit}.panel-head{margin-bottom:var(--space-5)}h2{margin-top:var(--space-1);font:var(--font-weight-bold) var(--text-xl)/1.2 var(--font-display)}.table-wrap{overflow:auto}table{width:100%;border-collapse:collapse}th,td{padding:var(--space-3);border-top:1px solid var(--ink-200);font-size:var(--text-sm);text-align:left;white-space:nowrap}thead th{border-top:0;background:var(--ink-100);color:var(--ink-600);font-size:var(--text-xs)}.actions{gap:var(--space-2)}.status{padding:var(--space-1) var(--space-2);border-radius:var(--radius-pill);font-size:var(--text-xs);font-weight:bold}.status.active{background:var(--brand-100);color:var(--brand-800)}.status.revoked,.status.expired{background:var(--ink-200);color:var(--ink-700)}.empty{text-align:center;color:var(--ink-500)}@media(max-width:55rem){.shell{padding:var(--space-4)}header,.secret,.form{align-items:stretch;flex-direction:column}.secret code{max-width:100%;width:100%}}
</style>
