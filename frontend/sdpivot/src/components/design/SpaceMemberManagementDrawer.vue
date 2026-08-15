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
              <div class="segmented" aria-label="成员选择方式">
                <button type="button" :class="{ active: selectionMode === 'people' }" @click="selectionMode = 'people'">按人员</button>
                <button type="button" :class="{ active: selectionMode === 'departments' }" @click="selectionMode = 'departments'">按部门</button>
              </div>
            </div>
            <div class="panel-body member-picker">
              <label v-if="selectionMode === 'people'" class="member-search">
                <span class="sr-only">搜索候选成员</span>
                <input v-model.trim="searchQuery" class="input" type="search" placeholder="搜索姓名、账号或邮箱" />
              </label>
              <div v-if="loadingCandidates" class="picker-empty">正在加载组织架构...</div>
              <div v-else-if="departmentRows.length" class="organization-tree">
                <div v-for="row in departmentRows" :key="row.node.id" class="organization-branch">
                  <div class="department-row" :style="{ paddingLeft: `${12 + row.depth * 20}px` }">
                    <button class="tree-toggle" type="button" @click="toggleDepartment(row.node.id)">
                      {{ row.hasChildren || selectionMode === 'people' ? (expandedDepartments.has(row.node.id) ? '−' : '+') : '·' }}
                    </button>
                    <label>
                      <input v-if="selectionMode === 'departments'" type="checkbox" :checked="selectedDepartmentIds.includes(row.node.id)" :disabled="departmentMeta(row.node.id)?.is_fully_added" @change="toggleSelectedDepartment(row.node.id)" />
                      <strong>{{ row.node.name }}</strong>
                    </label>
                    <span>{{ availableDepartmentCount(row.node.id) }} 人可加入</span>
                  </div>
                  <div v-if="selectionMode === 'people' && expandedDepartments.has(row.node.id)" class="department-users" :style="{ paddingLeft: `${44 + row.depth * 20}px` }">
                    <label v-for="user in usersForDepartment(row.node.id)" :key="user.id" :class="{ disabled: user.is_member }">
                      <input v-model="selectedUserIds" type="checkbox" :value="user.id" :disabled="user.is_member" />
                      <span class="candidate-avatar">{{ user.name.charAt(0).toUpperCase() }}</span>
                      <span><strong>{{ user.name }}</strong><small>{{ user.account || user.email }}{{ user.is_member ? ' · 已是成员' : '' }}</small></span>
                    </label>
                    <p v-if="!usersForDepartment(row.node.id).length">该部门暂无匹配人员</p>
                  </div>
                </div>
              </div>
              <div v-else class="picker-empty">暂无可选择的部门或人员</div>
              <div v-if="selectedItems.length" class="selected-items">
                <div class="selected-items-head"><strong>已选</strong><span>预计添加 {{ selectedPeopleCount }} 人</span></div>
                <div>
                  <button v-for="item in selectedItems" :key="`${item.type}-${item.id}`" type="button" @click="removeSelectedItem(item)">{{ item.label }} <span>×</span></button>
                </div>
              </div>
              <p v-if="formError" class="form-message error" role="alert">{{ formError }}</p>
              <div class="actions" style="margin-top:16px">
                <button class="btn primary" type="button" :disabled="submitting || !selectedPeopleCount" @click="addMembers">{{ submitting ? '添加中...' : `确认，添加 ${selectedPeopleCount} 人` }}</button>
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
                <button class="btn ghost remove-member" type="button" :disabled="removingUserId === member.user_id || member.role === 'owner'" @click="removeMember(member)">{{ removingUserId === member.user_id ? '移除中' : '移除' }}</button>
              </div>
            </div>
          </section>
        </div>
        <footer class="drawer-foot">
          <span class="subtle">成员关联变更将写入审计日志</span>
          <div class="actions">
            <button class="btn primary" type="button" @click="close">完成</button>
          </div>
        </footer>
      </aside>
    </section>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue';
import { listAdminDepartmentTree, type DepartmentNode } from '@/api/admin';
import { addSpaceMember, listSpaceMemberCandidates, removeSpaceMember, type SpaceMemberCandidate, type SpaceMemberCandidateDepartment } from '@/api/spaces';

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
const loadingCandidates = ref(false);
const removingUserId = ref('');
const formError = ref('');
const selectionMode = ref<'people' | 'departments'>('people');
const searchQuery = ref('');
const departmentTree = ref<DepartmentNode[]>([]);
const candidateUsers = ref<SpaceMemberCandidate[]>([]);
const candidateDepartments = ref<SpaceMemberCandidateDepartment[]>([]);
const selectedUserIds = ref<string[]>([]);
const selectedDepartmentIds = ref<string[]>([]);
const expandedDepartments = ref(new Set<string>());

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
const departmentRows = computed(() => {
  const rows: Array<{ node: DepartmentNode; depth: number; hasChildren: boolean }> = [];
  const visit = (nodes: DepartmentNode[], depth: number) => {
    for (const node of nodes) {
      const children = node.children || [];
      rows.push({ node, depth, hasChildren: children.length > 0 });
      if (expandedDepartments.value.has(node.id)) visit(children, depth + 1);
    }
  };
  visit(departmentTree.value, 0);
  return rows;
});
const filteredCandidateUsers = computed(() => {
  const query = searchQuery.value.toLocaleLowerCase();
  return query ? candidateUsers.value.filter(user => [user.name, user.username, user.account, user.email].some(value => value?.toLocaleLowerCase().includes(query))) : candidateUsers.value;
});
const selectedItems = computed(() => [
  ...selectedUserIds.value.map(id => ({ type: 'user' as const, id, label: candidateUsers.value.find(user => user.id === id)?.name || id })),
  ...selectedDepartmentIds.value.map(id => ({ type: 'department' as const, id, label: `${findDepartment(id)?.name || id} · ${availableDepartmentCount(id)} 人` })),
]);
const selectedPeopleCount = computed(() => {
  const ids = new Set(selectedUserIds.value);
  for (const departmentId of selectedDepartmentIds.value) candidateUsers.value.filter(user => user.department_id === departmentId && !user.is_member).forEach(user => ids.add(user.id));
  return ids.size;
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

function buildDepartmentTree(nodes: SpaceMemberCandidateDepartment[]): DepartmentNode[] {
  const byId = new Map(nodes.map(node => [node.id, { id: node.id, name: node.name, parent_id: node.parent_id, children: [] as DepartmentNode[] }]));
  const roots: DepartmentNode[] = [];
  for (const node of byId.values()) {
    const parent = node.parent_id ? byId.get(node.parent_id) : undefined;
    if (parent) parent.children?.push(node); else roots.push(node);
  }
  return roots;
}

async function loadCandidates() {
  if (!props.spaceId) return;
  loadingCandidates.value = true;
  formError.value = '';
  try {
    const [candidateResponse, treeResponse] = await Promise.allSettled([listSpaceMemberCandidates(props.spaceId), listAdminDepartmentTree()]);
    if (candidateResponse.status === 'rejected') throw candidateResponse.reason;
    candidateUsers.value = candidateResponse.value.data.users || [];
    candidateDepartments.value = candidateResponse.value.data.departments || [];
    departmentTree.value = treeResponse.status === 'fulfilled' ? treeResponse.value.data.data || [] : buildDepartmentTree(candidateDepartments.value);
    expandedDepartments.value = new Set(departmentTree.value[0]?.id ? [departmentTree.value[0].id] : []);
  } catch (error: unknown) {
    formError.value = errorMessage(error, '组织架构加载失败，请确认成员管理权限。');
  } finally {
    loadingCandidates.value = false;
  }
}

function departmentMeta(id: string) { return candidateDepartments.value.find(department => department.id === id); }
function availableDepartmentCount(id: string) { const item = departmentMeta(id); return Math.max(0, (item?.member_count || 0) - (item?.existing_member_count || 0)); }
function usersForDepartment(id: string) { return filteredCandidateUsers.value.filter(user => user.department_id === id); }
function findDepartment(id: string): DepartmentNode | undefined {
  const queue = [...departmentTree.value];
  while (queue.length) { const node = queue.shift(); if (node?.id === id) return node; if (node?.children?.length) queue.push(...node.children); }
}
function toggleDepartment(id: string) { const next = new Set(expandedDepartments.value); if (next.has(id)) next.delete(id); else next.add(id); expandedDepartments.value = next; }
function toggleSelectedDepartment(id: string) { selectedDepartmentIds.value = selectedDepartmentIds.value.includes(id) ? selectedDepartmentIds.value.filter(value => value !== id) : [...selectedDepartmentIds.value, id]; }
function removeSelectedItem(item: { type: 'user' | 'department'; id: string }) {
  if (item.type === 'user') selectedUserIds.value = selectedUserIds.value.filter(id => id !== item.id); else selectedDepartmentIds.value = selectedDepartmentIds.value.filter(id => id !== item.id);
}

async function addMembers() {
  if (!props.spaceId || !selectedPeopleCount.value || submitting.value) return;
  submitting.value = true;
  formError.value = '';
  try {
    await addSpaceMember(props.spaceId, { user_ids: selectedUserIds.value, department_ids: selectedDepartmentIds.value });
    selectedUserIds.value = [];
    selectedDepartmentIds.value = [];
    await loadCandidates();
    emit('updated');
  } catch (error: unknown) {
    formError.value = errorMessage(error, '批量添加成员失败，请稍后重试。');
  } finally {
    submitting.value = false;
  }
}
async function removeMember(member: SpaceMember) {
  if (!props.spaceId || member.role === 'owner' || removingUserId.value) return;
  removingUserId.value = member.user_id;
  try { await removeSpaceMember(props.spaceId, member.user_id); await loadCandidates(); emit('updated'); }
  catch (error: unknown) { formError.value = errorMessage(error, '移除成员失败，请稍后重试。'); }
  finally { removingUserId.value = ''; }
}
function close() { emit('update:open', false); }

function errorMessage(error: unknown, fallback: string) {
  if (typeof error !== 'object' || !error) return fallback;
  const value = error as { message?: string; response?: { data?: { error?: string; message?: string } } };
  return value.response?.data?.error || value.response?.data?.message || value.message || fallback;
}

watch(() => props.open, (value) => {
  document.body.style.overflow = value ? 'hidden' : '';
  if (value) void loadCandidates();
});
onUnmounted(() => { document.body.style.overflow = ''; });
</script>

<style scoped>
.role-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-3); }
.role-card { border: 1px solid var(--ink-200); background: white; border-radius: var(--radius-md); padding: var(--space-4); display: grid; gap: var(--space-2); }
.role-card h4 { margin: 0; font-size: var(--text-sm); }
.role-card p { margin: 0; color: var(--ink-600); font-size: var(--text-xs); }
.member-list { display: grid; gap: var(--space-3); }
.member-row { display: grid; grid-template-columns: 40px 1fr 92px 76px 64px; align-items: center; gap: var(--space-3); padding: var(--space-3); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: white; }
.member-avatar { width: 40px; height: 40px; border-radius: 50%; display: grid; place-items: center; background: var(--brand-100); color: var(--brand-800); font-weight: var(--font-weight-bold); }
.member-meta strong { display: block; font-size: var(--text-sm); }
.member-meta span { color: var(--ink-500); font-size: var(--text-xs); }
.row-action { width: 32px; height: 32px; }
.member-picker { display: grid; gap: var(--space-4); }
.member-search { display: block; }
.organization-tree { max-height: 360px; overflow: auto; border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: var(--ink-50); }
.organization-branch + .organization-branch { border-top: 1px solid var(--ink-100); }
.department-row { min-height: 48px; display: grid; grid-template-columns: 28px minmax(0, 1fr) auto; align-items: center; gap: var(--space-2); padding-block: var(--space-2); padding-right: var(--space-3); }
.department-row > label { min-width: 0; display: flex; align-items: center; gap: var(--space-2); cursor: pointer; }
.department-row > label strong { overflow: hidden; font-size: var(--text-sm); text-overflow: ellipsis; white-space: nowrap; }
.department-row > span { color: var(--ink-500); font-size: var(--text-xs); }
.tree-toggle { width: 24px; height: 24px; border: 0; border-radius: var(--radius-xs); color: var(--brand-800); background: var(--brand-50); cursor: pointer; }
.tree-toggle:disabled { color: var(--ink-400); background: transparent; }
.department-users { display: grid; gap: var(--space-1); padding-right: var(--space-3); padding-bottom: var(--space-3); }
.department-users > label { min-height: 48px; display: grid; grid-template-columns: 20px 34px minmax(0, 1fr); align-items: center; gap: var(--space-2); padding: var(--space-2); border-radius: var(--radius-sm); cursor: pointer; }
.department-users > label:hover { background: var(--brand-50); }
.department-users > label.disabled { cursor: not-allowed; opacity: .58; }
.department-users > label > span:last-child { min-width: 0; display: flex; flex-direction: column; }
.department-users strong, .department-users small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.department-users strong { font-size: var(--text-sm); }
.department-users small, .department-users > p { color: var(--ink-500); font-size: var(--text-xs); }
.department-users > p { padding: var(--space-3); }
.candidate-avatar { width: 32px; height: 32px; display: grid; place-items: center; border-radius: 50%; color: var(--brand-900); background: var(--brand-100); font-size: var(--text-xs); font-weight: var(--font-weight-bold); }
.selected-items { padding: var(--space-3); border: 1px solid var(--brand-200); border-radius: var(--radius-md); background: var(--brand-50); }
.selected-items-head { display: flex; justify-content: space-between; color: var(--ink-700); font-size: var(--text-xs); }
.selected-items > div:last-child { display: flex; flex-wrap: wrap; gap: var(--space-2); margin-top: var(--space-3); }
.selected-items button { min-height: 28px; padding: var(--space-1) var(--space-2); border: 1px solid var(--brand-200); border-radius: var(--radius-pill); color: var(--brand-900); background: white; font-size: var(--text-xs); cursor: pointer; }
.selected-items button span { margin-left: var(--space-1); }
.picker-empty { padding: var(--space-6); border: 1px dashed var(--ink-300); border-radius: var(--radius-md); color: var(--ink-500); font-size: var(--text-sm); text-align: center; }
.remove-member { padding-inline: var(--space-2); color: #991b1b; }
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
.input { width: 100%; min-height: 40px; border: 1px solid var(--ink-200); border-radius: var(--radius-sm); background: white; padding: 9px 12px; color: var(--ink-900); }
.form-message { margin: var(--space-3) 0 0; padding: var(--space-3); border-radius: var(--radius-sm); font-size: var(--text-xs); }
.form-message.error { color: #991b1b; background: #fef2f2; }
.badge { display: inline-flex; align-items: center; min-height: 24px; padding: 0 9px; border-radius: var(--radius-pill); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.badge.green { background: var(--success-50); color: oklch(0.42 0.12 155); }
.badge.yellow { background: var(--warning-50); color: oklch(0.48 0.14 70); }
.badge.blue { background: var(--info-50); color: oklch(0.43 0.13 230); }
.badge.gray { background: var(--ink-100); color: var(--ink-600); }
.segmented { display: inline-grid; grid-auto-flow: column; gap: var(--space-1); padding: var(--space-1); border-radius: var(--radius-md); background: var(--ink-100); }
.segmented button { border: 0; border-radius: var(--radius-sm); padding: 8px 12px; color: var(--ink-600); background: transparent; font-weight: var(--font-weight-semibold); cursor: pointer; }
.segmented button.active { background: white; color: var(--ink-900); box-shadow: var(--shadow-sm); }
@keyframes drawer-in { from { transform: translateX(36px); opacity: .8; } to { transform: translateX(0); opacity: 1; } }
@media (max-width: 760px) { .drawer, .drawer--wide { width: 100vw; } .drawer-foot { flex-direction: column; align-items: stretch; } }
@media (prefers-reduced-motion: reduce) { .drawer { animation-duration: .01ms; } }
</style>
