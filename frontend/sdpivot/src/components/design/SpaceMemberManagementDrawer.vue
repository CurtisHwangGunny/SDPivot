<template>
  <Teleport to="body">
    <section v-if="open" class="drawer-overlay" aria-label="成员管理抽屉" @click.self="close">
      <aside class="drawer" role="dialog" aria-modal="true" aria-labelledby="space-member-title">
        <header class="drawer-head">
          <div>
            <p class="eyebrow">Space / Members</p>
            <h2 id="space-member-title">成员管理</h2>
            <p class="subtle">为当前知识空间配置成员、角色与文档协作权限。</p>
          </div>
          <button class="close" type="button" aria-label="关闭成员管理" @click="close">×</button>
        </header>
        <div class="drawer-body">
          <div class="notice">
            <div class="notice-mark">i</div>
            <div><strong>权限说明</strong><br>空间 Owner 管理空间；Editor 可导入文档；Viewer 仅可查看与检索。添加成员需要组织管理权限。</div>
          </div>

          <section class="panel">
            <div class="panel-head">
              <h3 class="panel-title">添加成员</h3>
            </div>
            <div class="panel-body">
              <div class="form-grid">
                <label class="field">
                  <span>姓名</span>
                  <input v-model.trim="form.name" class="input" maxlength="100" placeholder="请输入姓名" />
                </label>
                <label class="field">
                  <span>手机号或邮箱</span>
                  <input v-model.trim="form.account" class="input" placeholder="手机号 / 邮箱" />
                </label>
                <label class="field">
                  <span>部门</span>
                  <select v-model="form.departmentId" class="select">
                    <option value="">请选择部门</option>
                    <option v-for="department in departments" :key="department.id" :value="department.id">{{ department.label }}</option>
                  </select>
                </label>
                <label class="field">
                  <span>初始角色</span>
                  <select v-model="form.role" class="select">
                    <option value="editor">Editor · 可导入文档</option>
                    <option value="viewer">Viewer · 只读访问</option>
                  </select>
                </label>
              </div>
              <p v-if="formError" class="form-message error" role="alert">{{ formError }}</p>
              <p v-if="initialPassword" class="form-message success" role="status">成员已添加，初始密码：<strong>{{ initialPassword }}</strong>（仅显示一次）</p>
              <div class="actions" style="margin-top:16px">
                <button class="btn primary" type="button" :disabled="submitting" @click="addMember">{{ submitting ? '添加中...' : '添加成员' }}</button>
              </div>
            </div>
          </section>

          <section class="panel">
            <div class="panel-head">
              <h3 class="panel-title">角色模板</h3>
            </div>
            <div class="panel-body role-grid">
              <article v-for="role in roleTemplatesWithCounts" :key="role.name" class="role-card">
                <h4>{{ role.name }}</h4>
                <p>{{ role.description }}</p>
                <span class="badge" :class="role.variant">{{ role.count }} 人</span>
              </article>
            </div>
          </section>

          <section class="panel">
            <div class="panel-head">
              <h3 class="panel-title">当前成员</h3>
              <div class="segmented">
                <button :class="{ active: memberFilter === 'all' }" @click="memberFilter = 'all'">全部 {{ members.length }}</button>
                <button :class="{ active: memberFilter === 'pending' }" @click="memberFilter = 'pending'">待确认 {{ pendingCount }}</button>
                <button :class="{ active: memberFilter === 'anomaly' }" @click="memberFilter = 'anomaly'">异常 {{ anomalyCount }}</button>
              </div>
            </div>
            <div class="panel-body member-list">
              <div v-for="member in filteredMembers" :key="member.user_id" class="member-row">
                <div class="member-avatar">{{ memberInitial(member) }}</div>
                <div class="member-meta">
                  <strong>{{ memberName(member) }}</strong>
                  <span>{{ memberMeta(member) }}</span>
                </div>
                <span class="badge" :class="roleVariant(member.role)">{{ roleLabel(member.role) }}</span>
                <span class="badge" :class="statusVariant(member)">{{ member.status || '正常' }}</span>
              </div>
            </div>
          </section>
        </div>
        <footer class="drawer-foot">
          <span class="subtle">成员创建与关联将写入审计日志</span>
          <div class="actions">
            <button class="btn primary" type="button" @click="close">完成</button>
          </div>
        </footer>
      </aside>
    </section>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onUnmounted, reactive, ref, watch } from 'vue';
import { createAdminUser, listAdminDepartments, type DepartmentNode } from '@/api/admin';
import { addSpaceMember } from '@/api/spaces';

const props = defineProps<{
  open: boolean;
  spaceId?: string;
  members?: SpaceMember[];
}>();
const emit = defineEmits<{
  'update:open': [value: boolean];
  updated: [];
}>();

interface SpaceMember {
  user_id: string;
  name?: string;
  nickname?: string;
  username?: string;
  email?: string;
  department?: string;
  role?: string;
  status?: string;
  created_at?: string;
}

const memberFilter = ref('all');
const submitting = ref(false);
const formError = ref('');
const initialPassword = ref('');
const departments = ref<Array<{ id: string; label: string }>>([]);
const form = reactive({ name: '', account: '', departmentId: '', role: 'viewer' as 'editor' | 'viewer' });

const roleTemplates = [
  { name: 'Owner', description: '全部空间权限，包含空间删除与成员角色管理。', role: 'owner', variant: 'green' },
  { name: 'Editor', description: '上传文档并参与知识协作。', role: 'editor', variant: 'yellow' },
  { name: 'Viewer', description: '浏览文档并使用知识检索。', role: 'viewer', variant: 'gray' },
];

const members = computed(() => props.members || []);
const pendingCount = computed(() => 0);
const anomalyCount = computed(() => 0);
const roleTemplatesWithCounts = computed(() => roleTemplates.map(role => ({ ...role, count: members.value.filter(member => member.role === role.role).length })));

const filteredMembers = computed(() => {
  if (memberFilter.value === 'all') return members.value;
  return members.value;
});

function memberName(member: SpaceMember) {
  return member.name || member.nickname || member.username || member.email || `成员 ${member.user_id.slice(0, 8)}`;
}
function memberInitial(member: SpaceMember) {
  return memberName(member).trim().charAt(0).toUpperCase() || 'U';
}
function memberMeta(member: SpaceMember) {
  if (member.created_at) {
    const details = [member.email, member.department].filter(Boolean).join(' · ');
    return details || `加入于 ${new Date(member.created_at).toLocaleDateString('zh-CN')}`;
  }
  return '空间成员';
}
function roleLabel(role?: string) {
  const map: Record<string, string> = { owner: 'Owner', admin: 'Admin', editor: 'Editor', viewer: 'Viewer' };
  return map[role?.toLowerCase() || ''] || role || 'Viewer';
}
function roleVariant(role?: string) {
  const map: Record<string, string> = { owner: 'green', admin: 'blue', editor: 'yellow', viewer: 'gray' };
  return map[role?.toLowerCase() || ''] || 'gray';
}
function statusVariant(member: SpaceMember) {
  return member.status === '在线' ? 'green' : 'gray';
}

function flattenDepartments(nodes: DepartmentNode[], prefix = ''): Array<{ id: string; label: string }> {
  return nodes.flatMap(node => [
    { id: node.id, label: `${prefix}${node.name}` },
    ...flattenDepartments(node.children || [], `${prefix}— `),
  ]);
}

async function loadDepartments() {
  try {
    const response = await listAdminDepartments();
    departments.value = flattenDepartments(response.data.data || []);
  } catch (error: unknown) {
    formError.value = errorMessage(error, '部门加载失败，请确认管理权限。');
  }
}

async function addMember() {
  if (!props.spaceId || !form.name || !form.account || !form.departmentId || submitting.value) {
    formError.value = '请完整填写姓名、手机号/邮箱、部门和初始角色。';
    return;
  }
  submitting.value = true;
  formError.value = '';
  initialPassword.value = '';
  const isEmail = form.account.includes('@');
  try {
    const created = await createAdminUser({
      name: form.name,
      email: isEmail ? form.account : undefined,
      phone: isEmail ? undefined : form.account,
      department_id: form.departmentId,
      access_role: form.role === 'editor' ? 'knowledge_editor' : 'knowledge_viewer',
    });
    await addSpaceMember(props.spaceId, { user_id: created.data.user.id, role: form.role });
    initialPassword.value = created.data.initial_password;
    form.name = '';
    form.account = '';
    form.departmentId = '';
    form.role = 'viewer';
    emit('updated');
  } catch (error: unknown) {
    formError.value = errorMessage(error, '添加成员失败，请稍后重试。');
  } finally {
    submitting.value = false;
  }
}
function close() { emit('update:open', false); }

function errorMessage(error: unknown, fallback: string) {
  if (typeof error !== 'object' || !error) return fallback;
  const value = error as { message?: string; response?: { data?: { error?: string; message?: string } } };
  return value.response?.data?.error || value.response?.data?.message || value.message || fallback;
}

watch(() => props.open, (value) => {
  document.body.style.overflow = value ? 'hidden' : '';
  if (value && !departments.value.length) void loadDepartments();
});
onUnmounted(() => { document.body.style.overflow = ''; });
</script>

<style scoped>
.role-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-3); }
.role-card { border: 1px solid var(--ink-200); background: white; border-radius: var(--radius-md); padding: var(--space-4); display: grid; gap: var(--space-2); }
.role-card h4 { margin: 0; font-size: var(--text-sm); }
.role-card p { margin: 0; color: var(--ink-600); font-size: var(--text-xs); }
.member-list { display: grid; gap: var(--space-3); }
.member-row { display: grid; grid-template-columns: 40px 1fr 112px 88px; align-items: center; gap: var(--space-3); padding: var(--space-3); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: white; }
.member-avatar { width: 40px; height: 40px; border-radius: 50%; display: grid; place-items: center; background: var(--brand-100); color: var(--brand-800); font-weight: var(--font-weight-bold); }
.member-meta strong { display: block; font-size: var(--text-sm); }
.member-meta span { color: var(--ink-500); font-size: var(--text-xs); }
.row-action { width: 32px; height: 32px; }
@media (max-width: 760px) { .role-grid, .member-row { grid-template-columns: 1fr; } }
</style>

<style scoped>
.drawer-overlay { position: fixed; inset: 0; z-index: var(--z-modal, 300); display: grid; justify-items: end; background: rgba(15, 23, 19, .42); backdrop-filter: blur(4px); }
.drawer { width: 620px; max-width: 100vw; height: 100%; display: grid; grid-template-rows: auto 1fr auto; background: var(--ink-50); border-left: 1px solid rgba(255,255,255,.32); box-shadow: -32px 0 80px rgba(0,0,0,.20); animation: drawer-in .24s var(--ease-out-expo); }
.drawer--wide { width: 720px; }
.drawer-head { padding: 24px 28px 18px; background: white; border-bottom: 1px solid var(--ink-200); display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-5); }
.drawer-head h2 { margin: var(--space-1) 0 0; font-size: var(--text-2xl); letter-spacing: -.035em; }
.drawer-body { padding: 22px 28px; overflow: auto; display: grid; gap: var(--space-5); }
.drawer-foot { padding: var(--space-4) 28px; border-top: 1px solid var(--ink-200); background: white; display: flex; justify-content: space-between; align-items: center; gap: var(--space-4); }
.eyebrow { margin: 0 0 var(--space-1); color: var(--brand-700); font-size: var(--text-xs); font-weight: var(--font-weight-bold); letter-spacing: .08em; text-transform: uppercase; }
.subtle { margin: var(--space-2) 0 0; color: var(--ink-600); font-size: var(--text-sm); }
.actions { display: flex; align-items: center; gap: var(--space-3); }
.btn { min-height: 36px; border: 1px solid transparent; border-radius: var(--radius-sm); padding: 8px 14px; font-weight: var(--font-weight-semibold); font-size: var(--text-sm); cursor: pointer; }
.btn.primary { background: var(--ink-900); color: white; border-color: var(--ink-900); }
.btn:disabled { opacity: .55; cursor: not-allowed; }
.btn.secondary { background: white; color: var(--ink-700); border-color: var(--ink-200); }
.btn.ghost { background: transparent; color: var(--ink-700); }
.close { width: 36px; height: 36px; border-radius: var(--radius-pill); border: 1px solid var(--ink-200); background: var(--ink-50); color: var(--ink-700); font-weight: var(--font-weight-bold); cursor: pointer; }
.notice { border: 1px solid var(--brand-200); background: var(--brand-50); color: var(--brand-900); border-radius: var(--radius-md); padding: var(--space-4); display: grid; grid-template-columns: 24px 1fr; gap: var(--space-3); font-size: var(--text-sm); }
.notice.warning { border-color: var(--warning-500); background: var(--warning-50); color: var(--ink-800); }
.notice-mark { width: 24px; height: 24px; border-radius: 50%; background: var(--brand-600); color: white; display: grid; place-items: center; font-weight: var(--font-weight-bold); font-size: var(--text-xs); }
.panel { background: white; border: 1px solid var(--ink-200); border-radius: var(--radius-lg); box-shadow: var(--shadow-sm); }
.panel-head { padding: var(--space-5); border-bottom: 1px solid var(--ink-200); display: flex; justify-content: space-between; align-items: center; gap: var(--space-4); }
.panel-title { margin: 0; font-size: var(--text-base); }
.panel-body { padding: var(--space-5); }
.input, .select { width: 100%; min-height: 40px; border: 1px solid var(--ink-200); border-radius: var(--radius-sm); background: white; padding: 9px 12px; color: var(--ink-900); }
.field { display: grid; gap: var(--space-2); }
.field span { font-size: var(--text-xs); color: var(--ink-600); font-weight: var(--font-weight-semibold); }
.form-message { margin: var(--space-3) 0 0; padding: var(--space-3); border-radius: var(--radius-sm); font-size: var(--text-xs); }
.form-message.error { color: #991b1b; background: #fef2f2; }
.form-message.success { color: #166534; background: var(--success-50); }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-4); }
.badge { display: inline-flex; align-items: center; min-height: 24px; padding: 0 9px; border-radius: var(--radius-pill); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.badge.green { background: var(--success-50); color: oklch(0.42 0.12 155); }
.badge.yellow { background: var(--warning-50); color: oklch(0.48 0.14 70); }
.badge.blue { background: var(--info-50); color: oklch(0.43 0.13 230); }
.badge.gray { background: var(--ink-100); color: var(--ink-600); }
.segmented { display: inline-grid; grid-auto-flow: column; gap: var(--space-1); padding: var(--space-1); border-radius: var(--radius-md); background: var(--ink-100); }
.segmented button { border: 0; border-radius: var(--radius-sm); padding: 8px 12px; color: var(--ink-600); background: transparent; font-weight: var(--font-weight-semibold); cursor: pointer; }
.segmented button.active { background: white; color: var(--ink-900); box-shadow: var(--shadow-sm); }
@keyframes drawer-in { from { transform: translateX(36px); opacity: .8; } to { transform: translateX(0); opacity: 1; } }
@media (max-width: 760px) { .drawer, .drawer--wide { width: 100vw; } .form-grid { grid-template-columns: 1fr; } .drawer-foot { flex-direction: column; align-items: stretch; } }
@media (prefers-reduced-motion: reduce) { .drawer { animation-duration: .01ms; } }
</style>
