import axios from 'axios'

// Overview
export const getOverview = () => axios.get('/api/overview')
export const getSystemMetrics = () => axios.get('/api/system-metrics')

// Services
export const getServices = () => axios.get('/api/services')
export const getService = (name: string) => axios.get(`/api/services/${name}`)

// Configs
export const getConfigs = () => axios.get('/api/configs')
export const getConfig = (name: string) => axios.get(`/api/configs/${name}`)
export const updateConfig = (name: string, values: Record<string, string>) => 
  axios.put(`/api/configs/${name}`, { values })
export const deleteConfigKey = (name: string, key: string) => 
  axios.delete(`/api/configs/${name}/keys/${key}`)

// Logs
export const getLogServices = () => axios.get('/api/logs/services')
export const getLogInstances = (service: string) => 
  axios.get('/api/logs/instances', { params: { service } })
export const getLogDates = (service: string, instance: string) => 
  axios.get('/api/logs/dates', { params: { service, instance } })
export const getLogHours = (service: string, instance: string, date: string) => 
  axios.get('/api/logs/hours', { params: { service, instance, date } })
export const searchLogs = (params: {
  service?: string
  instance?: string
  date?: string
  hour?: string
  keyword?: string
}) => axios.post('/api/logs/search', params)

// Alerts
export const getAlertConfig = () => axios.get('/api/alerts/config')
export const updateAlertConfig = (config: AlertConfig) => 
  axios.put('/api/alerts/config', config)

// Types
export interface AlertConfig {
  enabled: boolean
  min_level: string
  channels: string[]
  deduplicate: boolean
  deduplicate_window: number

  // Webhook URLs - simplified configuration
  wechat_url?: string
  dingtalk_url?: string
  feishu_url?: string

  // Aliyun SMS configuration
  sms_access_key_id?: string
  sms_access_key_secret?: string
  sms_sign_name?: string
  sms_template_code?: string
  sms_phone_numbers?: string
}

export interface Service {
  name: string
  instance_count: number
  instances?: Instance[]
}

export interface Instance {
  id: string
  service_name: string
  registered_at: string
  last_heartbeat: string
  cpu_cores: number
  cpu_percent: number
  memory_total_mb: number
  memory_avail_mb: number
  global_version: number
  service_version: number
}

export interface Config {
  version: number
  values: Record<string, string>
}

export interface LogEntry {
  time: string
  service: string
  instance: string
  level: string
  action: string
  message: string
  trace_id?: string
  stacks?: string[]
  context?: Record<string, string>
}
