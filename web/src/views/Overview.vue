<template>
  <div class="overview-page">
    <!-- Notice Banner -->
    <div :class="['notice-card', hostNotice.level]">
      <icon-info-circle class="notice-icon" />
      <div class="notice-content">
        <span class="notice-text">{{ t('overview.hostNotice', { host: overview.host, port: overview.port }) }}</span>
        <span v-if="hostNotice.warning" class="notice-warning">{{ hostNotice.warning }}</span>
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon blue">
          <icon-apps />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ overview.service_count }}</div>
          <div class="stat-label">{{ t('overview.serviceCount') }}</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon green">
          <icon-desktop />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ overview.instance_count }}</div>
          <div class="stat-label">{{ t('overview.instanceCount') }}</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon orange">
          <icon-mind-mapping />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ overview.cpu_cores }}</div>
          <div class="stat-label">{{ t('overview.cpuCores') }}</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon red">
          <icon-safe />
        </div>
        <div class="stat-info">
          <div class="stat-value stat-value-nowrap">{{ formatMemory(overview.memory_mb) }}</div>
          <div class="stat-label">{{ t('overview.memoryCapacity') }}</div>
        </div>
      </div>
    </div>

    <!-- Charts -->
    <div class="charts-grid">
      <div class="chart-card">
        <div class="chart-header">
          <h3 class="chart-title">{{ t('overview.cpuChartTitle') }}</h3>
          <span class="chart-subtitle">{{ t('overview.cpuChartSubtitle') }}</span>
        </div>
        <v-chart class="chart" :option="cpuChartOption" autoresize />
      </div>

      <div class="chart-card">
        <div class="chart-header">
          <h3 class="chart-title">{{ t('overview.memoryChartTitle') }}</h3>
          <span class="chart-subtitle">{{ t('overview.memoryChartSubtitle') }}</span>
        </div>
        <v-chart class="chart" :option="memoryChartOption" autoresize />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { getOverview, getSystemMetrics } from '@/api'
import { useI18n } from '@/i18n'
import {
  IconApps,
  IconDesktop,
  IconMindMapping,
  IconSafe,
  IconInfoCircle,
} from '@arco-design/web-vue/es/icon'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent])

const { t, messages } = useI18n()

interface OverviewData {
  service_count: number
  instance_count: number
  cpu_cores: number
  memory_mb: number
  host: string
  port: string
}

const overview = ref<OverviewData>({
  service_count: 0,
  instance_count: 0,
  cpu_cores: 0,
  memory_mb: 0,
  host: '',
  port: '',
})

type HostNoticeLevel = 'safe' | 'warn' | 'danger'

const hostNotice = computed(() => {
  void messages.value
  return classifyHostNotice(overview.value.host)
})

function parseIPv4(host: string): number[] | null {
  const parts = host.split('.')
  if (parts.length !== 4) return null

  const nums: number[] = []
  for (const part of parts) {
    if (part === '' || !/^\d+$/.test(part)) return null
    const n = Number(part)
    if (n < 0 || n > 255) return null
    nums.push(n)
  }
  return nums
}

function isPrivateIPv4(octets: number[]): boolean {
  const [a, b] = octets
  if (a === 10) return true
  if (a === 172 && b >= 16 && b <= 31) return true
  if (a === 192 && b === 168) return true
  if (a === 127) return true
  if (a === 169 && b === 254) return true
  return false
}

function isPrivateIPv6(host: string): boolean {
  const normalized = host.toLowerCase()
  if (normalized === '::1') return true
  if (normalized.startsWith('fc') || normalized.startsWith('fd')) return true
  if (normalized.startsWith('fe80')) return true
  return false
}

function classifyHostNotice(host: string): { level: HostNoticeLevel; warning: string } {
  const trimmed = host.trim().toLowerCase()

  if (!trimmed || trimmed === '0.0.0.0' || trimmed === '::' || trimmed === '[::]') {
    return {
      level: 'danger',
      warning: t('overview.hostWarningDanger'),
    }
  }

  if (trimmed === 'localhost') {
    return { level: 'safe', warning: '' }
  }

  const ipv4 = parseIPv4(trimmed)
  if (ipv4) {
    if (isPrivateIPv4(ipv4)) {
      return { level: 'safe', warning: '' }
    }
    return {
      level: 'warn',
      warning: t('overview.hostWarningPublic'),
    }
  }

  if (trimmed.includes(':')) {
    if (isPrivateIPv6(trimmed)) {
      return { level: 'safe', warning: '' }
    }
    return {
      level: 'warn',
      warning: t('overview.hostWarningPublic'),
    }
  }

  return {
    level: 'warn',
    warning: t('overview.hostWarningPublic'),
  }
}

const cpuData = reactive({
  times: [] as string[],
  values: [] as number[],
})

const memoryData = reactive({
  times: [] as string[],
  values: [] as number[],
})

const cpuChartOption = ref({
  tooltip: {
    trigger: 'axis',
    backgroundColor: 'rgba(255, 255, 255, 0.95)',
    borderColor: '#e5e6eb',
    borderWidth: 1,
    textStyle: { color: '#1d2129' },
  },
  xAxis: {
    type: 'category',
    data: cpuData.times,
    boundaryGap: false,
    axisLine: { lineStyle: { color: '#e5e6eb' } },
    axisLabel: { color: '#4e5969', fontSize: 11 },
  },
  yAxis: {
    type: 'value',
    min: 0,
    max: 100,
    axisLabel: { formatter: '{value}%', color: '#4e5969', fontSize: 11 },
    splitLine: { lineStyle: { color: '#f2f3f5' } },
  },
  series: [
    {
      name: 'CPU占用',
      type: 'line',
      smooth: true,
      symbol: 'none',
      data: cpuData.values,
      lineStyle: { color: '#165dff', width: 2 },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: 'rgba(22, 93, 255, 0.2)' },
            { offset: 1, color: 'rgba(22, 93, 255, 0)' },
          ],
        },
      },
    },
  ],
  grid: {
    left: 16,
    right: 16,
    top: 16,
    bottom: 16,
    containLabel: true,
  },
})

const memoryChartOption = ref({
  tooltip: {
    trigger: 'axis',
    backgroundColor: 'rgba(255, 255, 255, 0.95)',
    borderColor: '#e5e6eb',
    borderWidth: 1,
    textStyle: { color: '#1d2129' },
  },
  xAxis: {
    type: 'category',
    data: memoryData.times,
    boundaryGap: false,
    axisLine: { lineStyle: { color: '#e5e6eb' } },
    axisLabel: { color: '#4e5969', fontSize: 11 },
  },
  yAxis: {
    type: 'value',
    min: 0,
    max: 100,
    axisLabel: { formatter: '{value}%', color: '#4e5969', fontSize: 11 },
    splitLine: { lineStyle: { color: '#f2f3f5' } },
  },
  series: [
    {
      name: '内存占用',
      type: 'line',
      smooth: true,
      symbol: 'none',
      data: memoryData.values,
      lineStyle: { color: '#00b42a', width: 2 },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: 'rgba(0, 180, 42, 0.2)' },
            { offset: 1, color: 'rgba(0, 180, 42, 0)' },
          ],
        },
      },
    },
  ],
  grid: {
    left: 16,
    right: 16,
    top: 16,
    bottom: 16,
    containLabel: true,
  },
})

let metricsTimer: ReturnType<typeof setInterval> | null = null

const formatMemory = (mb: number): string => {
  if (mb >= 1024) {
    return (mb / 1024).toFixed(1) + ' GB'
  }
  return mb + ' MB'
}

const fetchOverview = async () => {
  try {
    const response = await getOverview()
    overview.value = response.data
  } catch (error) {
    console.error('Failed to fetch overview:', error)
  }
}

const fetchMetrics = async () => {
  try {
    const response = await getSystemMetrics()
    const { cpu_percent, memory_used_mb, memory_total_mb, timestamp } = response.data

    const memoryPercent = (memory_used_mb / memory_total_mb) * 100
    const timeStr = new Date(timestamp * 1000).toLocaleTimeString('zh-CN', { hour12: false })

    cpuData.times.push(timeStr)
    cpuData.values.push(Number(cpu_percent.toFixed(1)))
    if (cpuData.times.length > 20) {
      cpuData.times.shift()
      cpuData.values.shift()
    }

    memoryData.times.push(timeStr)
    memoryData.values.push(Number(memoryPercent.toFixed(1)))
    if (memoryData.times.length > 20) {
      memoryData.times.shift()
      memoryData.values.shift()
    }

    cpuChartOption.value = { ...cpuChartOption.value }
    memoryChartOption.value = { ...memoryChartOption.value }
  } catch (error) {
    console.error('Failed to fetch metrics:', error)
  }
}

onMounted(() => {
  fetchOverview()
  fetchMetrics()
  metricsTimer = setInterval(fetchMetrics, 3000)
})

onUnmounted(() => {
  if (metricsTimer) {
    clearInterval(metricsTimer)
  }
})
</script>

<style scoped>
.notice-card {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  margin-bottom: 24px;
}

.notice-card.safe {
  background: #e8ffea;
}

.notice-card.warn {
  background: #fff7e8;
}

.notice-card.danger {
  background: #ffece8;
}

.notice-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.notice-icon {
  font-size: 16px;
  margin-top: 1px;
  flex-shrink: 0;
}

.notice-card.safe .notice-icon {
  color: #00b42a;
}

.notice-card.warn .notice-icon {
  color: #ff7d00;
}

.notice-card.danger .notice-icon {
  color: #f53f3f;
}

.notice-text {
  font-size: 13px;
  color: #1d2129;
}

.notice-card.safe .notice-text {
  color: #009a29;
}

.notice-card.warn .notice-text {
  color: #d25f00;
}

.notice-card.danger .notice-text {
  color: #cb272d;
}

.notice-warning {
  font-size: 12px;
  line-height: 1.5;
}

.notice-card.warn .notice-warning {
  color: #ff7d00;
}

.notice-card.danger .notice-warning {
  color: #f53f3f;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 10px;
  font-size: 24px;
}

.stat-icon.blue {
  background: #e8f3ff;
  color: #165dff;
}

.stat-icon.green {
  background: #e8ffea;
  color: #00b42a;
}

.stat-icon.orange {
  background: #fff7e8;
  color: #ff7d00;
}

.stat-icon.red {
  background: #ffe8e8;
  color: #f53f3f;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #1d2129;
  line-height: 1.2;
}

.stat-value-nowrap {
  white-space: nowrap;
}

.stat-label {
  font-size: 13px;
  color: #4e5969;
  margin-top: 4px;
}

.charts-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.chart-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  padding: 20px;
}

.chart-header {
  margin-bottom: 16px;
}

.chart-title {
  font-size: 16px;
  font-weight: 500;
  color: #1d2129;
  margin: 0 0 4px 0;
}

.chart-subtitle {
  font-size: 12px;
  color: #4e5969;
}

.chart {
  width: 100%;
  height: 280px;
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }

  .stat-card {
    padding: 16px;
    gap: 12px;
  }

  .stat-value,
  .stat-value-nowrap {
    font-size: 20px;
  }

  .charts-grid {
    grid-template-columns: 1fr;
  }

  .chart {
    height: 220px;
  }

  .notice-text,
  .notice-warning {
    font-size: 12px;
  }
}
</style>
