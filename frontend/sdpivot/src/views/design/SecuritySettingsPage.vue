<template>
  <SdpSidebarLayout mode="admin">
    <main class="security-page"><div class="shell">
      <header><div><p>Security Center</p><h1>安全设置</h1><span>管理 SDPivot·文枢 的访问边界、密码、会话、脱敏与审计策略。</span></div><SdpButton :loading="saving" @click="save">保存当前策略</SdpButton></header>
      <SdpNotice v-if="notice" :type="notice.type" :title="notice.text" />
      <div class="workspace"><nav aria-label="安全策略分类"><button v-for="tab in tabs" :key="tab.id" type="button" :class="{active:active===tab.id}" @click="active=tab.id"><strong>{{ tab.label }}</strong><small>{{ tab.description }}</small></button></nav>
        <section class="panel"><div class="panel-head"><div><p>{{ current.eyebrow }}</p><h2>{{ current.label }}</h2></div><span>{{ current.description }}</span></div><div v-if="loading" class="empty">正在加载安全策略...</div><div v-else class="fields"><label v-for="field in current.fields" :key="field.key"><span>{{ field.label }}</span><small>{{ field.help }}</small><textarea v-if="field.multiline" v-model="settings[active][field.key]" rows="6" :placeholder="field.placeholder"/><select v-else-if="field.options" v-model="settings[active][field.key]"><option v-for="option in field.options" :key="option.value" :value="option.value">{{ option.label }}</option></select><input v-else v-model="settings[active][field.key]" :type="field.type||'text'" :placeholder="field.placeholder" /></label></div></section>
      </div>
    </div></main>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { getSecuritySettings, saveSecuritySettings, type SettingsMap } from '@/api/admin'
import { SdpButton, SdpNotice } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

interface SecurityOption { label: string; value: string }
interface SecurityField { key: string; label: string; help: string; placeholder?: string; type?: string; multiline?: boolean; options?: SecurityOption[] }
interface SecurityTab { id: 'ip_whitelist' | 'password_policy' | 'session' | 'desensitize' | 'audit'; label: string; eyebrow: string; description: string; fields: SecurityField[] }
const yesNo=[{label:'启用',value:'true'},{label:'停用',value:'false'}]
const tabs: SecurityTab[]=[
 {id:'ip_whitelist',label:'IP 白名单',eyebrow:'Network Access',description:'限制管理端访问来源',fields:[{key:'enabled',label:'启用白名单',help:'启用后仅允许清单内地址访问',options:yesNo},{key:'addresses',label:'允许的 IP / CIDR',help:'每行一个地址，支持 CIDR',placeholder:'10.0.0.0/8\n203.0.113.10',multiline:true}]},
 {id:'password_policy',label:'密码策略',eyebrow:'Credentials',description:'控制密码强度与有效期',fields:[{key:'min_length',label:'最小长度',help:'建议不低于 8 位',placeholder:'12',type:'number'},{key:'complexity',label:'复杂度要求',help:'是否要求大小写、数字和符号',options:yesNo},{key:'valid_days',label:'有效期（天）',help:'0 表示永不过期',placeholder:'90',type:'number'}]},
 {id:'session',label:'会话',eyebrow:'Session',description:'控制登录会话生命周期',fields:[{key:'timeout_minutes',label:'空闲超时（分钟）',help:'无操作后自动退出',placeholder:'30',type:'number'},{key:'max_concurrent',label:'最大并发会话',help:'单用户允许同时登录的设备数',placeholder:'5',type:'number'}]},
 {id:'desensitize',label:'脱敏',eyebrow:'Data Masking',description:'保护敏感字段展示',fields:[{key:'enabled',label:'启用脱敏',help:'对界面和导出数据应用脱敏',options:yesNo},{key:'fields',label:'脱敏字段',help:'逗号分隔，例如 phone,email,id_card',placeholder:'phone,email,id_card'}]},
 {id:'audit',label:'审计',eyebrow:'Audit',description:'记录关键管理操作',fields:[{key:'enabled',label:'启用审计',help:'记录登录、配置与权限操作',options:yesNo},{key:'retention_days',label:'保留天数',help:'审计日志在线保留周期',placeholder:'180',type:'number'},{key:'include_reads',label:'记录读取操作',help:'开启后审计只读查询',options:yesNo}]},
]
type Section=typeof tabs[number]['id']
const active=ref<Section>('ip_whitelist'),settings=reactive<SettingsMap>(Object.fromEntries(tabs.map(tab=>[tab.id,{}]))),loading=ref(true),saving=ref(false),notice=ref<{type:'success'|'danger';text:string}|null>(null)
const current=computed(()=>tabs.find(tab=>tab.id===active.value)!)
async function load(){loading.value=true;try{Object.assign(settings,(await getSecuritySettings()).data.settings)}catch(e){fail(e,'安全设置加载失败')}finally{loading.value=false}}
async function save(){saving.value=true;notice.value=null;try{await saveSecuritySettings(active.value,settings[active.value]);notice.value={type:'success',text:`${current.value.label}策略已保存`}}catch(e){fail(e,'安全策略保存失败')}finally{saving.value=false}}
function fail(error:unknown,fallback:string){const value=error as {response?:{data?:{error?:string}}};notice.value={type:'danger',text:value.response?.data?.error||fallback}}
onMounted(load)
</script>

<style scoped>
.security-page{min-height:100dvh;background:var(--ink-100);color:var(--ink-900);font-family:var(--font-body)}.shell{display:grid;gap:var(--space-5);max-width:var(--content-max-width);margin:auto;padding:var(--space-8)}header,.panel-head{display:flex;align-items:center;justify-content:space-between;gap:var(--space-5)}header p,.panel-head p{color:var(--brand-700);font:var(--font-weight-semibold) var(--text-xs)/1 var(--font-mono);letter-spacing:.08em;text-transform:uppercase}h1{margin-top:var(--space-1);font:var(--font-weight-bold) var(--text-4xl)/1.1 var(--font-display)}header span,.panel-head>span{display:block;margin-top:var(--space-2);color:var(--ink-600)}.workspace{display:grid;grid-template-columns:16rem minmax(0,1fr);gap:var(--space-4)}nav,.panel{border:1px solid var(--ink-200);border-radius:var(--radius-lg);background:var(--ink-50)}nav{display:grid;align-content:start;gap:var(--space-2);padding:var(--space-3)}nav button{display:grid;gap:var(--space-1);padding:var(--space-4);border:0;border-radius:var(--radius-sm);background:transparent;color:var(--ink-800);text-align:left;font:inherit;cursor:pointer}nav button small{color:var(--ink-500)}nav button.active{background:var(--brand-100);box-shadow:inset 3px 0 var(--brand-600)}.panel{padding:var(--space-6)}h2{margin-top:var(--space-1);font:var(--font-weight-bold) var(--text-xl)/1.2 var(--font-display)}.fields{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:var(--space-5);margin-top:var(--space-6)}label{display:grid;gap:var(--space-2)}label span{font-weight:var(--font-weight-semibold)}label small{color:var(--ink-500)}input,textarea,select{width:100%;padding:var(--space-3);border:1px solid var(--ink-300);border-radius:var(--radius-sm);background:var(--ink-50);color:var(--ink-900);font:inherit}textarea{resize:vertical}.empty{padding:var(--space-8);text-align:center;color:var(--ink-500)}@media(max-width:55rem){.shell{padding:var(--space-4)}header,.panel-head{align-items:flex-start;flex-direction:column}.workspace{grid-template-columns:1fr}nav{grid-template-columns:repeat(2,1fr)}.fields{grid-template-columns:1fr}}@media(max-width:35rem){nav{grid-template-columns:1fr}.panel{padding:var(--space-5)}}
</style>
