<template>
  <Teleport to="body">
    <section v-if="open" class="drawer-overlay" aria-label="导入文档抽屉" @click.self="close">
      <aside class="drawer drawer--wide" role="dialog" aria-modal="true" aria-labelledby="space-import-title">
        <header class="drawer-head">
          <div>
            <p class="eyebrow">Space / Document Import</p>
            <h2 id="space-import-title">导入文档</h2>
            <p class="subtle">上传文件后自动解析、切片、生成多维标签，并进入知识库待确认队列。</p>
          </div>
          <button class="close" type="button" aria-label="关闭导入文档" @click="close">×</button>
        </header>
        <div class="drawer-body">
          <div class="import-stepper">
            <div v-for="step in steps" :key="step.title" class="step" :class="{ active: step.active }">
              <strong>{{ step.title }}</strong>
              <span>{{ step.description }}</span>
            </div>
          </div>

          <section class="upload-zone" @drop.prevent="handleDrop" @dragover.prevent>
            <div>
              <div class="upload-icon">↑</div>
              <h3>拖入文档或选择文件</h3>
              <p class="subtle">支持 PDF、Office、Markdown、CSV 与常见图片；单文件建议不超过 50MB。</p>
              <div class="actions" style="justify-content:center;margin-top:16px">
                <button class="btn primary" type="button" @click="triggerFileInput">选择文件</button>
                <button class="btn secondary" type="button">从本地目录导入</button>
              </div>
              <input ref="fileInput" type="file" multiple accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.md,.csv,.txt,.png,.jpg,.jpeg" style="display:none" @change="handleFileSelect" />
            </div>
          </section>

          <section class="panel">
            <div class="panel-head">
              <h3 class="panel-title">导入策略</h3>
              <span class="badge blue">自动标签已开启</span>
            </div>
            <div class="panel-body rule-grid">
              <label v-for="rule in rules" :key="rule.title" class="check-card">
                <input v-model="rule.enabled" type="checkbox" />
                <span>
                  <strong>{{ rule.title }}</strong>
                  <span>{{ rule.description }}</span>
                </span>
              </label>
            </div>
          </section>

          <section class="panel">
            <div class="panel-head">
              <h3 class="panel-title">上传队列</h3>
              <button class="btn ghost" type="button" @click="clearCompleted">清空完成项</button>
            </div>
            <div class="panel-body file-list">
              <div v-for="(file, index) in fileQueue" :key="file.key" class="file-row">
                <div class="file-icon">{{ fileTypeIcon(file.name) }}</div>
                <div class="file-copy">
                  <strong>{{ file.name }}</strong>
                  <div class="subtle">{{ file.meta }}</div>
                  <div v-if="file.started" class="file-progress" :aria-label="`${file.name} 解析进度 ${file.progress}%`">
                    <div class="progress"><span :style="{ width: file.progress + '%' }"></span></div>
                    <strong>{{ file.progress }}%</strong>
                  </div>
                </div>
                <span class="badge" :class="file.variant">{{ statusText(file) }}</span>
                <div class="file-actions">
                  <button v-if="file.state === 'failed' && !file.oversized" class="retry" type="button" :disabled="file.retrying" @click="retryFile(file)">{{ file.retrying ? '重试中' : '重试' }}</button>
                  <button class="close row-action" type="button" aria-label="移除文件" :disabled="file.state === 'uploading'" @click="removeFile(index)">×</button>
                </div>
              </div>
              <SdpEmptyState v-if="!fileQueue.length" variant="compact" title="队列为空" description="选择或拖入文件后，将在此显示导入进度。" />
            </div>
            <div v-if="batchSummary" class="import-summary" role="status">
              <strong>本次导入完成</strong>
              <span>{{ batchSummary.success }} 成功 / {{ batchSummary.failed }} 失败</span>
            </div>
          </section>

          <div class="notice warning">
            <div class="notice-mark warning-mark">!</div>
            <div><strong>导入前检查</strong><br>包含敏感信息的文档将先进入脱敏规则检查，未通过时不会进入向量库。</div>
          </div>
        </div>
        <footer class="drawer-foot">
          <span class="subtle">预计 {{ fileQueue.length }} 个文件将在 {{ estimatedMinutes }} 分钟内完成解析</span>
          <div class="actions">
            <button class="btn secondary" type="button" @click="close">稍后处理</button>
            <button class="btn primary" type="button" :disabled="!hasPendingFiles || uploading" @click="startImport">{{ uploading ? '上传中...' : '开始导入' }}</button>
          </div>
        </footer>
      </aside>
    </section>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onUnmounted, watch } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { getDocumentParseStatus, reparseDocument, uploadDocument } from '@/api/documents';
import SdpEmptyState from './SdpEmptyState.vue';

const props = defineProps<{
  open: boolean;
  spaceId?: string;
  existingFiles?: number;
}>();
const emit = defineEmits<{
  'update:open': [value: boolean];
  imported: [];
}>();

const fileInput = ref<HTMLInputElement | null>(null);
type QueueState = 'waiting' | 'uploading' | 'parsing' | 'completed' | 'failed';
interface QueueItem {
  key: string;
  name: string;
  meta: string;
  progress: number;
  state: QueueState;
  variant: string;
  file: File;
  documentId?: string;
  started: boolean;
  retrying: boolean;
  oversized?: boolean;
  notified?: boolean;
}

const fileQueue = reactive<QueueItem[]>([]);
const uploading = ref(false);
const pollTimers = new Map<string, number>();

const steps = [
  { title: '1 上传', description: '选择文件或批量拖入', active: true },
  { title: '2 解析', description: 'OCR / 表格 / 文本抽取', active: true },
  { title: '3 标签', description: '文档类别与业务标签', active: false },
  { title: '4 入库', description: '向量化并可检索', active: false },
];

const rules = reactive([
  { title: '自动生成多维标签', description: '文档类别、业务类别、行业领域将进入待确认队列。', enabled: true },
  { title: '写作模板自学习', description: '同标签文档可作为 AI 写作辅助优先知识源。', enabled: true },
  { title: '解析完成后通知我', description: '通过站内通知提示导入结果与失败原因。', enabled: true },
  { title: '覆盖同名文件', description: '默认保留历史版本，勾选后替换最新版本。', enabled: false },
]);

const estimatedMinutes = computed(() => Math.max(1, Math.ceil(fileQueue.length * 0.8)));
const hasPendingFiles = computed(() => fileQueue.some(file => file.state === 'waiting' || (file.state === 'failed' && !file.oversized)));
const batchSummary = computed(() => {
  const processed = fileQueue.filter(file => file.started && file.state !== 'waiting');
  if (!processed.length || processed.some(file => !['completed', 'failed'].includes(file.state))) return null;
  return {
    success: processed.filter(file => file.state === 'completed').length,
    failed: processed.filter(file => file.state === 'failed').length,
  };
});

function fileTypeIcon(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() || '';
  const map: Record<string, string> = { pdf: 'PDF', doc: 'DOC', docx: 'DOC', xls: 'XLS', xlsx: 'XLS', ppt: 'PPT', pptx: 'PPT', md: 'MD', csv: 'CSV', txt: 'TXT', png: 'IMG', jpg: 'IMG', jpeg: 'IMG' };
  return map[ext] || 'FILE';
}

function addFiles(files: FileList | File[]) {
  const available = Math.max(0, 20 - fileQueue.length);
  const selected = Array.from(files);
  if (selected.length > available) MessagePlugin.warning(`一次最多导入 20 个文件，已保留前 ${available} 个`);
  for (const file of selected.slice(0, available)) {
    if (fileQueue.some(f => f.file.name === file.name && f.file.size === file.size)) continue;
    if (file.size > 50 * 1024 * 1024) {
      fileQueue.push({ key: fileKey(file), name: file.name, meta: '文件超过 50MB 限制', progress: 0, state: 'failed', variant: 'red', file, started: true, retrying: false, oversized: true });
      continue;
    }
    const size = file.size > 1024 * 1024
      ? `${(file.size / (1024 * 1024)).toFixed(1)}MB`
      : `${(file.size / 1024).toFixed(0)}KB`;
    fileQueue.push({
      key: fileKey(file),
      name: file.name,
      meta: `${size} · 等待上传`,
      progress: 0,
      state: 'waiting',
      variant: 'gray',
      file,
      started: false,
      retrying: false,
    });
  }
}

function fileKey(file: File) { return `${file.name}-${file.size}-${file.lastModified}`; }

function statusText(item: QueueItem) {
  if (item.state === 'waiting') return '等待中';
  if (item.state === 'uploading') return `上传中(${item.progress}%)`;
  if (item.state === 'parsing') return `解析中(${item.progress}%)`;
  if (item.state === 'completed') return '已完成';
  return '失败';
}

function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement;
  if (target.files?.length) addFiles(target.files);
  target.value = '';
}

function handleDrop(event: DragEvent) {
  if (event.dataTransfer?.files?.length) addFiles(event.dataTransfer.files);
}

function triggerFileInput() {
  fileInput.value?.click();
}

function removeFile(index: number) {
  if (fileQueue[index]?.state === 'uploading') return;
  stopPolling(fileQueue[index]?.key);
  fileQueue.splice(index, 1);
}

function clearCompleted() {
  for (let i = fileQueue.length - 1; i >= 0; i--) {
    if (fileQueue[i].state === 'completed') {
      stopPolling(fileQueue[i].key);
      fileQueue.splice(i, 1);
    }
  }
}

async function startImport() {
  if (!props.spaceId || uploading.value) return;
  uploading.value = true;
  for (const item of fileQueue) {
    if (item.state !== 'waiting') continue;
    await uploadFile(item);
  }
  uploading.value = false;
}

async function uploadFile(item: QueueItem) {
  if (!props.spaceId) return;
  stopPolling(item.key);
  item.started = true;
  item.state = 'uploading';
  item.variant = 'blue';
  item.meta = '正在上传';
  item.progress = 0;
  try {
    const response = await uploadDocument({ space_id: props.spaceId, file: item.file }, percent => { item.progress = percent; });
    item.documentId = response.data.document.id;
    item.state = 'parsing';
    item.progress = 0;
    item.meta = '上传完成，等待解析';
    schedulePoll(item, 0);
  } catch (error: unknown) {
    markFailed(item, errorMessage(error, '上传失败，请重试'));
  }
}

async function retryFile(item: QueueItem) {
  if (item.retrying || item.oversized) return;
  item.retrying = true;
  item.notified = false;
  try {
    if (item.documentId) {
      await reparseDocument(item.documentId);
      item.started = true;
      item.state = 'parsing';
      item.variant = 'blue';
      item.progress = 0;
      item.meta = '已重新提交解析';
      schedulePoll(item, 0);
    } else {
      await uploadFile(item);
    }
  } catch (error: unknown) {
    markFailed(item, errorMessage(error, '重试失败，请稍后再试'));
  } finally {
    item.retrying = false;
  }
}

function schedulePoll(item: QueueItem, delay = 1200) {
  stopPolling(item.key);
  pollTimers.set(item.key, window.setTimeout(() => pollParseStatus(item), delay));
}

async function pollParseStatus(item: QueueItem) {
  if (!item.documentId || !fileQueue.includes(item)) return;
  try {
    const response = await getDocumentParseStatus(item.documentId);
    const status = String(response.data.status || '').toLowerCase();
    item.progress = Math.min(100, Math.max(0, Math.round(response.data.progress || 0)));
    if (status === 'completed') {
      item.state = 'completed';
      item.variant = 'green';
      item.progress = 100;
      item.meta = '解析完成，已进入知识库';
      stopPolling(item.key);
      if (!item.notified) {
        item.notified = true;
        MessagePlugin.success(`${item.name}：文档已入库，可检索`);
        emit('imported');
      }
      return;
    }
    if (status === 'failed') {
      markFailed(item, response.data.error || '文档解析失败，请重试');
      return;
    }
    item.state = 'parsing';
    item.variant = 'blue';
    item.meta = status === 'pending' ? '等待解析任务' : '正在解析文档内容';
    schedulePoll(item);
  } catch (error: unknown) {
    markFailed(item, errorMessage(error, '解析状态获取失败，请重试'));
  }
}

function markFailed(item: QueueItem, reason: string) {
  stopPolling(item.key);
  item.state = 'failed';
  item.variant = 'red';
  item.meta = reason;
}

function stopPolling(key?: string) {
  if (!key) return;
  const timer = pollTimers.get(key);
  if (timer !== undefined) window.clearTimeout(timer);
  pollTimers.delete(key);
}

function close() { emit('update:open', false); }

watch(() => props.open, (value) => {
  document.body.style.overflow = value ? 'hidden' : '';
});
onUnmounted(() => {
  document.body.style.overflow = '';
  pollTimers.forEach(timer => window.clearTimeout(timer));
  pollTimers.clear();
});

function errorMessage(error: unknown, fallback: string) {
  if (typeof error !== 'object' || !error) return fallback;
  const value = error as { message?: string; response?: { data?: { error?: string; message?: string } } };
  return value.response?.data?.error || value.response?.data?.message || value.message || fallback;
}
</script>

<style scoped>
.import-stepper { display: grid; grid-template-columns: repeat(4, 1fr); gap: var(--space-2); }
.step { min-height: 76px; border: 1px solid var(--ink-200); background: white; border-radius: var(--radius-md); padding: var(--space-3); }
.step.active { border-color: var(--brand-300); background: var(--brand-50); }
.step strong { display: block; font-size: var(--text-sm); margin-bottom: var(--space-1); }
.step span { color: var(--ink-500); font-size: var(--text-xs); }
.upload-zone { min-height: 172px; border: 1.5px dashed var(--brand-300); border-radius: var(--radius-xl); background: linear-gradient(180deg, var(--brand-50), white); display: grid; place-items: center; text-align: center; padding: var(--space-8); cursor: pointer; }
.upload-icon { width: 52px; height: 52px; border-radius: var(--radius-lg); margin: 0 auto var(--space-4); background: var(--brand-100); color: var(--brand-800); display: grid; place-items: center; font-size: var(--text-2xl); font-weight: var(--font-weight-bold); }
.upload-zone h3 { margin: 0 0 var(--space-2); }
.rule-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-4); }
.check-card { border: 1px solid var(--ink-200); border-radius: var(--radius-md); padding: var(--space-4); background: white; display: flex; align-items: flex-start; gap: var(--space-3); cursor: pointer; }
.check-card input { margin-top: 3px; }
.check-card strong { display: block; font-size: var(--text-sm); }
.check-card span span { display: block; color: var(--ink-500); font-size: var(--text-xs); margin-top: var(--space-1); }
.file-list { display: grid; gap: var(--space-3); }
.file-row { display: grid; grid-template-columns: 44px minmax(0, 1fr) 112px auto; align-items: center; gap: var(--space-3); padding: var(--space-3); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: white; }
.file-icon { width: 44px; height: 44px; border-radius: var(--radius-md); display: grid; place-items: center; background: var(--ink-100); color: var(--ink-700); font-size: var(--text-xs); font-weight: var(--font-weight-bold); }
.file-copy { min-width: 0; }
.file-copy > strong, .file-copy .subtle { overflow-wrap: anywhere; }
.file-progress { display: grid; grid-template-columns: minmax(80px, 1fr) 38px; align-items: center; gap: var(--space-2); margin-top: var(--space-2); }
.file-progress strong { color: var(--brand-800); font-size: var(--text-xs); text-align: right; }
.file-actions { display: flex; align-items: center; gap: var(--space-2); }
.retry { min-height: 30px; padding: 4px 10px; border: 1px solid var(--brand-300); border-radius: var(--radius-sm); color: var(--brand-900); background: var(--brand-50); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); cursor: pointer; }
.retry:disabled, .row-action:disabled { cursor: not-allowed; opacity: .55; }
.row-action { width: 32px; height: 32px; cursor: pointer; }
.import-summary { display: flex; justify-content: space-between; gap: var(--space-3); padding: var(--space-3) var(--space-5); border-top: 1px solid var(--ink-200); color: var(--ink-700); background: var(--brand-50); font-size: var(--text-sm); }
.import-summary strong { color: var(--brand-900); }
.warning-mark { background: var(--warning-500); }
@media (max-width: 760px) { .import-stepper, .rule-grid, .file-row { grid-template-columns: 1fr; } }
</style>

<style scoped>
.drawer-overlay { position: fixed; inset: 0; z-index: var(--z-modal, 300); display: grid; justify-items: end; background: rgba(15, 23, 19, .42); backdrop-filter: blur(4px); }
.drawer { width: 620px; max-width: 100vw; height: 100%; display: grid; grid-template-rows: auto 1fr auto; background: var(--ink-50); border-left: 1px solid rgba(255,255,255,.32); box-shadow: -32px 0 80px rgba(0,0,0,.20); animation: drawer-in .24s var(--ease-out-expo); }
.drawer--wide { width: 720px; }
.drawer-head { padding: 24px 28px 18px; background: white; border-bottom: 1px solid var(--ink-200); display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-5); }
.drawer-head h2 { margin: var(--space-1) 0 0; font-size: var(--text-2xl); letter-spacing: -.035em; }
.drawer-body { padding: 22px 28px; overflow: auto; display: grid; gap: var(--space-5); }
.drawer-foot { padding: var(--space-4) 28px; border-top: 1px solid var(--ink-200); background: white; display: flex; justify-content: space-between; align-items: center; gap: var(--space-4); }
.eyebrow { margin: 0 0 var(--space-1); color: var(--brand-700); font-size: var(--text-xs); font-weight: var(--font-weight-bold); letter-spacing: .08em; text-transform: uppercase; }
.subtle { margin: var(--space-2) 0 0; color: var(--ink-600); font-size: var(--text-sm); }
.actions { display: flex; align-items: center; gap: var(--space-3); }
.btn { min-height: 36px; border: 1px solid transparent; border-radius: var(--radius-sm); padding: 8px 14px; font-weight: var(--font-weight-semibold); font-size: var(--text-sm); cursor: pointer; }
.btn.primary { background: var(--ink-900); color: white; border-color: var(--ink-900); }
.btn.primary:disabled { opacity: .5; cursor: not-allowed; }
.btn.secondary { background: white; color: var(--ink-700); border-color: var(--ink-200); }
.btn.ghost { background: transparent; color: var(--ink-700); }
.close { width: 36px; height: 36px; border-radius: var(--radius-pill); border: 1px solid var(--ink-200); background: var(--ink-50); color: var(--ink-700); font-weight: var(--font-weight-bold); }
.notice { border: 1px solid var(--brand-200); background: var(--brand-50); color: var(--brand-900); border-radius: var(--radius-md); padding: var(--space-4); display: grid; grid-template-columns: 24px 1fr; gap: var(--space-3); font-size: var(--text-sm); }
.notice.warning { border-color: var(--warning-500); background: var(--warning-50); color: var(--ink-800); }
.notice-mark { width: 24px; height: 24px; border-radius: 50%; background: var(--brand-600); color: white; display: grid; place-items: center; font-weight: var(--font-weight-bold); font-size: var(--text-xs); }
.panel { background: white; border: 1px solid var(--ink-200); border-radius: var(--radius-lg); box-shadow: var(--shadow-sm); }
.panel-head { padding: var(--space-5); border-bottom: 1px solid var(--ink-200); display: flex; justify-content: space-between; align-items: center; gap: var(--space-4); }
.panel-title { margin: 0; font-size: var(--text-base); }
.panel-body { padding: var(--space-5); }
.progress { height: 8px; border-radius: var(--radius-pill); background: var(--ink-100); overflow: hidden; }
.progress > span { display: block; height: 100%; background: linear-gradient(90deg, var(--brand-400), var(--brand-700)); }
.badge { display: inline-flex; align-items: center; min-height: 24px; padding: 0 9px; border-radius: var(--radius-pill); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.badge.green { background: var(--success-50); color: oklch(0.42 0.12 155); }
.badge.yellow { background: var(--warning-50); color: oklch(0.48 0.14 70); }
.badge.red { background: var(--danger-50, #fff1f0); color: var(--danger-700, #b42318); }
.badge.blue { background: var(--info-50); color: oklch(0.43 0.13 230); }
.badge.gray { background: var(--ink-100); color: var(--ink-600); }
@keyframes drawer-in { from { transform: translateX(36px); opacity: .8; } to { transform: translateX(0); opacity: 1; } }
@media (max-width: 760px) { .drawer, .drawer--wide { width: 100vw; } .drawer-foot { flex-direction: column; align-items: stretch; } }
@media (prefers-reduced-motion: reduce) { .drawer { animation-duration: .01ms; } }
</style>
