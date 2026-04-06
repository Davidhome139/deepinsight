import { createRouter, createWebHistory } from 'vue-router'

// Lazy load views for code splitting - only ChatView and LoginView are eagerly loaded
// for immediate access, others are loaded on demand
import ChatView from '../views/ChatView.vue'
import LoginView from '../views/LoginView.vue'
import RegisterView from '../views/RegisterView.vue'

// Lazy loaded views - each creates its own chunk
const AIChatView = () => import('../views/AIChatView.vue')
const SettingsView = () => import('../views/SettingsView.vue')
const VideoGenerationView = () => import('../views/VideoGenerationView.vue')
const AIProgrammingView = () => import('../views/AIProgrammingView.vue')
const AnalyticsView = () => import('../views/AnalyticsView.vue')
const KnowledgeBaseView = () => import('../views/KnowledgeBaseView.vue')
const ImageGenerationView = () => import('../views/ImageGenerationView.vue')
const AgentSystemView = () => import('../views/AgentSystemView.vue')

const routes = [
  {
    path: '/',
    name: 'Chat',
    component: ChatView,
    meta: { requiresAuth: true }
  },
  {
    path: '/ai-chat',
    name: 'AIChat',
    component: AIChatView,
    meta: { requiresAuth: true }
  },
  {
    path: '/settings',
    name: 'Settings',
    component: SettingsView,
    meta: { requiresAuth: true }
  },
  {
    path: '/video',
    name: 'VideoGeneration',
    component: VideoGenerationView,
    meta: { requiresAuth: true }
  },
  {
    path: '/programming',
    name: 'AIProgramming',
    component: AIProgrammingView,
    meta: { requiresAuth: true }
  },
  {
    path: '/analytics',
    name: 'Analytics',
    component: AnalyticsView,
    meta: { requiresAuth: true }
  },
  {
    path: '/rag',
    name: 'KnowledgeBase',
    component: KnowledgeBaseView,
    meta: { requiresAuth: true }
  },
  {
    path: '/image',
    name: 'ImageGeneration',
    component: ImageGenerationView,
    meta: { requiresAuth: true }
  },
  {
    path: '/agent-studio',
    name: 'AgentSystem',
    component: AgentSystemView,
    meta: { requiresAuth: true }
  },
  {
    path: '/login',
    name: 'Login',
    component: LoginView
  },
  {
    path: '/register',
    name: 'Register',
    component: RegisterView
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const isAuthenticated = !!localStorage.getItem('token')
  if (to.meta.requiresAuth && !isAuthenticated) {
    next('/login')
  } else {
    next()
  }
})

export default router
