<template>
  <SdpSidebarLayout mode="admin">
    <main class="admin-page"><div class="shell">
      <header><div><p>AI Configuration</p><h1>模型管理</h1><span>统一配置 SDPivot·文枢 的 LLM 与 Embedding 模型。</span></div><SdpButton @click="openCreate">新增模型</SdpButton></header>
      <SdpNotice v-if="notice" :type="notice.type" :title="notice.text" />
      <form v-if="editing" class="form" @submit.prevent="save">
        <div class="form-head"><div><p>Model Profile</p><h2>{{ draft.id ? '编辑模型' : '新增模型' }}</h2></div><button type="button" @click="editing=false">关闭</button></div>
        <div class="fields">
          <label><span>模型名称</span><input v-model.trim="draft.name" required placeholder="deepseek-chat" /></label>
          <label><span>显示名称</span><input v-model.trim="draft.displayName" placeholder="DeepSeek Chat" /></label>
          <label><span>模型类型</span><select v-model="draft.type"><option value="KnowledgeQA">LLM / 知识问答</option><option value="Embedding">Embedding</option><option value="Rerank">Rerank</option><option value="VLLM">视觉模型</option></select></label>
          <label><span>服务商</span><select v-model="draft.source"><option value="openai">OpenAI 兼容</option><option value="deepseek">DeepSeek</option><option value="hunyuan">混元</option><option value="aliyun">Qwen / 阿里云</option><option value="local">本地模型</option><option value="remote">自定义远程</option></select></label>
          <label class="wide"><span>API Endpoint</span><input v-model.trim="draft.baseUrl" placeholder="https://api.example.com/v1" /></label>
          <label><span>API Key</span><input v-model="draft.apiKey" type="password" :placeholder="draft.apiConfigured ? '已配置，留空保持不变' : '输入 API Key'" /></label>
          <label><span>状态</span><select v-model="draft.status"><option value="active">启用</option><option value="download_failed">禁用</option></select></label>
          <label class="wide"><span>说明</span><textarea v-model.trim="draft.description" rows="3" placeholder="模型用途与能力说明" /></label>
        </div>
        <div class="form-actions"><SdpButton :loading="saving" @click="save">保存模型</SdpButton><SdpButton variant="ghost" @click="editing=false">取消</SdpButton></div>
      </form>
      <section class="panel"><div class="panel-head"><div><p>Model Pool</p><h2>已配置模型</h2></div><SdpButton variant="secondary" size="sm" :loading="loading" @click="load">刷新</SdpButton></div>
        <div class="cards"><article v-for="model in models" :key="model.id" class="model-card"><div class="model-top"><div><span class="type">{{ typeLabel(model.type) }}</span><h3>{{ model.display_name || model.name }}</h3><code>{{ model.name }}</code></div><span :class="['status', model.status]">{{ model.status === 'active' ? '启用' : '禁用' }}</span></div><p>{{ model.description || '暂无说明' }}</p><dl><div><dt>服务商</dt><dd>{{ model.source }}</dd></div><div><dt>Endpoint</dt><dd>{{ model.parameters?.base_url || '默认地址' }}</dd></div><div><dt>凭据</dt><dd>{{ model.credentials?.api_key?.configured ? '已配置' : model.source === 'local' ? '无需凭据' : '未配置' }}</dd></div></dl><div class="actions"><SdpButton size="sm" variant="ghost" :disabled="model.is_builtin" @click="openEdit(model)">编辑</SdpButton><SdpButton size="sm" variant="secondary" :loading="working===model.id+'test'" @click="test(model)">测连</SdpButton><SdpButton size="sm" :disabled="model.is_default || model.is_builtin" :loading="working===model.id+'default'" @click="setDefault(model)">{{ model.is_default ? '默认模型' : '设为默认' }}</SdpButton></div></article><div v-if="!loading&&!models.length" class="empty">暂无模型配置</div></div>
      </section>
    </div></main>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { createAdminModel, listAdminModels, setDefaultAdminModel, testAdminModel, updateAdminModel, type AdminModel } from '@/api/admin'
import { SdpButton, SdpNotice } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

const models=ref<AdminModel[]>([]),loading=ref(true),saving=ref(false),editing=ref(false),working=ref('')
const notice=ref<{type:'success'|'danger';text:string}|null>(null)
const draft=reactive({id:'',name:'',displayName:'',type:'KnowledgeQA',source:'openai',baseUrl:'',apiKey:'',description:'',status:'active',apiConfigured:false})
function reset(){Object.assign(draft,{id:'',name:'',displayName:'',type:'KnowledgeQA',source:'openai',baseUrl:'',apiKey:'',description:'',status:'active',apiConfigured:false})}
function openCreate(){reset();editing.value=true}
function openEdit(model:AdminModel){Object.assign(draft,{id:model.id,name:model.name,displayName:model.display_name,type:model.type,source:model.source,baseUrl:model.parameters?.base_url||'',apiKey:'',description:model.description,status:model.status,apiConfigured:!!model.credentials?.api_key?.configured});editing.value=true}
async function load(){loading.value=true;try{models.value=(await listAdminModels()).data.models||[]}catch(e){fail(e,'模型列表加载失败')}finally{loading.value=false}}
async function save(){if(!draft.name||saving.value)return;saving.value=true;const payload={name:draft.name,display_name:draft.displayName,type:draft.type,source:draft.source,description:draft.description,status:draft.status,parameters:{base_url:draft.baseUrl,api_key:draft.apiKey,provider:draft.source,interface_type:draft.source==='local'?'ollama':'openai'}};try{draft.id?await updateAdminModel(draft.id,payload):await createAdminModel(payload);notice.value={type:'success',text:'模型配置已保存'};editing.value=false;await load()}catch(e){fail(e,'模型保存失败')}finally{saving.value=false}}
async function test(model:AdminModel){working.value=model.id+'test';try{const result=await testAdminModel(model.id);notice.value={type:'success',text:`连接成功：${result.data.response||'OK'}`}}catch(e){fail(e,'模型连接失败')}finally{working.value=''}}
async function setDefault(model:AdminModel){working.value=model.id+'default';try{await setDefaultAdminModel(model.id);notice.value={type:'success',text:'默认模型已更新'};await load()}catch(e){fail(e,'默认模型设置失败')}finally{working.value=''}}
function typeLabel(value:string){return value==='KnowledgeQA'?'LLM':value}
function fail(error:unknown,fallback:string){const value=error as {response?:{data?:{error?:string;detail?:string}}};notice.value={type:'danger',text:value.response?.data?.detail||value.response?.data?.error||fallback}}
onMounted(load)
</script>

<style scoped>
.admin-page{min-height:100dvh;background:var(--ink-100);color:var(--ink-900);font-family:var(--font-body)}.shell{display:grid;gap:var(--space-5);max-width:var(--content-max-width);margin:auto;padding:var(--space-8)}header,.panel-head,.form-head,.actions,.form-actions{display:flex;align-items:center}header,.panel-head,.form-head{justify-content:space-between;gap:var(--space-5)}header p,.panel-head p,.form-head p{color:var(--brand-700);font:var(--font-weight-semibold) var(--text-xs)/1 var(--font-mono);letter-spacing:.08em;text-transform:uppercase}h1{margin-top:var(--space-1);font:var(--font-weight-bold) var(--text-4xl)/1.1 var(--font-display)}header span{display:block;margin-top:var(--space-2);color:var(--ink-600)}.form,.panel{padding:var(--space-5);border:1px solid var(--ink-200);border-radius:var(--radius-lg);background:var(--ink-50)}h2,h3{font-family:var(--font-display)}.form-head button{border:0;background:transparent;color:var(--ink-600);cursor:pointer}.fields{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:var(--space-4);margin-top:var(--space-5)}label{display:grid;gap:var(--space-2);font-size:var(--text-sm);font-weight:var(--font-weight-semibold)}label.wide{grid-column:1/-1}input,select,textarea{width:100%;padding:var(--space-3);border:1px solid var(--ink-300);border-radius:var(--radius-sm);background:var(--ink-50);font:inherit}.form-actions,.actions{gap:var(--space-2);margin-top:var(--space-5)}.cards{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:var(--space-4);margin-top:var(--space-5)}.model-card{padding:var(--space-5);border:1px solid var(--ink-200);border-radius:var(--radius-md);background:linear-gradient(145deg,var(--ink-50),var(--ink-100))}.model-top{display:flex;justify-content:space-between;gap:var(--space-3)}.model-top code,.model-card>p{color:var(--ink-600);font-size:var(--text-sm)}.type,.status{display:inline-block;padding:var(--space-1) var(--space-2);border-radius:var(--radius-pill);font-size:var(--text-xs);font-weight:var(--font-weight-bold)}.type{margin-bottom:var(--space-2);background:var(--brand-100);color:var(--brand-800)}.status{height:max-content;background:var(--ink-200)}.status.active{background:var(--success-100);color:var(--success-800)}.model-card>p{min-height:2.8em;margin-top:var(--space-4)}dl{display:grid;gap:var(--space-2);margin-top:var(--space-4)}dl div{display:grid;grid-template-columns:5rem minmax(0,1fr);gap:var(--space-2);font-size:var(--text-sm)}dt{color:var(--ink-500)}dd{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.empty{grid-column:1/-1;padding:var(--space-8);text-align:center;color:var(--ink-500)}@media(max-width:48rem){.shell{padding:var(--space-5)}header{align-items:flex-start;flex-direction:column}.fields,.cards{grid-template-columns:1fr}}
</style>
