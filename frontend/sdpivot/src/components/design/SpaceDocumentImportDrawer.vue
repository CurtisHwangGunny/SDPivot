<template>
  <Teleport to="body">
    <section v-if="open" class="drawer-overlay" aria-label="导入文档抽屉" @click.self="close">
      <aside class="drawer" role="dialog" aria-modal="true" aria-labelledby="space-import-title">
        <header class="drawer-head">
          <div>
            <p class="eyebrow">Space / Document Import</p>
            <h2 id="space-import-title">导入文档</h2>
            <p class="subtle">文件加入队列并确认标签后，系统将依次完成上传、解析和入库。</p>
          </div>
          <button class="close" type="button" aria-label="关闭导入文档" @click="close">×</button>
        </header>

        <div class="drawer-body">
          <div class="notice">
            <div class="notice-mark">i</div>
            <div><strong>导入状态全程可见</strong><br>最多 20 个文件，单文件不超过 50MB。每个文件独立展示解析进度、失败原因与重试操作。</div>
          </div>

          <section class="panel">
            <div class="panel-head">
              <h3 class="panel-title">添加文件</h3>
              <span class="badge blue">队列 {{ fileQueue.length }} / 20</span>
            </div>
            <div class="panel-body">
              <div class="upload-zone" role="button" tabindex="0" @click="triggerFileInput" @keydown.enter="triggerFileInput" @keydown.space.prevent="triggerFileInput" @drop.prevent="handleDrop" @dragover.prevent>
                <div>
                  <div class="upload-icon" aria-hidden="true">↑</div>
                  <h3>拖入文件或点击选择</h3>
                  <p class="subtle">PDF、Word、Excel、PowerPoint、Markdown、TXT、CSV</p>
                  <button class="btn primary select-file" type="button" @click.stop="triggerFileInput">选择文件</button>
                  <input ref="fileInput" type="file" multiple accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.md,.csv,.txt" hidden @change="handleFileSelect" />
                </div>
              </div>
              <label class="tag-field">
                <span>文档标签（选填）</span>
                <input v-model.trim="confirmedTags" type="text" placeholder="例如：合同, 客户案例" :disabled="uploading">
                <small>标签将应用到本次尚未上传的文件，开始导入即视为确认。</small>
              </label>
            </div>
          </section>

          <section class="panel">
            <div class="panel-head queue-head">
              <h3 class="panel-title">上传队列</h3>
              <div class="queue-summary" aria-label="队列状态汇总">
                <span class="badge green">{{ completed }} 成功</span>
                <span class="badge blue">{{ parsing }} 解析中</span>
                <span class="badge red">{{ failed }} 失败</span>
              </div>
            </div>
            <div class="panel-body file-list">
              <div v-for="(file, index) in fileQueue" :key="file.key" class="file-row">
                <div class="file-icon">{{ fileTypeIcon(file.name) }}</div>
                <div class="file-copy">
                  <strong>{{ file.name }}</strong>
                  <small :class="{ danger: file.state === 'failed' }">{{ fileDescription(file) }}</small>
                </div>
                <div class="progress" :class="{ failed: file.state === 'failed' }" :aria-label="`${file.name} 进度 ${file.progress}%`">
                  <span :style="{ width: `${file.progress}%` }"></span>
                </div>
                <span class="badge" :class="statusVariant(file)">{{ statusText(file) }}</span>
                <div class="file-actions">
                  <button v-if="file.state === 'waiting'" class="btn ghost row-button" type="button" @click="removeFile(index)">移除</button>
                  <button v-else-if="file.state === 'uploading'" class="btn ghost row-button" type="button" @click="cancelUpload(file)">取消</button>
                  <button v-else-if="file.state === 'parsing'" class="btn ghost row-button" type="button" @click="stopWaiting(file)">停止等待</button>
                  <button v-else-if="file.state === 'completed'" class="btn ghost row-button" type="button" @click="viewFile(file)">查看</button>
                  <button v-else-if="file.failureKind === 'duplicate' && file.documentId" class="btn secondary row-button" type="button" @click="viewFile(file)">查看已有</button>
                  <button v-else class="btn secondary row-button" type="button" :disabled="file.retrying || file.oversized" @click="retryFile(file)">{{ file.retrying ? '处理中' : retryText(file) }}</button>
                </div>
              </div>
              <SdpEmptyState v-if="!fileQueue.length" variant="compact" title="队列为空" description="选择或拖入文件后，将在此显示导入进度。" />
            </div>
          </section>

          <div v-if="failed > 0" class="notice danger" role="alert">
            <div class="notice-mark">!</div>
             <div><strong>{{ failed }} 个文件需要处理</strong><br>可根据失败原因重试；如提示文档已存在，可直接查看已有文档。</div>
          </div>
        </div>

        <footer class="drawer-foot">
          <span class="subtle" :class="{ 'import-error': !spaceIdAvailable }">
            {{ spaceIdAvailable ? `当前：${completed} 个成功 / ${parsing} 个解析中 / ${waiting} 个等待 / ${failed} 个失败` : '缺少空间信息' }}
          </span>
          <div class="actions">
            <button class="btn secondary" type="button" @click="close">关闭</button>
            <button class="btn primary" type="button" :disabled="!canStartImport" @click="startImport">{{ uploading ? '导入中…' : '开始导入' }}</button>
          </div>
        </footer>
      </aside>
    </section>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onUnmounted, reactive, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
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

const router = useRouter();
const fileInput = ref<HTMLInputElement | null>(null);
type QueueState = 'waiting' | 'uploading' | 'parsing' | 'completed' | 'failed';
type FailureKind = 'validation' | 'upload' | 'parse' | 'status' | 'duplicate';
interface QueueItem {
  key: string;
  name: string;
  size: string;
  meta: string;
  progress: number;
  state: QueueState;
  file: File;
  documentId?: string;
  started: boolean;
  retrying: boolean;
  oversized?: boolean;
  notified?: boolean;
  failureKind?: FailureKind;
}

const fileQueue = reactive<QueueItem[]>([]);
const uploading = ref(false);
const confirmedTags = ref('');
const pollTimers = new Map<string, number>();
const uploadControllers = new Map<string, AbortController>();

const completed = computed(() => fileQueue.filter(file => file.state === 'completed').length);
const parsing = computed(() => fileQueue.filter(file => file.state === 'parsing').length);
const waiting = computed(() => fileQueue.filter(file => file.state === 'waiting').length);
const failed = computed(() => fileQueue.filter(file => file.state === 'failed').length);
const hasPendingFiles = computed(() => fileQueue.some(file => file.state === 'waiting' || (file.state === 'failed' && !file.oversized)));
const spaceIdAvailable = computed(() => Boolean(props.spaceId?.trim()));
const canStartImport = computed(() => spaceIdAvailable.value && hasPendingFiles.value && !uploading.value);

function fileTypeIcon(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() || '';
  const map: Record<string, string> = { pdf: 'PDF', doc: 'DOC', docx: 'DOC', xls: 'XLS', xlsx: 'XLS', ppt: 'PPT', pptx: 'PPT', md: 'MD', csv: 'CSV', txt: 'TXT' };
  return map[ext] || 'FILE';
}

function formatFileSize(size: number): string {
  return size > 1024 * 1024 ? `${(size / (1024 * 1024)).toFixed(1)}MB` : `${(size / 1024).toFixed(0)}KB`;
}

function addFiles(files: FileList | File[]) {
  const available = Math.max(0, 20 - fileQueue.length);
  const selected = Array.from(files);
  if (selected.length > available) MessagePlugin.warning(`一次最多导入 20 个文件，已保留前 ${available} 个`);
  for (const file of selected.slice(0, available)) {
    if (fileQueue.some(item => item.file.name === file.name && item.file.size === file.size && item.file.lastModified === file.lastModified)) {
      MessagePlugin.warning(`${file.name} 已在队列中`);
      continue;
    }
    const size = formatFileSize(file.size);
    if (file.size > 50 * 1024 * 1024) {
      fileQueue.push({ key: fileKey(file), name: file.name, size, meta: '文件超过 50MB 限制', progress: 0, state: 'failed', file, started: true, retrying: false, oversized: true, failureKind: 'validation' });
      continue;
    }
    fileQueue.push({ key: fileKey(file), name: file.name, size, meta: '等待开始导入', progress: 0, state: 'waiting', file, started: false, retrying: false });
  }
}

function fileKey(file: File) { return `${file.name}-${file.size}-${file.lastModified}`; }

function fileDescription(item: QueueItem): string {
  if (item.state === 'waiting') return `${item.size} · 等待开始导入`;
  if (item.state === 'uploading') return `${item.size} · 上传中 ${item.progress}%`;
  if (item.state === 'parsing') return `正在解析 ${item.progress}%`;
  if (item.state === 'completed') return '已入库，可在问答中检索';
  return item.meta;
}

function statusText(item: QueueItem): string {
  if (item.state === 'waiting') return '等待中';
  if (item.state === 'uploading') return `上传中 ${item.progress}%`;
  if (item.state === 'parsing') return `解析中 ${item.progress}%`;
  if (item.state === 'completed') return '已完成';
  return '失败';
}

function statusVariant(item: QueueItem): string {
  if (item.state === 'waiting') return 'gray';
  if (item.state === 'completed') return 'green';
  if (item.state === 'failed') return 'red';
  return 'blue';
}

function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement;
  if (target.files?.length) addFiles(target.files);
  target.value = '';
}

function handleDrop(event: DragEvent) {
  if (event.dataTransfer?.files?.length) addFiles(event.dataTransfer.files);
}

function triggerFileInput() { fileInput.value?.click(); }

function removeFile(index: number) {
  const item = fileQueue[index];
  if (!item) return;
  stopPolling(item.key);
  uploadControllers.get(item.key)?.abort();
  uploadControllers.delete(item.key);
  fileQueue.splice(index, 1);
}

function cancelUpload(item: QueueItem) {
  uploadControllers.get(item.key)?.abort();
  uploadControllers.delete(item.key);
  resetToWaiting(item);
}

function stopWaiting(item: QueueItem) {
  stopPolling(item.key);
  const index = fileQueue.indexOf(item);
  if (index >= 0) fileQueue.splice(index, 1);
  MessagePlugin.info(`${item.name} 已移出队列，服务器解析任务可能仍在继续`);
}

function resetToWaiting(item: QueueItem) {
  item.state = 'waiting';
  item.progress = 0;
  item.meta = '等待开始导入';
  item.started = false;
  item.retrying = false;
  item.notified = false;
  item.documentId = undefined;
  item.failureKind = undefined;
}

function viewFile(item: QueueItem) {
  const spaceId = props.spaceId?.trim();
  if (!spaceId || !item.documentId) return;
  close();
  router.push({ name: 'spaceDocuments', params: { id: spaceId }, query: { document: item.documentId } });
}

async function startImport() {
  if (uploading.value) return;
  const spaceId = props.spaceId?.trim();
  if (!spaceId) {
    MessagePlugin.warning('缺少空间信息，无法导入文档');
    return;
  }
  uploading.value = true;
  try {
    for (const item of fileQueue) {
      if (item.state !== 'waiting') continue;
      if (item.documentId) await restartParsing(item);
      else await uploadFile(item, spaceId);
    }
  } finally {
    uploading.value = false;
  }
}

async function uploadFile(item: QueueItem, spaceId?: string) {
  const targetSpaceId = spaceId || props.spaceId?.trim();
  if (!targetSpaceId) {
    markFailed(item, '缺少空间信息，无法上传');
    MessagePlugin.warning('缺少空间信息，无法导入文档');
    return;
  }
  stopPolling(item.key);
  const controller = new AbortController();
  uploadControllers.set(item.key, controller);
  item.started = true;
  item.state = 'uploading';
  item.meta = '正在上传';
  item.progress = 0;
  try {
    const response = await uploadDocument({ space_id: targetSpaceId, file: item.file, tags: confirmedTags.value || undefined }, percent => { item.progress = percent; }, controller.signal);
    if (controller.signal.aborted) return;
    item.documentId = response.data.document.id;
    item.state = 'parsing';
    item.progress = 0;
    item.meta = '上传完成，等待解析';
    schedulePoll(item, 0);
  } catch (error: unknown) {
    if (!controller.signal.aborted) {
      const duplicateId = duplicateDocumentId(error);
      if (duplicateId) {
        item.documentId = duplicateId;
        markFailed(item, '相同内容的文档已存在，请查看已有文档', 'duplicate');
      } else {
        markFailed(item, errorMessage(error, '上传失败，请重试'), 'upload');
      }
    }
  } finally {
    uploadControllers.delete(item.key);
  }
}

async function restartParsing(item: QueueItem) {
  if (!item.documentId) return;
  await reparseDocument(item.documentId);
  item.started = true;
  item.state = 'parsing';
  item.progress = 0;
  item.meta = '已重新提交解析';
  schedulePoll(item, 0);
}

async function retryFile(item: QueueItem) {
  if (item.retrying || item.oversized) return;
  item.retrying = true;
  item.notified = false;
  try {
    if (item.failureKind === 'status') await refreshParseStatus(item);
    else if (item.documentId) await restartParsing(item);
    else await uploadFile(item);
  } catch (error: unknown) {
    markFailed(item, errorMessage(error, '重试失败，请稍后再试'), item.failureKind || 'upload');
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
      markFailed(item, response.data.error || '文档解析失败，请重试', 'parse');
      return;
    }
    item.state = 'parsing';
    item.meta = status === 'pending' ? '等待解析任务' : '正在解析文档内容';
    schedulePoll(item);
  } catch (error: unknown) {
    markFailed(item, errorMessage(error, '解析状态获取失败，请刷新状态'), 'status');
  }
}

async function refreshParseStatus(item: QueueItem) {
  item.state = 'parsing';
  item.meta = '正在刷新解析状态';
  await pollParseStatus(item);
}

function retryText(item: QueueItem) {
  return item.failureKind === 'status' ? '刷新状态' : '重试';
}

function markFailed(item: QueueItem, reason: string, kind: FailureKind = 'parse') {
  stopPolling(item.key);
  item.state = 'failed';
  item.meta = reason;
  item.failureKind = kind;
}

function stopPolling(key?: string) {
  if (!key) return;
  const timer = pollTimers.get(key);
  if (timer !== undefined) window.clearTimeout(timer);
  pollTimers.delete(key);
}

function close() { emit('update:open', false); }

watch(() => props.open, value => {
  document.body.style.overflow = value ? 'hidden' : '';
});
onUnmounted(() => {
  document.body.style.overflow = '';
  pollTimers.forEach(timer => window.clearTimeout(timer));
  pollTimers.clear();
  uploadControllers.forEach(controller => controller.abort());
  uploadControllers.clear();
});

function errorMessage(error: unknown, fallback: string) {
  if (typeof error !== 'object' || !error) return fallback;
  const value = error as { message?: string; response?: { data?: { error?: string; message?: string } } };
  return value.response?.data?.error || value.response?.data?.message || value.message || fallback;
}

function duplicateDocumentId(error: unknown): string {
  if (typeof error !== 'object' || !error) return '';
  const value = error as { response?: { status?: number; data?: { document_id?: string } } };
  return value.response?.status === 409 ? value.response.data?.document_id || '' : '';
}
</script>

<style scoped>
.drawer-overlay { position: fixed; inset: 0; height: 100vh; height: 100dvh; z-index: var(--z-modal, 300); display: grid; grid-template-rows: minmax(0, 1fr); justify-items: end; overflow: hidden; background: rgba(15, 23, 19, .42); backdrop-filter: blur(4px); }
.drawer { width: 720px; max-width: 100vw; min-height: 0; height: 100%; max-height: 100vh; max-height: 100dvh; overflow: hidden; display: grid; grid-template-rows: auto minmax(0, 1fr) auto; background: var(--ink-50); border-left: 1px solid rgba(255,255,255,.32); box-shadow: -32px 0 80px rgba(0,0,0,.20); animation: drawer-in .24s var(--ease-out-expo); }
.drawer-head { padding: 24px 28px 18px; background: white; border-bottom: 1px solid var(--ink-200); display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-5); }
.drawer-head h2 { margin: var(--space-1) 0 0; font-size: var(--text-2xl); letter-spacing: -.035em; }
.drawer-body { min-height: 0; padding: 22px 28px; overflow: auto; display: grid; align-content: start; gap: var(--space-5); }
.drawer-foot { padding: var(--space-4) 28px; border-top: 1px solid var(--ink-200); background: white; display: flex; justify-content: space-between; align-items: center; gap: var(--space-4); }
.drawer-foot .subtle { margin: 0; }
.eyebrow { margin: 0 0 var(--space-1); color: var(--brand-700); font-size: var(--text-xs); font-weight: var(--font-weight-bold); letter-spacing: .08em; text-transform: uppercase; }
.subtle { margin: var(--space-2) 0 0; color: var(--ink-600); font-size: var(--text-sm); }
.actions, .queue-summary, .file-actions { display: flex; align-items: center; }
.actions { gap: var(--space-3); }
.queue-summary { flex-wrap: wrap; justify-content: flex-end; gap: var(--space-2); }
.btn { min-height: 36px; border: 1px solid transparent; border-radius: var(--radius-sm); padding: 8px 14px; font-weight: var(--font-weight-semibold); font-size: var(--text-sm); cursor: pointer; }
.btn.primary { background: var(--ink-900); color: white; border-color: var(--ink-900); }
.btn.primary:disabled, .btn.secondary:disabled { opacity: .5; cursor: not-allowed; }
.btn.secondary { background: white; color: var(--ink-700); border-color: var(--ink-200); }
.btn.ghost { background: transparent; color: var(--ink-700); }
.close { width: 36px; height: 36px; flex: 0 0 36px; border-radius: var(--radius-pill); border: 1px solid var(--ink-200); background: var(--ink-50); color: var(--ink-700); font-weight: var(--font-weight-bold); cursor: pointer; }
.notice { border: 1px solid var(--brand-200); background: var(--brand-50); color: var(--brand-900); border-radius: var(--radius-md); padding: var(--space-4); display: grid; grid-template-columns: 24px 1fr; gap: var(--space-3); font-size: var(--text-sm); }
.notice.danger { border-color: var(--danger-200); background: var(--danger-50); color: var(--danger-500); }
.notice-mark { width: 24px; height: 24px; border-radius: 50%; background: var(--brand-600); color: white; display: grid; place-items: center; font-weight: var(--font-weight-bold); font-size: var(--text-xs); }
.notice.danger .notice-mark { background: var(--danger-500); }
.panel { overflow: hidden; background: white; border: 1px solid var(--ink-200); border-radius: var(--radius-lg); box-shadow: var(--shadow-sm); }
.panel-head { padding: var(--space-5); border-bottom: 1px solid var(--ink-200); display: flex; justify-content: space-between; align-items: center; gap: var(--space-4); }
.panel-title { margin: 0; font-size: var(--text-base); }
.panel-body { padding: var(--space-5); }
.upload-zone { min-height: 172px; border: 1.5px dashed var(--brand-400); border-radius: var(--radius-lg); background: linear-gradient(180deg, var(--brand-50), white); display: grid; place-items: center; text-align: center; padding: var(--space-8); cursor: pointer; }
.upload-icon { width: 44px; height: 44px; border-radius: var(--radius-md); margin: 0 auto var(--space-3); background: var(--brand-100); color: var(--brand-800); display: grid; place-items: center; font-size: var(--text-xl); font-weight: var(--font-weight-bold); }
.upload-zone h3 { margin: 0; font-size: var(--text-base); }
.upload-zone .subtle { margin-top: var(--space-1); }
.select-file { margin-top: var(--space-4); }
.tag-field { display: grid; gap: var(--space-2); margin-top: var(--space-4); color: var(--ink-800); font-size: var(--text-sm); font-weight: var(--font-weight-semibold); }
.tag-field input { width: 100%; min-height: 40px; padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: white; font: inherit; font-weight: var(--font-weight-regular); }
.tag-field small { color: var(--ink-500); font-size: var(--text-xs); font-weight: var(--font-weight-regular); }
.file-list { display: grid; gap: 10px; }
.file-row { display: grid; grid-template-columns: 44px minmax(0, 1fr) 112px 94px 76px; align-items: center; gap: var(--space-3); padding: var(--space-3); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: white; }
.file-icon { width: 44px; height: 44px; border-radius: var(--radius-md); display: grid; place-items: center; background: var(--brand-50); color: var(--brand-800); border: 1px solid var(--brand-100); font-size: var(--text-xs); font-weight: var(--font-weight-bold); }
.file-copy { min-width: 0; }
.file-copy strong, .file-copy small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-copy strong { color: var(--ink-900); font-size: var(--text-sm); }
.file-copy small { margin-top: var(--space-1); color: var(--ink-500); font-size: var(--text-xs); }
.file-copy small.danger, .import-error { color: var(--danger-500); }
.progress { width: 112px; height: 8px; border-radius: var(--radius-pill); background: var(--ink-100); overflow: hidden; }
.progress > span { display: block; height: 100%; border-radius: inherit; background: linear-gradient(90deg, var(--brand-400), var(--brand-700)); transition: width var(--duration-normal) var(--ease-in-out); }
.progress.failed > span { background: var(--danger-500); }
.file-row > .badge { justify-content: center; white-space: nowrap; }
.file-actions { width: 76px; justify-content: flex-end; }
.row-button { width: 76px; min-height: 32px; padding: 6px 10px; white-space: nowrap; }
.badge { display: inline-flex; align-items: center; min-height: 24px; padding: 0 9px; border-radius: var(--radius-pill); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.badge.green { background: var(--success-50); color: var(--success-500); }
.badge.red { background: var(--danger-50); color: var(--danger-500); }
.badge.blue { background: var(--info-50); color: var(--info-500); }
.badge.gray { background: var(--ink-100); color: var(--ink-600); }
@keyframes drawer-in { from { transform: translateX(36px); opacity: .8; } to { transform: translateX(0); opacity: 1; } }
@media (max-width: 760px) {
  .drawer { width: 100vw; }
  .drawer-head, .drawer-body, .drawer-foot { padding-left: var(--space-4); padding-right: var(--space-4); }
  .drawer-foot { flex-direction: column; align-items: stretch; }
  .drawer-foot .actions { justify-content: flex-end; }
  .queue-head { align-items: flex-start; }
  .file-row { grid-template-columns: 44px minmax(0, 1fr) 76px; }
  .file-row > .progress { grid-column: 2 / -1; width: 100%; }
  .file-row > .badge { grid-column: 2; justify-self: start; }
  .file-actions { grid-column: 3; grid-row: 2; width: auto; }
  .row-button { width: auto; min-width: 76px; }
}
@media (max-height: 650px) {
  .drawer-head { padding-top: var(--space-4); padding-bottom: var(--space-3); }
  .drawer-head .subtle { margin-top: var(--space-1); }
  .drawer-body { padding-top: var(--space-3); padding-bottom: var(--space-3); gap: var(--space-3); }
  .notice, .panel-head, .panel-body { padding: var(--space-3); }
  .upload-zone { min-height: 140px; padding: var(--space-4); }
  .upload-icon { width: 36px; height: 36px; margin-bottom: var(--space-2); }
  .select-file { margin-top: var(--space-2); }
  .drawer-foot { padding-top: var(--space-3); padding-bottom: var(--space-3); }
}
@media (prefers-reduced-motion: reduce) { .drawer { animation-duration: .01ms; } }
</style>
