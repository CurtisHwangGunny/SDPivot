<template>
  <div class="model-configs">
    <div class="section-header">
      <div class="section-header-row">
        <div>
          <h2>{{ t('system.modelConfigs.title') }}</h2>
          <p class="section-description">{{ t('system.modelConfigs.description') }}</p>
        </div>
        <t-button theme="primary" @click="openCreate">
          <template #icon><t-icon name="add" /></template>
          {{ t('system.modelConfigs.add') }}
        </t-button>
      </div>
    </div>

    <t-loading :loading="loading" size="small">
      <div v-if="!loading && configs.length === 0" class="empty-state">
        <t-empty :description="t('system.modelConfigs.empty')">
          <t-button theme="primary" variant="outline" @click="openCreate">
            {{ t('system.modelConfigs.addFirst') }}
          </t-button>
        </t-empty>
      </div>

      <template v-else-if="selectedConfig">
        <div v-if="configs.length > 1" class="model-switch-row">
          <div>
            <span class="switch-label">{{ t('system.modelConfigs.current') }}</span>
            <t-select
              v-model="selectedId"
              :options="modelOptions"
              class="model-select"
            />
          </div>
          <t-tag theme="primary" variant="light">{{ selectedConfig.provider }}</t-tag>
        </div>

        <div class="config-card">
          <div class="config-card-header">
            <div>
              <h3>{{ selectedConfig.name }}</h3>
              <p>{{ selectedConfig.endpoint || t('system.modelConfigs.noEndpoint') }}</p>
            </div>
            <div class="config-actions">
              <t-button variant="outline" @click="openEdit(selectedConfig)">
                {{ t('common.edit') }}
              </t-button>
              <t-popconfirm
                :content="t('system.modelConfigs.deleteConfirm', { name: selectedConfig.name })"
                :confirm-btn="{ content: t('common.delete'), theme: 'danger' }"
                :cancel-btn="{ content: t('common.cancel') }"
                @confirm="removeSelected"
              >
                <t-button theme="danger" variant="text" :loading="deleting">
                  {{ t('common.delete') }}
                </t-button>
              </t-popconfirm>
            </div>
          </div>

          <div class="config-meta-grid">
            <div class="meta-item">
              <span>{{ t('system.modelConfigs.provider') }}</span>
              <strong>{{ selectedConfig.provider }}</strong>
            </div>
            <div class="meta-item">
              <span>{{ t('system.modelConfigs.apiKey') }}</span>
              <strong>{{ selectedConfig.api_key_configured ? t('system.modelConfigs.configured') : t('system.modelConfigs.notConfigured') }}</strong>
            </div>
            <div class="meta-item">
              <span>{{ t('system.modelConfigs.temperature') }}</span>
              <strong>{{ selectedConfig.temperature }}</strong>
            </div>
            <div class="meta-item">
              <span>{{ t('system.modelConfigs.maxTokens') }}</span>
              <strong>{{ selectedConfig.max_tokens }}</strong>
            </div>
            <div class="meta-item">
              <span>{{ t('system.modelConfigs.topP') }}</span>
              <strong>{{ selectedConfig.top_p }}</strong>
            </div>
          </div>
        </div>
      </template>
    </t-loading>

    <t-dialog
      v-model:visible="dialogVisible"
      :header="editingId ? t('system.modelConfigs.editTitle') : t('system.modelConfigs.createTitle')"
      width="620px"
      :confirm-btn="{ content: t('common.save'), loading: saving }"
      :cancel-btn="{ content: t('common.cancel'), disabled: saving }"
      :close-on-overlay-click="!saving"
      @confirm="saveConfig"
    >
      <t-form label-align="top">
        <div class="form-grid">
          <t-form-item :label="t('system.modelConfigs.name')" required-mark>
            <t-input v-model="form.name" :placeholder="t('system.modelConfigs.namePlaceholder')" />
          </t-form-item>
          <t-form-item :label="t('system.modelConfigs.provider')" required-mark>
            <t-input v-model="form.provider" :placeholder="t('system.modelConfigs.providerPlaceholder')" />
          </t-form-item>
        </div>
        <t-form-item :label="t('system.modelConfigs.endpoint')">
          <t-input v-model="form.endpoint" placeholder="https://api.example.com/v1" />
        </t-form-item>
        <t-form-item
          :label="t('system.modelConfigs.apiKey')"
          :help="editingId && apiKeyConfigured ? t('system.modelConfigs.apiKeyEditHelp') : t('system.modelConfigs.apiKeyHelp')"
        >
          <t-input v-model="form.apiKey" type="password" :placeholder="t('system.modelConfigs.apiKeyPlaceholder')" />
        </t-form-item>
        <div class="form-grid form-grid--generation">
          <t-form-item :label="t('system.modelConfigs.temperature')">
            <t-input-number v-model="form.temperature" :min="0" :max="2" :step="0.1" />
          </t-form-item>
          <t-form-item :label="t('system.modelConfigs.maxTokens')">
            <t-input-number v-model="form.maxTokens" :min="1" :step="1" />
          </t-form-item>
          <t-form-item :label="t('system.modelConfigs.topP')">
            <t-input-number v-model="form.topP" :min="0" :max="1" :step="0.1" />
          </t-form-item>
        </div>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  createPlatformModelConfig,
  deletePlatformModelConfig,
  listPlatformModelConfigs,
  updatePlatformModelConfig,
  type PlatformModelConfig,
  type PlatformModelConfigInput,
} from '@/api/system'

const { t } = useI18n()
const configs = ref<PlatformModelConfig[]>([])
const selectedId = ref('')
const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const dialogVisible = ref(false)
const editingId = ref('')
const apiKeyConfigured = ref(false)
const form = reactive({ name: '', provider: '', endpoint: '', apiKey: '', temperature: 0.7, maxTokens: 2048, topP: 1 })

const selectedConfig = computed(() =>
  configs.value.find((config) => config.id === selectedId.value) || configs.value[0],
)
const modelOptions = computed(() => configs.value.map((config) => ({
  label: `${config.name} · ${config.provider}`,
  value: config.id,
})))

async function loadConfigs(preferredId?: string) {
  loading.value = true
  try {
    configs.value = await listPlatformModelConfigs()
    const nextId = preferredId || selectedId.value
    selectedId.value = configs.value.some((config) => config.id === nextId)
      ? nextId
      : configs.value[0]?.id || ''
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('system.modelConfigs.loadFailed'))
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.name = ''
  form.provider = ''
  form.endpoint = ''
  form.apiKey = ''
  form.temperature = 0.7
  form.maxTokens = 2048
  form.topP = 1
  editingId.value = ''
  apiKeyConfigured.value = false
}

function openCreate() {
  resetForm()
  dialogVisible.value = true
}

function openEdit(config: PlatformModelConfig) {
  editingId.value = config.id
  apiKeyConfigured.value = config.api_key_configured
  form.name = config.name
  form.provider = config.provider
  form.endpoint = config.endpoint
  form.apiKey = ''
  form.temperature = config.temperature
  form.maxTokens = config.max_tokens
  form.topP = config.top_p
  dialogVisible.value = true
}

async function saveConfig() {
  const name = form.name.trim()
  const provider = form.provider.trim()
  const endpoint = form.endpoint.trim()
  if (!name || !provider) {
    MessagePlugin.warning(t('system.modelConfigs.required'))
    return
  }
  if (endpoint) {
    try {
      new URL(endpoint)
    } catch {
      MessagePlugin.warning(t('system.modelConfigs.invalidEndpoint'))
      return
    }
  }

  const input: PlatformModelConfigInput = {
    name,
    provider,
    endpoint,
    temperature: form.temperature,
    max_tokens: form.maxTokens,
    top_p: form.topP,
  }
  if (form.apiKey.trim()) input.api_key = form.apiKey.trim()

  saving.value = true
  try {
    const saved = editingId.value
      ? await updatePlatformModelConfig(editingId.value, input)
      : await createPlatformModelConfig(input)
    dialogVisible.value = false
    MessagePlugin.success(t('system.modelConfigs.saveSuccess'))
    await loadConfigs(saved.id)
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('system.modelConfigs.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function removeSelected() {
  if (!selectedConfig.value || deleting.value) return
  deleting.value = true
  try {
    await deletePlatformModelConfig(selectedConfig.value.id)
    MessagePlugin.success(t('system.modelConfigs.deleteSuccess'))
    await loadConfigs()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('system.modelConfigs.deleteFailed'))
  } finally {
    deleting.value = false
  }
}

onMounted(() => loadConfigs())
</script>

<style scoped>
.model-configs {
  display: grid;
  gap: 20px;
}
.section-header-row,
.config-card-header,
.model-switch-row,
.config-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.section-header h2,
.config-card h3 {
  margin: 0;
  color: var(--td-text-color-primary);
}
.section-description,
.config-card-header p {
  margin: 8px 0 0;
  color: var(--td-text-color-secondary);
  line-height: 1.6;
}
.empty-state {
  min-height: 320px;
  display: grid;
  place-items: center;
}
.model-switch-row,
.config-card {
  border: 1px solid var(--td-component-border);
  border-radius: 10px;
  background: var(--td-bg-color-container);
}
.model-switch-row {
  padding: 16px 18px;
}
.model-switch-row > div:first-child {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}
.switch-label,
.meta-item span {
  color: var(--td-text-color-secondary);
  font-size: 13px;
}
.model-select {
  width: 300px;
}
.config-card {
  padding: 22px;
}
.config-card-header {
  align-items: flex-start;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--td-component-stroke);
}
.config-card-header p {
  overflow-wrap: anywhere;
}
.config-meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-top: 20px;
}
.meta-item {
  display: grid;
  gap: 8px;
  padding: 14px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
}
.meta-item strong {
  color: var(--td-text-color-primary);
}
.meta-item--wide {
  grid-column: 1 / -1;
}
.meta-item pre {
  max-height: 260px;
  margin: 0;
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  color: var(--td-text-color-primary);
  font: 12px/1.6 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}
.form-grid--generation {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
@media (max-width: 720px) {
  .section-header-row,
  .config-card-header,
  .model-switch-row {
    align-items: stretch;
    flex-direction: column;
  }
  .model-switch-row > div:first-child,
  .config-actions {
    align-items: stretch;
    flex-direction: column;
  }
  .model-select,
  .config-actions > * {
    width: 100%;
  }
  .config-meta-grid,
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
