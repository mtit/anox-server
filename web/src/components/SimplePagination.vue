<template>
  <div class="simple-pagination">
    <a-button size="small" :disabled="current <= 1" @click="goPrev">
      {{ t('common.prevPage') }}
    </a-button>
    <span class="page-info">{{ t('common.pageInfo', { current, total: totalPages }) }}</span>
    <a-button size="small" :disabled="current >= totalPages" @click="goNext">
      {{ t('common.nextPage') }}
    </a-button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@/i18n'

const props = defineProps<{
  current: number
  pageSize: number
  total: number
}>()

const emit = defineEmits<{
  'update:current': [value: number]
}>()

const { t } = useI18n()

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))

const goPrev = () => {
  if (props.current > 1) {
    emit('update:current', props.current - 1)
  }
}

const goNext = () => {
  if (props.current < totalPages.value) {
    emit('update:current', props.current + 1)
  }
}
</script>

<style scoped>
.simple-pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  width: 100%;
}

.page-info {
  font-size: 14px;
  color: #4e5969;
  min-width: 64px;
  text-align: center;
}
</style>
