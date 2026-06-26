<template>
  <div class="services-page">
    <!-- Services List -->
    <div v-if="services.length > 0" class="services-list">
      <div
        v-for="service in services"
        :key="service.name"
        class="service-card"
      >
        <div class="service-header" @click="toggleService(service.name)">
          <div class="service-info">
            <icon-apps class="service-icon" />
            <span class="service-name">{{ service.name }}</span>
            <span class="service-count">{{ service.instance_count }} {{ t('services.instances') }}</span>
          </div>
          <div class="service-actions">
            <icon-down
              :class="['expand-icon', { expanded: expandedServices.includes(service.name) }]"
            />
          </div>
        </div>

        <div
          v-show="expandedServices.includes(service.name)"
          class="service-content"
        >
          <!-- Desktop Table -->
          <div class="instances-table instances-desktop">
            <div class="table-header">
              <div class="th">{{ t('services.instanceId') }}</div>
              <div class="th">{{ t('services.registerTime') }}</div>
              <div class="th">{{ t('services.cpuUsage') }}</div>
              <div class="th">{{ t('services.memoryUsage') }}</div>
            </div>
            <div
              v-for="instance in service.instances"
              :key="instance.id"
              class="table-row"
            >
              <div class="td instance-name" :title="instance.id">{{ instance.id }}</div>
              <div class="td">{{ formatDate(instance.registered_at) }}</div>
              <div class="td">
                <div class="metric-bar">
                  <div class="progress-track">
                    <div
                      class="progress-fill"
                      :style="{ width: instance.cpu_percent + '%', background: getCpuColor(instance.cpu_percent) }"
                    />
                  </div>
                  <span class="metric-value">{{ instance.cpu_percent.toFixed(1) }}%</span>
                </div>
              </div>
              <div class="td">
                <div class="metric-bar">
                  <div class="progress-track">
                    <div
                      class="progress-fill"
                      :style="{ width: getMemoryPercent(instance) + '%', background: getMemoryColor(getMemoryPercent(instance)) }"
                    />
                  </div>
                  <span class="metric-value">{{ getMemoryPercent(instance).toFixed(1) }}%</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Mobile Cards -->
          <div class="instances-cards instances-mobile">
            <div
              v-for="instance in service.instances"
              :key="instance.id"
              class="instance-card"
            >
              <div class="instance-card-header">
                <div class="instance-id" :title="instance.id">{{ instance.id }}</div>
                <div class="instance-time">{{ formatDate(instance.registered_at) }}</div>
              </div>
              <div class="instance-metrics">
                <div class="metric-item">
                  <span class="metric-label">{{ t('services.cpuUsage') }}</span>
                  <div class="metric-bar">
                    <div class="progress-track">
                      <div
                        class="progress-fill"
                        :style="{ width: instance.cpu_percent + '%', background: getCpuColor(instance.cpu_percent) }"
                      />
                    </div>
                    <span class="metric-value">{{ instance.cpu_percent.toFixed(1) }}%</span>
                  </div>
                </div>
                <div class="metric-item">
                  <span class="metric-label">{{ t('services.memoryUsage') }}</span>
                  <div class="metric-bar">
                    <div class="progress-track">
                      <div
                        class="progress-fill"
                        :style="{ width: getMemoryPercent(instance) + '%', background: getMemoryColor(getMemoryPercent(instance)) }"
                      />
                    </div>
                    <span class="metric-value">{{ getMemoryPercent(instance).toFixed(1) }}%</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-card">
      <div class="empty-icon">
        <icon-apps />
      </div>
      <div class="empty-title">{{ t('services.emptyTitle') }}</div>
      <div class="empty-desc">{{ t('services.emptyDesc') }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { getServices } from '@/api'
import { useI18n } from '@/i18n'
import type { Service, Instance } from '@/api'
import {
  IconApps,
  IconDown,
} from '@arco-design/web-vue/es/icon'

const { t } = useI18n()

const services = ref<Service[]>([])
const expandedServices = ref<string[]>([])
let pollTimer: ReturnType<typeof setInterval> | null = null

const getMemoryPercent = (instance: Instance): number => {
  if (!instance.memory_total_mb) return 0
  const used = instance.memory_total_mb - instance.memory_avail_mb
  return (used / instance.memory_total_mb) * 100
}

const getCpuColor = (percent: number): string => {
  if (percent < 50) return '#00b42a'
  if (percent < 80) return '#ff7d00'
  return '#f53f3f'
}

const getMemoryColor = (percent: number): string => {
  if (percent < 50) return '#00b42a'
  if (percent < 80) return '#ff7d00'
  return '#f53f3f'
}

const formatDate = (dateStr: string): string => {
  return new Date(dateStr).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const toggleService = (name: string) => {
  const index = expandedServices.value.indexOf(name)
  if (index > -1) {
    expandedServices.value.splice(index, 1)
  } else {
    expandedServices.value.push(name)
  }
}

const fetchServices = async () => {
  try {
    const response = await getServices()
    const newServices = response.data.services || []
    
    // Preserve expanded state and merge data to avoid UI flicker
    newServices.forEach(service => {
      const existingService = services.value.find(s => s.name === service.name)
      if (existingService) {
        // Update instance data while preserving the array reference for reactivity
        service.instances?.forEach((instance: Instance) => {
          const existingInstance = existingService.instances?.find(i => i.id === instance.id)
          if (existingInstance) {
            // Update existing instance properties
            Object.assign(existingInstance, instance)
          }
        })
      }
    })
    
    // If first load or service list changed, replace entirely
    if (services.value.length === 0 || newServices.length !== services.value.length) {
      services.value = newServices
    }
  } catch (error) {
    console.error('Failed to fetch services:', error)
  }
}

const startPolling = () => {
  fetchServices()
  pollTimer = setInterval(fetchServices, 3000) // Poll every 3 seconds
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

onMounted(() => {
  startPolling()
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.services-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.service-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.service-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  cursor: pointer;
  transition: background 0.2s;
}

.service-header:hover {
  background: #f7f8fa;
}

.service-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.service-icon {
  font-size: 20px;
  color: #165dff;
}

.service-name {
  font-size: 16px;
  font-weight: 500;
  color: #1d2129;
}

.service-count {
  font-size: 13px;
  color: #86909c;
  padding: 2px 8px;
  background: #f2f3f5;
  border-radius: 4px;
}

.expand-icon {
  font-size: 16px;
  color: #4e5969;
  transition: transform 0.3s;
}

.expand-icon.expanded {
  transform: rotate(180deg);
}

.service-content {
  border-top: 1px solid #f2f3f5;
  padding: 16px 20px;
}

.instances-table {
  width: 100%;
}

.instances-mobile {
  display: none;
}

.instance-card {
  background: #f7f8fa;
  border-radius: 10px;
  padding: 12px;
}

.instance-card + .instance-card {
  margin-top: 12px;
}

.instance-card-header {
  margin-bottom: 12px;
}

.instance-id {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 12px;
  color: #1d2129;
  word-break: break-all;
  line-height: 1.5;
}

.instance-time {
  margin-top: 4px;
  font-size: 12px;
  color: #86909c;
}

.instance-metrics {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.metric-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.metric-label {
  font-size: 12px;
  color: #4e5969;
}

.table-header {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 16px;
  padding: 8px 0;
  border-bottom: 1px solid #e5e6eb;
  margin-bottom: 8px;
}

.th {
  font-size: 13px;
  font-weight: 500;
  color: #4e5969;
}

.table-row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid #f2f3f5;
}

.table-row:last-child {
  border-bottom: none;
}

.td {
  font-size: 13px;
  color: #1d2129;
  display: flex;
  align-items: center;
}

.instance-name {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 12px;
  color: #4e5969;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.metric-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.progress-track {
  flex: 1;
  max-width: 100px;
  height: 6px;
  background: #f2f3f5;
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.3s ease;
  min-width: 0;
}

.metric-value {
  font-size: 12px;
  color: #4e5969;
  min-width: 40px;
  flex-shrink: 0;
}

.empty-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
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

@media (max-width: 768px) {
  .service-header {
    padding: 14px 16px;
  }

  .service-content {
    padding: 12px 16px;
    overflow-x: visible;
  }

  .instances-desktop {
    display: none;
  }

  .instances-mobile {
    display: block;
  }

  .instances-cards .metric-bar .progress-track {
    max-width: none;
  }

  .service-name {
    font-size: 15px;
  }

  .instance-id {
    font-size: 15px;
    font-weight: 600;
    color: #1d2129;
  }
}
</style>
