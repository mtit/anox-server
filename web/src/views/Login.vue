<template>
  <div class="login-page">
    <div class="login-box">
      <div class="login-header">
        <img src="/logo.png" alt="Anox" class="logo-img" />
        <p>{{ t('app.subtitle') }}</p>
      </div>
      <a-form :model="form" layout="vertical" @submit="handleSubmit" class="login-form">
        <a-form-item hide-label>
          <a-input-password
            v-model="form.password"
            :placeholder="t('login.passwordPlaceholder')"
            size="large"
            allow-clear
          >
            <template #prefix>
              <icon-lock />
            </template>
          </a-input-password>
        </a-form-item>
        <a-form-item hide-label>
          <a-button
            type="primary"
            html-type="submit"
            size="large"
            long
            :loading="loading"
          >
            {{ t('login.submit') }}
          </a-button>
        </a-form-item>
      </a-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from '@/i18n'
import { IconLock } from '@arco-design/web-vue/es/icon'

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()

const loading = ref(false)
const form = reactive({
  password: '',
})

const handleSubmit = async () => {
  if (!form.password) {
    Message.warning(t('login.passwordRequired'))
    return
  }

  loading.value = true
  try {
    const success = await authStore.login(form.password)
    if (success) {
      Message.success(t('login.success'))
      router.push('/overview')
    } else {
      Message.error(t('login.wrongPassword'))
    }
  } catch (error) {
    Message.error(t('login.failed'))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
}

.login-box {
  background: #fff;
  padding: 48px 40px;
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  width: 400px;
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo-img {
  height: 32px;
  width: auto;
  display: block;
  margin: 10px auto;
}

.login-header p {
  color: #86909c;
  font-size: 14px;
}

.login-form {
  margin-top: 24px;
}

.login-form :deep(.arco-form-item) {
  margin-bottom: 16px;
}

.login-form :deep(.arco-form-item:last-child) {
  margin-bottom: 0;
}

@media (max-width: 768px) {
  .login-page {
    padding: 16px;
    align-items: flex-start;
    padding-top: 15vh;
  }

  .login-box {
    width: 100%;
    max-width: 400px;
    padding: 32px 24px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  }
}
</style>
