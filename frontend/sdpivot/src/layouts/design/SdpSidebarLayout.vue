<template>
  <div class="sdp-sidebar-layout">
    <header class="sdp-sidebar-layout__mobile-header">
      <RouterLink class="sdp-sidebar-layout__mobile-brand" to="/spaces" aria-label="前往 SDPivot 首页">
        <span class="sdp-sidebar-layout__brand-mark" aria-hidden="true">
          <svg viewBox="0 0 32 32" role="img">
            <path d="M16 2 29 9.5v13L16 30 3 22.5v-13L16 2Z" fill="currentColor" />
            <path d="m16 9 7 4v6l-7 4-7-4v-6l7-4Z" fill="var(--ink-950)" />
          </svg>
        </span>
        <span>SDPivot</span>
      </RouterLink>

      <button
        class="sdp-sidebar-layout__menu-button"
        type="button"
        aria-label="打开导航菜单"
        aria-controls="sdp-sidebar-navigation"
        :aria-expanded="isMenuOpen"
        @click="isMenuOpen = true"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
    </header>

    <button
      v-if="isMenuOpen"
      class="sdp-sidebar-layout__backdrop"
      type="button"
      aria-label="关闭导航菜单"
      @click="closeMenu"
    />

    <aside
      id="sdp-sidebar-navigation"
      ref="sidebarRef"
      class="sdp-sidebar-layout__sidebar"
      :class="{ 'sdp-sidebar-layout__sidebar--open': isMenuOpen }"
      aria-label="SDPivot 侧边栏"
      @keydown.esc="closeMenu"
    >
      <div class="sdp-sidebar-layout__top">
        <div class="sdp-sidebar-layout__brand-row">
          <RouterLink class="sdp-sidebar-layout__brand" to="/spaces" aria-label="前往 SDPivot 首页" @click="closeMenu">
            <span class="sdp-sidebar-layout__brand-mark" aria-hidden="true">
              <svg viewBox="0 0 32 32" role="img">
                <path d="M16 2 29 9.5v13L16 30 3 22.5v-13L16 2Z" fill="currentColor" />
                <path d="m16 9 7 4v6l-7 4-7-4v-6l7-4Z" fill="var(--ink-950)" />
              </svg>
            </span>
            <span class="sdp-sidebar-layout__brand-copy">
              <strong>SDPivot</strong>
              <span>{{ mode === 'admin' ? '管理中心' : '知识工作台' }}</span>
            </span>
          </RouterLink>

          <button class="sdp-sidebar-layout__close-button" type="button" aria-label="关闭导航菜单" @click="closeMenu">
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="m6 6 12 12M18 6 6 18" />
            </svg>
          </button>
        </div>

        <nav
          class="sdp-sidebar-layout__nav"
          role="navigation"
          :aria-label="mode === 'admin' ? '管理导航' : '用户导航'"
          @keydown="handleNavKeydown"
        >
          <p class="sdp-sidebar-layout__nav-label">{{ mode === 'admin' ? '系统管理' : '工作空间' }}</p>
          <RouterLink
            v-for="item in navigationItems"
            :key="item.path"
            class="sdp-sidebar-layout__nav-item"
            :class="{ 'sdp-sidebar-layout__nav-item--active': isItemActive(item) }"
            :to="item.path"
            :aria-label="item.label"
            :aria-current="isItemActive(item) ? 'page' : undefined"
            @click="closeMenu"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path :d="item.icon" />
            </svg>
            <span>{{ item.label }}</span>
          </RouterLink>
        </nav>
      </div>

      <div class="sdp-sidebar-layout__user-card" aria-label="当前用户信息">
        <span class="sdp-sidebar-layout__avatar" aria-hidden="true">{{ userInitial }}</span>
        <span class="sdp-sidebar-layout__user-copy">
          <strong>{{ userName }}</strong>
          <span>{{ mode === 'admin' || user?.is_system_admin || user?.is_ops_admin ? '管理员' : '成员' }}</span>
        </span>
      </div>
    </aside>

    <main class="sdp-sidebar-layout__content">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

type SidebarMode = 'user' | 'admin'

interface NavigationItem {
  label: string
  path: string
  icon: string
  exact?: boolean
}

const props = withDefaults(defineProps<{
  mode?: SidebarMode
}>(), {
  mode: 'user',
})

const userNavigation: NavigationItem[] = [
  { label: '知识空间', path: '/spaces', icon: 'M3 6.75A1.75 1.75 0 0 1 4.75 5h5l2 2h7.5A1.75 1.75 0 0 1 21 8.75v8.5A1.75 1.75 0 0 1 19.25 19H4.75A1.75 1.75 0 0 1 3 17.25V6.75Z' },
  { label: 'AI 问答', path: '/qa', icon: 'M5 4h14a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-7l-5 3v-3H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2Z' },
  { label: 'AI 写作', path: '/writing', icon: 'm4 20 4.5-1 10-10a2.12 2.12 0 0 0-3-3l-10 10L4 20Zm10-12 3 3M4 20h16' },
]

const adminNavigation: NavigationItem[] = [
  { label: '仪表盘', path: '/admin', icon: 'M4 13h6V4H4v9Zm10 7h6v-9h-6v9ZM4 20h6v-3H4v3Zm10-13h6V4h-6v3Z', exact: true },
  { label: '人员管理', path: '/admin/people', icon: 'M16 20v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2m7-10a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm13 10v-2a4 4 0 0 0-3-3.87m-2-11.96a4 4 0 0 1 0 7.75' },
  { label: '系统配置', path: '/admin/config', icon: 'M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Zm7.4-3.5a7.8 7.8 0 0 0-.1-1l2-1.6-2-3.4-2.5 1a8.1 8.1 0 0 0-1.7-1L14.7 3h-4L10.3 6a8.1 8.1 0 0 0-1.7 1L6.1 6 4 9.4 6 11a7.8 7.8 0 0 0 0 2l-2 1.6L6.1 18l2.5-1a8.1 8.1 0 0 0 1.7 1l.4 3h4l.4-3a8.1 8.1 0 0 0 1.7-1l2.5 1 2-3.4-2-1.6a7.8 7.8 0 0 0 .1-1Z' },
  { label: '安全设置', path: '/admin/security', icon: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Zm-3-10 2 2 4-5' },
]

const route = useRoute()
const authStore = useAuthStore()
const isMenuOpen = ref(false)
const sidebarRef = ref<HTMLElement | null>(null)

const navigationItems = computed(() => props.mode === 'admin' ? adminNavigation : userNavigation)
const user = computed(() => authStore.user)
const userName = computed(() => authStore.user?.nickname || authStore.user?.username || '当前用户')
const userInitial = computed(() => String(userName.value).trim().charAt(0).toUpperCase() || 'U')

function isItemActive(item: NavigationItem) {
  return item.exact ? route.path === item.path : route.path === item.path || route.path.startsWith(`${item.path}/`)
}

function closeMenu() {
  isMenuOpen.value = false
}

function handleNavKeydown(event: KeyboardEvent) {
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return

  const links = Array.from(sidebarRef.value?.querySelectorAll<HTMLElement>('.sdp-sidebar-layout__nav-item') ?? [])
  if (links.length === 0) return

  event.preventDefault()
  const currentIndex = links.indexOf(document.activeElement as HTMLElement)
  let nextIndex = currentIndex

  if (event.key === 'Home') nextIndex = 0
  if (event.key === 'End') nextIndex = links.length - 1
  if (event.key === 'ArrowDown') nextIndex = currentIndex < links.length - 1 ? currentIndex + 1 : 0
  if (event.key === 'ArrowUp') nextIndex = currentIndex > 0 ? currentIndex - 1 : links.length - 1

  links[nextIndex]?.focus()
}

watch(isMenuOpen, async (isOpen) => {
  if (!isOpen) return
  await nextTick()
  sidebarRef.value?.querySelector<HTMLElement>('.sdp-sidebar-layout__nav-item')?.focus()
})

watch(() => route.fullPath, closeMenu)
</script>

<style scoped>
.sdp-sidebar-layout {
  min-height: 100dvh;
  display: grid;
  grid-template-columns: var(--sidebar-width) minmax(var(--space-0), 1fr);
  color: var(--ink-900);
  background: var(--ink-50);
}

.sdp-sidebar-layout__sidebar {
  position: sticky;
  top: var(--space-0);
  z-index: var(--z-sticky);
  width: var(--sidebar-width);
  height: 100dvh;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: var(--space-5) var(--space-3) var(--space-4);
  overflow-y: auto;
  color: var(--ink-100);
  background: var(--ink-950);
}

.sdp-sidebar-layout__top {
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
}

.sdp-sidebar-layout__brand-row,
.sdp-sidebar-layout__brand,
.sdp-sidebar-layout__mobile-brand,
.sdp-sidebar-layout__user-card,
.sdp-sidebar-layout__nav-item {
  display: flex;
  align-items: center;
}

.sdp-sidebar-layout__brand-row {
  justify-content: space-between;
}

.sdp-sidebar-layout__brand,
.sdp-sidebar-layout__mobile-brand {
  gap: var(--space-3);
  color: var(--ink-50);
  text-decoration: none;
}

.sdp-sidebar-layout__brand {
  min-width: var(--space-0);
  padding: var(--space-2);
  border-radius: var(--radius-sm);
}

.sdp-sidebar-layout__brand-mark {
  width: var(--space-8);
  height: var(--space-8);
  flex: 0 0 var(--space-8);
  color: var(--brand-500);
}

.sdp-sidebar-layout__brand-mark svg,
.sdp-sidebar-layout__nav-item svg,
.sdp-sidebar-layout__menu-button svg,
.sdp-sidebar-layout__close-button svg {
  width: 100%;
  height: 100%;
}

.sdp-sidebar-layout__brand-copy,
.sdp-sidebar-layout__user-copy {
  min-width: var(--space-0);
  display: flex;
  flex-direction: column;
}

.sdp-sidebar-layout__brand-copy strong {
  font-family: var(--font-display);
  font-size: var(--text-lg);
  line-height: var(--leading-tight);
}

.sdp-sidebar-layout__brand-copy span,
.sdp-sidebar-layout__user-copy span {
  color: var(--ink-400);
  font-size: var(--text-xs);
}

.sdp-sidebar-layout__nav {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.sdp-sidebar-layout__nav-label {
  padding: var(--space-0) var(--space-3);
  color: var(--ink-400);
  font-family: var(--font-body);
  font-size: var(--text-xs);
  font-weight: var(--font-weight-semibold);
  letter-spacing: 0.08em;
}

.sdp-sidebar-layout__nav-item {
  min-height: var(--space-10);
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  color: var(--ink-200);
  font-family: var(--font-body);
  font-size: var(--text-sm);
  font-weight: var(--font-weight-medium);
  text-decoration: none;
  transition:
    color var(--duration-fast) var(--ease-out-quart),
    background-color var(--duration-fast) var(--ease-out-quart);
}

.sdp-sidebar-layout__nav-item svg {
  width: var(--space-5);
  height: var(--space-5);
  flex: 0 0 var(--space-5);
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.8;
}

.sdp-sidebar-layout__nav-item:hover {
  color: var(--ink-50);
  background: var(--ink-800);
}

.sdp-sidebar-layout__nav-item--active,
.sdp-sidebar-layout__nav-item--active:hover {
  color: var(--ink-950);
  background: var(--brand-500);
}

.sdp-sidebar-layout__brand:focus-visible,
.sdp-sidebar-layout__mobile-brand:focus-visible,
.sdp-sidebar-layout__nav-item:focus-visible,
.sdp-sidebar-layout__menu-button:focus-visible,
.sdp-sidebar-layout__close-button:focus-visible {
  outline: 2px solid var(--brand-500);
  outline-offset: 2px;
}

.sdp-sidebar-layout__user-card {
  gap: var(--space-3);
  padding: var(--space-3);
  border: 1px solid var(--ink-800);
  border-radius: var(--radius-md);
  background: var(--ink-900);
}

.sdp-sidebar-layout__avatar {
  width: var(--space-8);
  height: var(--space-8);
  flex: 0 0 var(--space-8);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-pill);
  color: var(--ink-950);
  background: var(--brand-500);
  font-family: var(--font-body);
  font-size: var(--text-sm);
  font-weight: var(--font-weight-bold);
}

.sdp-sidebar-layout__user-copy strong {
  overflow: hidden;
  color: var(--ink-50);
  font-family: var(--font-body);
  font-size: var(--text-sm);
  font-weight: var(--font-weight-semibold);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sdp-sidebar-layout__content {
  min-width: var(--space-0);
  min-height: 100dvh;
}

.sdp-sidebar-layout__mobile-header,
.sdp-sidebar-layout__backdrop,
.sdp-sidebar-layout__close-button {
  display: none;
}

@media (max-width: 639px) {
  .sdp-sidebar-layout {
    display: block;
    padding-top: var(--header-height);
  }

  .sdp-sidebar-layout__mobile-header {
    position: fixed;
    inset: var(--space-0) var(--space-0) auto var(--space-0);
    z-index: var(--z-sticky);
    height: var(--header-height);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-2) var(--space-4);
    color: var(--ink-50);
    background: var(--ink-950);
  }

  .sdp-sidebar-layout__mobile-brand {
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: var(--font-weight-bold);
  }

  .sdp-sidebar-layout__menu-button,
  .sdp-sidebar-layout__close-button {
    width: var(--space-10);
    height: var(--space-10);
    align-items: center;
    justify-content: center;
    padding: var(--space-2);
    border: 1px solid var(--ink-700);
    border-radius: var(--radius-sm);
    color: var(--ink-50);
    background: var(--ink-900);
    cursor: pointer;
  }

  .sdp-sidebar-layout__menu-button {
    display: inline-flex;
  }

  .sdp-sidebar-layout__close-button {
    display: inline-flex;
    flex: 0 0 var(--space-10);
  }

  .sdp-sidebar-layout__menu-button svg,
  .sdp-sidebar-layout__close-button svg {
    fill: none;
    stroke: currentColor;
    stroke-linecap: round;
    stroke-width: 2;
  }

  .sdp-sidebar-layout__backdrop {
    position: fixed;
    inset: var(--space-0);
    z-index: var(--z-sticky);
    display: block;
    width: 100%;
    height: 100%;
    padding: var(--space-0);
    border: 0;
    background: color-mix(in oklch, var(--ink-950) 72%, transparent);
    cursor: pointer;
  }

  .sdp-sidebar-layout__sidebar {
    position: fixed;
    inset: var(--space-0) auto var(--space-0) var(--space-0);
    z-index: calc(var(--z-sticky) + 1);
    transform: translateX(-100%);
    transition: transform var(--duration-normal) var(--ease-out-expo);
  }

  .sdp-sidebar-layout__sidebar--open {
    transform: translateX(var(--space-0));
  }
}
</style>
