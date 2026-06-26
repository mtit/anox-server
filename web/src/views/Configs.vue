<template>
  <div class="configs-page">
    <div class="content-layout">
      <!-- Config List -->
      <div class="sidebar-card">
        <div class="sidebar-header">
          <span class="sidebar-title">{{ t('configs.configFiles') }}</span>
          <div class="sidebar-header-actions">
            <span class="sidebar-count">{{ Object.keys(configs).length }}</span>
            <a-button type="primary" size="mini" class="sidebar-create-btn" @click="showCreateModal = true">
              <template #icon><icon-plus /></template>
              {{ t('configs.create') }}
            </a-button>
          </div>
        </div>
        <div class="config-list">
          <div
            v-for="(config, name) in configs"
            :key="name"
            :class="['config-item', { active: selectedConfig === name }]"
            @click="selectConfig(name)"
          >
            <icon-file class="config-icon" />
            <div class="config-info">
              <div class="config-name">{{ name }}</div>
              <div class="config-version">{{ t('common.version') }} {{ config.version }}</div>
            </div>
            <a-button
              type="text"
              size="mini"
              status="danger"
              class="delete-btn"
              @click.stop="deleteConfig(name)"
            >
              <template #icon><icon-delete /></template>
            </a-button>
          </div>
        </div>
      </div>

      <!-- Config Editor -->
      <div class="editor-card">
        <template v-if="selectedConfig">
          <div class="editor-header">
            <div class="editor-title">
              <span class="filename">{{ selectedConfig }}</span>
              <span class="file-tag">.json</span>
            </div>
            <div v-if="selectedConfig === 'anox'" class="editor-hint">
              {{ t('configs.anoxRestartHint') }}
            </div>
            <div class="editor-actions">
              <a-radio-group v-if="!isMobile" v-model="editMode" type="button" size="small" class="edit-mode-switch">
                <a-radio value="kv">{{ t('configs.modeKv') }}</a-radio>
                <a-radio value="json">{{ t('configs.modeJson') }}</a-radio>
              </a-radio-group>
              <a-button type="primary" size="small" :loading="saving" @click="saveConfig">
                <template #icon><icon-save /></template>
                {{ t('common.save') }}
              </a-button>
            </div>
          </div>

          <!-- Key-Value Editor -->
          <div v-if="editMode === 'kv' || isMobile" class="kv-editor">
            <div class="kv-header">
              <div class="kv-col key-col">{{ t('configs.keyName') }}</div>
              <div class="kv-col value-col">{{ t('configs.value') }}</div>
              <div class="kv-col action-col"></div>
            </div>
            <div class="kv-body">
              <div
                v-for="(value, key) in editingValues"
                :key="key"
                class="kv-row"
              >
                <div class="kv-fields">
                  <div class="kv-col key-col">
                    <a-input v-model="keyNames[key]" :placeholder="t('configs.keyPlaceholder')" size="small" />
                  </div>
                  <div class="kv-col value-col">
                    <a-input v-model="editingValues[key]" :placeholder="t('configs.valuePlaceholder')" size="small" />
                  </div>
                </div>
                <div class="kv-col action-col">
                  <a-button type="text" size="mini" status="danger" class="kv-delete-btn" @click="removeKey(key)">
                    <template #icon><icon-delete /></template>
                  </a-button>
                </div>
              </div>
            </div>
            <a-button type="dashed" long class="add-btn" @click="addKey">
              <template #icon><icon-plus /></template>
              {{ t('configs.addItem') }}
            </a-button>
          </div>

          <!-- JSON Editor (desktop only) -->
          <div v-else-if="!isMobile" class="json-editor">
            <JsonCodeEditor v-model="jsonContent" />
          </div>
        </template>

        <!-- Empty State -->
        <div v-else class="editor-empty">
          <div class="empty-icon">
            <icon-settings />
          </div>
          <div class="empty-title">{{ t('configs.selectConfig') }}</div>
          <div class="empty-desc">{{ t('configs.selectConfigDesc') }}</div>
        </div>
      </div>
    </div>

    <!-- Create Config Modal -->
    <a-modal
      v-model:visible="showCreateModal"
      :title="t('configs.createModalTitle')"
      @ok="createConfig"
      @cancel="showCreateModal = false"
      :width="400"
    >
      <a-form :model="newConfigForm" layout="vertical">
        <a-form-item :label="t('configs.configName')">
          <a-input v-model="newConfigForm.name" :placeholder="t('configs.configNamePlaceholder')" />
          <template #extra>
            <span class="input-hint">{{ t('configs.configNameHint') }}</span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { getConfigs, getConfig, updateConfig, deleteConfigKey, type Config } from '@/api'
import JsonCodeEditor from '@/components/JsonCodeEditor.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { useI18n } from '@/i18n'
import {
  IconPlus,
  IconDelete,
  IconSave,
  IconFile,
  IconSettings,
} from '@arco-design/web-vue/es/icon'

const configs = ref<Record<string, Config>>({})
const selectedConfig = ref<string>('')
const editingValues = ref<Record<string, string>>({})
const keyNames = ref<Record<string, string>>({})
const editMode = ref<'kv' | 'json'>('kv')
const jsonContent = ref('')
const saving = ref(false)
const showCreateModal = ref(false)
const { isMobile } = useIsMobile()
const { t } = useI18n()

const newConfigForm = reactive({
  name: '',
})

const fetchConfigs = async () => {
  try {
    const response = await getConfigs()
    configs.value = response.data.configs || {}
  } catch (error) {
    console.error('Failed to fetch configs:', error)
  }
}

const selectConfig = (name: string) => {
  selectedConfig.value = name
  const config = configs.value[name]
  if (config) {
    editingValues.value = { ...config.values }
    keyNames.value = Object.fromEntries(
      Object.keys(config.values).map(key => [key, key])
    )
    updateJsonContent()
  }
}

const updateJsonContent = () => {
  jsonContent.value = JSON.stringify(editingValues.value, null, 2)
}

const addKey = () => {
  const newKey = `key_${Date.now()}`
  editingValues.value[newKey] = ''
  keyNames.value[newKey] = ''
}

const removeKey = (key: string) => {
  delete editingValues.value[key]
  delete keyNames.value[key]
}

const saveConfig = async () => {
  saving.value = true
  try {
    let values: Record<string, string>

    if (editMode.value === 'json' && !isMobile.value) {
      try {
        values = JSON.parse(jsonContent.value)
      } catch {
        Message.error(t('configs.jsonInvalid'))
        return
      }
    } else {
      values = {}
      for (const [oldKey, newKey] of Object.entries(keyNames.value)) {
        if (newKey.trim()) {
          values[newKey.trim()] = editingValues.value[oldKey] || ''
        }
      }
    }

    await updateConfig(selectedConfig.value, values)
    Message.success(t('configs.saved'))
    await fetchConfigs()
  } catch (error) {
    Message.error(t('configs.saveFailed'))
  } finally {
    saving.value = false
  }
}

const createConfig = async () => {
  if (!newConfigForm.name.trim()) {
    Message.warning(t('configs.configNameRequired'))
    return
  }

  try {
    await updateConfig(newConfigForm.name.trim(), {})
    Message.success(t('configs.created'))
    showCreateModal.value = false
    newConfigForm.name = ''
    await fetchConfigs()
    selectConfig(newConfigForm.name.trim())
  } catch (error) {
    Message.error(t('configs.createFailed'))
  }
}

const deleteConfig = (name: string) => {
  Modal.confirm({
    title: t('configs.deleteConfirmTitle'),
    content: t('configs.deleteConfirmContent', { name }),
    okText: t('common.delete'),
    cancelText: t('common.cancel'),
    onOk: async () => {
      try {
        const config = configs.value[name]
        if (config && config.values) {
          for (const key of Object.keys(config.values)) {
            await deleteConfigKey(name, key)
          }
        }
        Message.success(t('configs.deleted'))
        if (selectedConfig.value === name) {
          selectedConfig.value = ''
        }
        await fetchConfigs()
      } catch (error) {
        Message.error(t('configs.deleteFailed'))
      }
    },
  })
}

watch(editMode, (newMode) => {
  if (newMode === 'json') {
    const values: Record<string, string> = {}
    for (const [oldKey, newKey] of Object.entries(keyNames.value)) {
      if (newKey.trim()) {
        values[newKey.trim()] = editingValues.value[oldKey] || ''
      }
    }
    jsonContent.value = JSON.stringify(values, null, 2)
  } else {
    try {
      const values = JSON.parse(jsonContent.value)
      editingValues.value = values
      keyNames.value = Object.fromEntries(
        Object.keys(values).map(key => [key, key])
      )
    } catch {
      // Ignore parse error
    }
  }
})

onMounted(() => {
  fetchConfigs()
})
</script>

<style scoped>
.content-layout {
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: 16px;
  height: calc(100vh - 112px);
}

.sidebar-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid #f2f3f5;
}

.sidebar-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sidebar-create-btn {
  flex-shrink: 0;
}

.sidebar-title {
  font-size: 14px;
  font-weight: 500;
  color: #1d2129;
}

.sidebar-count {
  font-size: 12px;
  color: #86909c;
  background: #f2f3f5;
  padding: 2px 8px;
  border-radius: 10px;
}

.config-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.config-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.config-item:hover {
  background: #f7f8fa;
}

.config-item.active {
  background: #e8f3ff;
}

.config-item.active .config-name {
  color: #165dff;
}

.config-icon {
  font-size: 20px;
  color: #86909c;
}

.config-item.active .config-icon {
  color: #165dff;
}

.config-info {
  flex: 1;
  min-width: 0;
}

.config-name {
  font-size: 14px;
  font-weight: 500;
  color: #1d2129;
  margin-bottom: 2px;
}

.config-version {
  font-size: 12px;
  color: #86909c;
}

.delete-btn {
  opacity: 0;
  transition: opacity 0.2s;
}

.config-item:hover .delete-btn {
  opacity: 1;
}

.editor-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid #f2f3f5;
  gap: 12px;
}

.editor-hint {
  flex: 1;
  font-size: 12px;
  color: #ff7d00;
}

.editor-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filename {
  font-size: 16px;
  font-weight: 500;
  color: #1d2129;
}

.file-tag {
  font-size: 12px;
  color: #86909c;
  background: #f2f3f5;
  padding: 2px 6px;
  border-radius: 4px;
}

.editor-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.kv-editor {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
}

.kv-header {
  display: grid;
  grid-template-columns: 1fr 1fr 40px;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid #e5e6eb;
  margin-bottom: 8px;
}

.kv-col {
  font-size: 13px;
  font-weight: 500;
  color: #4e5969;
}

.kv-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.kv-fields {
  display: contents;
}

.kv-row {
  display: grid;
  grid-template-columns: 1fr 1fr 40px;
  gap: 12px;
  align-items: center;
}

.add-btn {
  margin-top: 12px;
}

.json-editor {
  flex: 1;
  min-height: 0;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
}

.editor-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
}

.empty-icon {
  font-size: 48px;
  color: #c9cdd4;
  margin-bottom: 16px;
}

.empty-title {
  font-size: 16px;
  font-weight: 500;
  color: #1d2129;
  margin-bottom: 4px;
}

.empty-desc {
  font-size: 13px;
  color: #86909c;
}

.input-hint {
  font-size: 12px;
  color: #86909c;
}

@media (max-width: 768px) {
  .content-layout {
    grid-template-columns: 1fr;
    height: auto;
  }

  .sidebar-card {
    max-height: 240px;
  }

  .editor-card {
    min-height: 420px;
  }

  .editor-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .editor-actions {
    width: 100%;
    justify-content: flex-end;
  }

  .kv-header {
    display: none;
  }

  .kv-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: #f7f8fa;
    border-radius: 8px;
  }

  .kv-fields {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 8px;
    min-width: 0;
  }

  .kv-col.action-col {
    flex-shrink: 0;
    align-self: center;
  }

  .delete-btn,
  .kv-delete-btn {
    opacity: 1;
    color: #f53f3f !important;
  }

  .delete-btn :deep(svg),
  .kv-delete-btn :deep(svg) {
    font-size: 16px;
  }
}
</style>
