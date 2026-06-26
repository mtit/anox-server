<template>
  <div class="layout">
    <!-- Header -->
    <header class="header">
      <div class="header-content">
        <div class="header-left">
          <div class="logo">
            <img src="/logo.png" alt="Anox" class="logo-img" />
          </div>
          <nav class="nav-menu">
            <a
              v-for="item in menuItems"
              :key="item.key"
              :class="['nav-item', { active: activeMenu === item.key }]"
              @click="$router.push(item.path)"
            >
              <component :is="item.icon" class="nav-icon" />
              <span>{{ item.label }}</span>
            </a>
          </nav>
        </div>
        <div class="header-actions">
          <a-dropdown trigger="click" @select="handleLocaleChange">
            <a-button type="text" class="lang-btn">
              <template #icon><icon-language /></template>
              <span class="lang-text">{{ currentLocaleName }}</span>
            </a-button>
            <template #content>
              <a-doption
                v-for="loc in availableLocales"
                :key="loc.code"
                :value="loc.code"
              >
                {{ loc.name }}
              </a-doption>
            </template>
          </a-dropdown>
          <a-button type="text" class="logout-btn" @click="handleLogout">
            <template #icon>
              <icon-poweroff />
            </template>
            <span class="logout-text">{{ t('common.logout') }}</span>
          </a-button>
        </div>
      </div>
    </header>

    <!-- Mobile Bottom Nav -->
    <nav class="mobile-nav">
      <a
        v-for="item in menuItems"
        :key="item.key"
        :class="['mobile-nav-item', { active: activeMenu === item.key }]"
        @click="$router.push(item.path)"
      >
        <component :is="item.icon" class="mobile-nav-icon" />
        <span>{{ item.label }}</span>
      </a>
    </nav>

    <!-- Main Content -->
    <main class="main">
      <div class="content-wrapper">
        <router-view />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useI18n, loadLocale, type LocaleCode } from '@/i18n'
import {
  IconDashboard,
  IconApps,
  IconSettings,
  IconFile,
  IconPoweroff,
  IconLanguage,
} from '@arco-design/web-vue/es/icon'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { t, locale, messages, availableLocales } = useI18n()

const menuItems = computed(() => {
  void messages.value
  return [
    { key: 'overview', label: t('nav.overview'), path: '/overview', icon: IconDashboard },
    { key: 'services', label: t('nav.services'), path: '/services', icon: IconApps },
    { key: 'configs', label: t('nav.configs'), path: '/configs', icon: IconSettings },
    { key: 'logs', label: t('nav.logs'), path: '/logs', icon: IconFile },
  ]
})

const activeMenu = computed(() => {
  const path = route.path
  if (path === '/overview') return 'overview'
  if (path === '/services') return 'services'
  if (path === '/configs') return 'configs'
  if (path === '/logs') return 'logs'
  return 'overview'
})

const currentLocaleName = computed(() => {
  return availableLocales.value.find(item => item.code === locale.value)?.name || locale.value
})

const handleLocaleChange = async (code: string | number | Record<string, unknown> | undefined) => {
  if (typeof code !== 'string') return
  await loadLocale(code as LocaleCode)
}

const handleLogout = () => {
  authStore.logout()
  router.push('/login')
}
</script>

<style scoped>
.layout {
  min-height: 100vh;
  background: #f7f8fa;
}

.header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 64px;
  background: #fff;
  z-index: 100;
  border-bottom: 1px solid #e5e6eb;
}

.header-content {
  max-width: 1440px;
  margin: 0 auto;
  padding: 0 24px;
  height: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 48px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.logo {
  display: flex;
  align-items: center;
}

.logo-img {
  height: 32px;
  width: auto;
  display: block;
}

.nav-menu {
  display: flex;
  align-items: center;
  gap: 8px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 8px;
  color: #4e5969;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
  text-decoration: none;
}

.nav-item:hover {
  color: #1d2129;
  background: #f2f3f5;
}

.nav-item.active {
  color: #165dff;
  background: #e8f3ff;
  font-weight: 500;
}

.nav-icon {
  font-size: 16px;
}

.lang-btn,
.logout-btn {
  color: #4e5969;
}

.lang-btn:hover,
.logout-btn:hover {
  color: #165dff;
  background: transparent;
}

.logout-btn:hover {
  color: #f53f3f;
}

.main {
  padding-top: 64px;
  min-height: 100vh;
}

.content-wrapper {
  max-width: 1440px;
  margin: 0 auto;
  padding: 24px;
}

.mobile-nav {
  display: none;
}

@media (max-width: 768px) {
  .header-content {
    padding: 0 16px;
  }

  .header-left {
    gap: 12px;
  }

  .nav-menu {
    display: none;
  }

  .lang-text,
  .logout-text {
    display: none;
  }

  .logout-btn {
    color: #f53f3f;
  }

  .logout-btn:hover {
    color: #f53f3f;
  }

  .main {
    padding-top: 56px;
    padding-bottom: calc(56px + env(safe-area-inset-bottom, 0px));
  }

  .content-wrapper {
    padding: 16px;
  }

  .mobile-nav {
    display: flex;
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    height: calc(56px + env(safe-area-inset-bottom, 0px));
    padding-bottom: env(safe-area-inset-bottom, 0px);
    background: #fff;
    border-top: 1px solid #e5e6eb;
    z-index: 100;
  }

  .mobile-nav-item {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    color: #86909c;
    font-size: 11px;
    text-decoration: none;
    cursor: pointer;
  }

  .mobile-nav-item.active {
    color: #165dff;
  }

  .mobile-nav-icon {
    font-size: 20px;
  }
}
</style>
