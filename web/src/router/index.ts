import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      name: 'Layout',
      component: () => import('@/views/Layout.vue'),
      redirect: '/overview',
      children: [
        {
          path: '/overview',
          name: 'Overview',
          component: () => import('@/views/Overview.vue'),
        },
        {
          path: '/services',
          name: 'Services',
          component: () => import('@/views/Services.vue'),
        },
        {
          path: '/configs',
          name: 'Configs',
          component: () => import('@/views/Configs.vue'),
        },
        {
          path: '/logs',
          name: 'Logs',
          component: () => import('@/views/Logs.vue'),
        },
      ],
    },
  ],
})

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()
  
  if (to.meta?.public) {
    next()
    return
  }
  
  if (!authStore.isAuthenticated) {
    next('/login')
    return
  }
  
  next()
})

export default router
