<template>
  <div class="logs-page">
    <!-- Filter Card -->
    <div class="filter-card">
      <div class="filter-row compact">
        <a-select
          v-model="filterForm.service"
          :placeholder="t('logs.selectService')"
          allow-clear
          style="width: 140px"
          @change="onServiceChange"
        >
          <a-option v-for="service in services" :key="service" :value="service">
            {{ service }}
          </a-option>
        </a-select>

        <a-select
          v-model="filterForm.instance"
          :placeholder="t('logs.selectInstance')"
          allow-clear
          style="width: 324px"
          @change="onInstanceChange"
        >
          <a-option v-for="instance in instances" :key="instance" :value="instance">
            {{ instance }}
          </a-option>
        </a-select>

        <a-select
          v-model="filterForm.date"
          :placeholder="t('logs.selectDate')"
          allow-clear
          style="width: 120px"
          @change="onDateChange"
        >
          <a-option v-for="date in dates" :key="date" :value="date">
            {{ date }}
          </a-option>
        </a-select>

        <a-select
          v-model="filterForm.hour"
          :placeholder="t('logs.selectHour')"
          allow-clear
          style="width: 90px"
        >
          <a-option v-for="hour in hours" :key="hour" :value="hour">
            {{ hour }}:00
          </a-option>
        </a-select>

        <a-input
          v-model="filterForm.keyword"
          :placeholder="t('logs.keywordPlaceholder')"
          allow-clear
          style="width: 160px"
        />

        <a-button type="primary" @click="handleSearchLogs" :loading="loading">
          <template #icon><icon-search /></template>
          {{ t('common.search') }}
        </a-button>

        <a-button @click="showAlertModal = true">
          <template #icon><icon-settings /></template>
          {{ t('logs.alertConfig') }}
        </a-button>
      </div>
    </div>

    <!-- Results Card -->
    <div class="results-card">
      <div v-if="logs.length > 0" class="logs-table logs-desktop">
        <div class="table-header">
          <div class="th time-col">{{ t('logs.time') }}</div>
          <div class="th service-col">{{ t('logs.service') }}</div>
          <div class="th level-col">{{ t('logs.level') }}</div>
          <div class="th action-col">{{ t('logs.action') }}</div>
          <div class="th message-col">{{ t('logs.message') }}</div>
        </div>
        <div class="table-body">
          <div
            v-for="log in paginatedLogs"
            :key="log.time + log.message"
            class="table-row"
          >
            <div class="td time-col">{{ formatTime(log.time) }}</div>
            <div class="td service-col">
              <div class="service-tag">{{ log.service }}</div>
            </div>
            <div class="td level-col">
              <span :class="['level-tag', getLevelClass(log.level)]">
                {{ log.level }}
              </span>
            </div>
            <div class="td action-col">{{ log.action }}</div>
            <div class="td message-col" :title="log.message">{{ log.message }}</div>
          </div>
        </div>
      </div>

      <div v-if="logs.length > 0" class="logs-cards logs-mobile">
        <div
          v-for="log in paginatedLogs"
          :key="log.time + log.message"
          class="log-card"
        >
          <div class="log-card-header">
            <div class="log-card-time">{{ formatTime(log.time) }}</div>
            <span :class="['level-tag', getLevelClass(log.level)]">
              {{ log.level }}
            </span>
          </div>
          <div class="log-card-meta">
            <div class="service-tag">{{ log.service }}</div>
            <span v-if="log.action" class="log-card-action">{{ log.action }}</span>
          </div>
          <div class="log-card-message">{{ log.message }}</div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <icon-file />
        </div>
        <div class="empty-title">{{ t('logs.emptyTitle') }}</div>
        <div class="empty-desc">{{ t('logs.emptyDesc') }}</div>
      </div>

      <!-- Pagination -->
      <div v-if="logs.length > 0" class="pagination-wrapper">
        <SimplePagination
          v-if="isMobile"
          v-model:current="pagination.current"
          :page-size="pagination.pageSize"
          :total="logs.length"
        />
        <a-pagination
          v-else
          v-model:current="pagination.current"
          v-model:page-size="pagination.pageSize"
          :total="logs.length"
          show-total
          show-jumper
          show-page-size
          :page-size-options="[20, 50, 100]"
        />
      </div>
    </div>

    <!-- Alert Config Modal -->
    <a-modal
      v-model:visible="showAlertModal"
      :title="t('logs.alertModalTitle')"
      :width="720"
      @ok="saveAlertConfig"
      @cancel="showAlertModal = false"
    >
      <div class="alert-form">
        <!-- Basic Settings -->
        <div class="form-section">
          <h4 class="section-title">{{ t('logs.basicConfig') }}</h4>
          <div class="basic-config-row">
            <div class="basic-config-item">
              <label>{{ t('logs.enableAlert') }}</label>
              <a-switch v-model="alertForm.enabled" size="small" />
            </div>
            <div class="basic-config-item">
              <label>{{ t('logs.minLevel') }}</label>
              <a-select v-model="alertForm.min_level" size="small" style="width: 120px">
                <a-option value="Debug">Debug</a-option>
                <a-option value="Info">Info</a-option>
                <a-option value="Important">Important</a-option>
                <a-option value="Emergency">Emergency</a-option>
              </a-select>
            </div>
            <div class="basic-config-item">
              <label>{{ t('logs.enableDedup') }}</label>
              <a-switch v-model="alertForm.deduplicate" size="small" />
            </div>
            <div class="basic-config-item">
              <label>{{ t('logs.dedupWindow') }}</label>
              <a-input-number v-model="alertForm.deduplicate_window" :min="60" :max="3600" size="small" style="width: 90px" />
            </div>
          </div>
        </div>

        <!-- Channels -->
        <div class="form-section">
          <h4 class="section-title">{{ t('logs.alertChannels') }}</h4>
          
          <!-- WeChat Work -->
          <div class="channel-item">
            <div class="channel-header">
              <a-checkbox v-model="alertForm.channels" value="wechat">{{ t('logs.wechat') }}</a-checkbox>
            </div>
            <div v-if="alertForm.channels.includes('wechat')" class="channel-config">
              <div class="form-item full-width">
                <label>{{ t('logs.pushUrl') }}</label>
                <a-input v-model="alertForm.wechat_url" placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx" />
              </div>
            </div>
          </div>

          <!-- DingTalk -->
          <div class="channel-item">
            <div class="channel-header">
              <a-checkbox v-model="alertForm.channels" value="dingtalk">{{ t('logs.dingtalk') }}</a-checkbox>
            </div>
            <div v-if="alertForm.channels.includes('dingtalk')" class="channel-config">
              <div class="form-item full-width">
                <label>{{ t('logs.pushUrl') }}</label>
                <a-input v-model="alertForm.dingtalk_url" placeholder="https://oapi.dingtalk.com/robot/send?access_token=xxx" />
              </div>
            </div>
          </div>

          <!-- Feishu -->
          <div class="channel-item">
            <div class="channel-header">
              <a-checkbox v-model="alertForm.channels" value="feishu">{{ t('logs.feishu') }}</a-checkbox>
            </div>
            <div v-if="alertForm.channels.includes('feishu')" class="channel-config">
              <div class="form-item full-width">
                <label>{{ t('logs.pushUrl') }}</label>
                <a-input v-model="alertForm.feishu_url" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/xxx" />
              </div>
            </div>
          </div>

          <!-- SMS (阿里云) -->
          <div class="channel-item">
            <div class="channel-header">
              <a-checkbox v-model="alertForm.channels" value="sms">{{ t('logs.sms') }}</a-checkbox>
            </div>
            <div v-if="alertForm.channels.includes('sms')" class="channel-config">
              <div class="sms-config-grid">
                <div class="form-item">
                  <label>{{ t('logs.accessKeyId') }}</label>
                  <a-input v-model="alertForm.sms_access_key_id" placeholder="LTAIxxxxx" />
                </div>
                <div class="form-item">
                  <label>{{ t('logs.accessKeySecret') }}</label>
                  <a-input-password v-model="alertForm.sms_access_key_secret" placeholder="Your Secret Key" />
                </div>
                <div class="form-item">
                  <label>{{ t('logs.signName') }}</label>
                  <a-input v-model="alertForm.sms_sign_name" placeholder="阿里云短信测试" />
                </div>
                <div class="form-item">
                  <label>{{ t('logs.templateCode') }}</label>
                  <a-input v-model="alertForm.sms_template_code" placeholder="SMS_xxxxxx" />
                </div>
                <div class="form-item full-width">
                  <label>{{ t('logs.phoneNumbers') }}</label>
                  <a-input v-model="alertForm.sms_phone_numbers" placeholder="13800138000,13900139000" />
                  <span class="input-hint">{{ t('logs.phoneHint') }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  searchLogs as searchLogsApi,
  getLogServices,
  getLogInstances,
  getLogDates,
  getLogHours,
  getAlertConfig,
  updateAlertConfig,
  type LogEntry,
  type AlertConfig,
} from '@/api'
import SimplePagination from '@/components/SimplePagination.vue'
import { useIsMobile } from '@/composables/useIsMobile'
import { useI18n } from '@/i18n'
import {
  IconSearch,
  IconSettings,
  IconFile,
} from '@arco-design/web-vue/es/icon'

const { t } = useI18n()
const { isMobile } = useIsMobile()

const services = ref<string[]>([])
const instances = ref<string[]>([])
const dates = ref<string[]>([])
const hours = ref<string[]>([])
const logs = ref<LogEntry[]>([])
const loading = ref(false)
const showAlertModal = ref(false)

const filterForm = reactive({
  service: '',
  instance: '',
  date: '',
  hour: '',
  keyword: '',
})

// Alert form - simplified for new API
const alertForm = reactive({
  enabled: false,
  min_level: 'Emergency',
  channels: [] as string[],
  deduplicate: true,
  deduplicate_window: 300,
  // WeChat
  wechat_url: '',
  // DingTalk
  dingtalk_url: '',
  // Feishu
  feishu_url: '',
  // Aliyun SMS
  sms_access_key_id: '',
  sms_access_key_secret: '',
  sms_sign_name: '',
  sms_template_code: '',
  sms_phone_numbers: '',
})

const pagination = reactive({
  current: 1,
  pageSize: 20,
})

const paginatedLogs = computed(() => {
  const start = (pagination.current - 1) * pagination.pageSize
  const end = start + pagination.pageSize
  return logs.value.slice(start, end)
})

const getLevelClass = (level: string): string => {
  const map: Record<string, string> = {
    'Debug': 'level-debug',
    'Info': 'level-info',
    'Important': 'level-warn',
    'Emergency': 'level-error',
  }
  return map[level] || 'level-debug'
}

const formatTime = (time: string): string => {
  return new Date(time).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

const fetchServices = async () => {
  try {
    const response = await getLogServices()
    services.value = response.data.services || []
  } catch (error) {
    console.error('Failed to fetch services:', error)
  }
}

const onServiceChange = async () => {
  filterForm.instance = ''
  filterForm.date = ''
  filterForm.hour = ''
  instances.value = []
  dates.value = []
  hours.value = []

  if (filterForm.service) {
    try {
      const response = await getLogInstances(filterForm.service)
      instances.value = response.data.instances || []
    } catch (error) {
      console.error('Failed to fetch instances:', error)
    }
  }
}

const onInstanceChange = async () => {
  filterForm.date = ''
  filterForm.hour = ''
  dates.value = []
  hours.value = []

  if (filterForm.service && filterForm.instance) {
    try {
      const response = await getLogDates(filterForm.service, filterForm.instance)
      dates.value = response.data.dates || []
    } catch (error) {
      console.error('Failed to fetch dates:', error)
    }
  }
}

const onDateChange = async () => {
  filterForm.hour = ''
  hours.value = []

  if (filterForm.service && filterForm.instance && filterForm.date) {
    try {
      const response = await getLogHours(filterForm.service, filterForm.instance, filterForm.date)
      hours.value = response.data.hours || []
    } catch (error) {
      console.error('Failed to fetch hours:', error)
    }
  }
}

const handleSearchLogs = async () => {
  if (!filterForm.service || !filterForm.instance || !filterForm.date) {
    Message.warning(t('logs.filterRequired'))
    return
  }

  loading.value = true
  try {
    const response = await searchLogsApi({
      service: filterForm.service,
      instance: filterForm.instance,
      date: filterForm.date,
      hour: filterForm.hour,
      keyword: filterForm.keyword,
    })
    logs.value = response.data.logs || []
    pagination.current = 1
  } catch (error) {
    Message.error(t('logs.searchFailed'))
  } finally {
    loading.value = false
  }
}

const fetchAlertConfig = async () => {
  try {
    const response = await getAlertConfig()
    const config = response.data.config
    if (config) {
      alertForm.enabled = config.enabled
      alertForm.min_level = config.min_level || 'Emergency'
      alertForm.channels = config.channels || []
      alertForm.deduplicate = config.deduplicate !== false
      alertForm.deduplicate_window = config.deduplicate_window || 300
      
      // URLs
      alertForm.wechat_url = config.wechat_url || ''
      alertForm.dingtalk_url = config.dingtalk_url || ''
      alertForm.feishu_url = config.feishu_url || ''
      
      // SMS
      alertForm.sms_access_key_id = config.sms_access_key_id || ''
      alertForm.sms_access_key_secret = config.sms_access_key_secret || ''
      alertForm.sms_sign_name = config.sms_sign_name || ''
      alertForm.sms_template_code = config.sms_template_code || ''
      alertForm.sms_phone_numbers = config.sms_phone_numbers || ''
    }
  } catch (error) {
    console.error('Failed to fetch alert config:', error)
  }
}

const saveAlertConfig = async () => {
  try {
    const config = {
      enabled: alertForm.enabled,
      min_level: alertForm.min_level,
      channels: alertForm.channels,
      deduplicate: alertForm.deduplicate,
      deduplicate_window: alertForm.deduplicate_window,
      wechat_url: alertForm.wechat_url,
      dingtalk_url: alertForm.dingtalk_url,
      feishu_url: alertForm.feishu_url,
      sms_access_key_id: alertForm.sms_access_key_id,
      sms_access_key_secret: alertForm.sms_access_key_secret,
      sms_sign_name: alertForm.sms_sign_name,
      sms_template_code: alertForm.sms_template_code,
      sms_phone_numbers: alertForm.sms_phone_numbers,
    }
    await updateAlertConfig(config as AlertConfig)
    Message.success(t('logs.alertSaved'))
    showAlertModal.value = false
  } catch (error) {
    Message.error(t('logs.alertSaveFailed'))
  }
}

onMounted(() => {
  fetchServices()
  fetchAlertConfig()
})
</script>

<style scoped>
.filter-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  padding: 20px;
  margin-bottom: 16px;
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.filter-row.compact {
  gap: 8px;
}

.results-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.logs-table {
  width: 100%;
}

.logs-mobile {
  display: none;
}

.log-card {
  background: #f7f8fa;
  border-radius: 10px;
  padding: 12px;
}

.log-card + .log-card {
  margin-top: 12px;
}

.log-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.log-card-time {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 12px;
  color: #4e5969;
}

.log-card-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.log-card-action {
  font-size: 12px;
  color: #4e5969;
}

.log-card-message {
  font-size: 13px;
  color: #1d2129;
  line-height: 1.6;
  word-break: break-word;
  white-space: pre-wrap;
}

.table-header {
  display: grid;
  grid-template-columns: 140px 120px 80px 100px 1fr;
  gap: 12px;
  padding: 12px 20px;
  background: #f7f8fa;
  border-bottom: 1px solid #e5e6eb;
}

.th {
  font-size: 13px;
  font-weight: 500;
  color: #4e5969;
}

.table-body {
  max-height: 600px;
  overflow-y: auto;
}

.table-row {
  display: grid;
  grid-template-columns: 140px 120px 80px 100px 1fr;
  gap: 12px;
  padding: 12px 20px;
  border-bottom: 1px solid #f2f3f5;
  transition: background 0.2s;
}

.table-row:hover {
  background: #f7f8fa;
}

.table-row:last-child {
  border-bottom: none;
}

.td {
  font-size: 13px;
  color: #1d2129;
  display: flex;
  align-items: center;
  overflow: hidden;
}

.time-col {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 12px;
  color: #4e5969;
}

.service-tag {
  padding: 2px 8px;
  background: #e8f3ff;
  color: #165dff;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.level-tag {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.level-debug {
  background: #f2f3f5;
  color: #86909c;
}

.level-info {
  background: #e8f3ff;
  color: #165dff;
}

.level-warn {
  background: #fff7e8;
  color: #ff7d00;
}

.level-error {
  background: #ffe8e8;
  color: #f53f3f;
}

.message-col {
  color: #4e5969;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-state {
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

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid #f2f3f5;
}

/* Alert Form */
.alert-form {
  max-height: 500px;
  overflow-y: auto;
}

.form-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #1d2129;
  margin: 0 0 16px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid #f2f3f5;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 12px;
}

.form-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-item.full-width {
  grid-column: 1 / -1;
}

.form-item label {
  font-size: 12px;
  color: #4e5969;
  font-weight: 500;
}

.basic-config-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px 24px;
}

.basic-config-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 28px;
}

.basic-config-item label {
  font-size: 12px;
  color: #4e5969;
  font-weight: 500;
  white-space: nowrap;
  line-height: 28px;
}

.basic-config-item :deep(.arco-switch) {
  flex-shrink: 0;
}

.basic-config-item :deep(.arco-select),
.basic-config-item :deep(.arco-input-number) {
  flex-shrink: 0;
}

.channel-item {
  background: #f7f8fa;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
}

.channel-header {
  margin-bottom: 8px;
}

.channel-config {
  padding-top: 8px;
  border-top: 1px solid #e5e6eb;
}

.sms-config-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.input-hint {
  font-size: 12px;
  color: #86909c;
}

@media (max-width: 768px) {
  .filter-card {
    padding: 16px;
  }

  .filter-row.compact {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-row.compact :deep(.arco-select),
  .filter-row.compact :deep(.arco-input-wrapper),
  .filter-row.compact :deep(.arco-btn) {
    width: 100% !important;
  }

  .logs-desktop {
    display: none;
  }

  .logs-mobile {
    display: block;
    padding: 12px 16px;
  }

  .table-body {
    max-height: none;
  }

  .pagination-wrapper {
    justify-content: center;
    padding: 12px 16px;
  }

  .basic-config-row {
    flex-direction: column;
    align-items: stretch;
  }

  .basic-config-item {
    width: 100%;
    justify-content: space-between;
  }

  .basic-config-item :deep(.arco-select),
  .basic-config-item :deep(.arco-input-number) {
    flex: 1;
    max-width: 180px;
  }

  .sms-config-grid {
    grid-template-columns: 1fr;
  }
}
</style>
