<template>
  <div class="permission-manager">
    <!-- Header -->
    <div class="manager-header">
      <h2>Tool Permissions</h2>
      <el-button type="primary" @click="showCreateDialog">Create Permission</el-button>
    </div>

    <!-- Stats Cards -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-value">{{ stats.totalInvocations }}</div>
        <div class="stat-label">Total Invocations</div>
      </div>
      <div class="stat-card success">
        <div class="stat-value">{{ stats.allowedCount }}</div>
        <div class="stat-label">Allowed</div>
      </div>
      <div class="stat-card danger">
        <div class="stat-value">{{ stats.deniedCount }}</div>
        <div class="stat-label">Denied</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{ permissions.length }}</div>
        <div class="stat-label">Active Rules</div>
      </div>
    </div>

    <!-- Tabs -->
    <el-tabs v-model="activeTab">
      <!-- Permission Rules Tab -->
      <el-tab-pane label="Permission Rules" name="rules">
        <div class="permissions-list">
          <el-empty v-if="permissions.length === 0" description="No permission rules configured">
            <el-button type="primary" @click="createDefaults">Create Default Rules</el-button>
          </el-empty>

          <div v-for="perm in permissions" :key="perm.id" class="permission-card">
            <div class="perm-header">
              <div class="perm-title">
                <el-switch v-model="perm.is_enabled" @change="togglePermission(perm)" />
                <span class="perm-name">{{ perm.name }}</span>
                <el-tag v-if="perm.requires_approval" type="warning" size="small">Approval Required</el-tag>
              </div>
              <el-dropdown @command="handlePermCommand($event, perm)">
                <el-button :icon="MoreFilled" link />
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="edit">Edit</el-dropdown-item>
                    <el-dropdown-item command="duplicate">Duplicate</el-dropdown-item>
                    <el-dropdown-item command="delete" divided>Delete</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
            <div class="perm-body">
              <div class="perm-pattern">
                <span class="label">Pattern:</span>
                <code>{{ perm.tool_pattern }}</code>
              </div>
              <div class="perm-meta">
                <span v-if="perm.scope !== 'global'" class="perm-scope">
                  <el-tag size="small">{{ perm.scope }}</el-tag>
                </span>
                <span v-if="getRateLimit(perm)" class="perm-rate">
                  Rate: {{ getRateLimit(perm) }}/min
                </span>
                <span class="perm-audit">
                  Audit: {{ perm.audit_level }}
                </span>
              </div>
              <div v-if="perm.description" class="perm-desc">{{ perm.description }}</div>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- Invocation Logs Tab -->
      <el-tab-pane label="Activity Logs" name="logs">
        <div class="logs-filter">
          <el-select v-model="logFilter.status" placeholder="Status" clearable style="width: 120px">
            <el-option label="Allowed" value="allowed" />
            <el-option label="Denied" value="denied" />
          </el-select>
          <el-input v-model="logFilter.toolName" placeholder="Tool name" style="width: 200px" />
          <el-button @click="loadLogs">Search</el-button>
        </div>

        <el-table :data="logs" style="width: 100%" max-height="500">
          <el-table-column prop="tool_name" label="Tool" width="200" />
          <el-table-column prop="status" label="Status" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'allowed' ? 'success' : 'danger'" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="denial_reason" label="Reason" />
          <el-table-column prop="latency_ms" label="Latency" width="100">
            <template #default="{ row }">
              {{ row.latency_ms }}ms
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="Time" width="180">
            <template #default="{ row }">
              {{ formatTime(row.created_at) }}
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- Usage Analytics Tab -->
      <el-tab-pane label="Analytics" name="analytics">
        <div class="analytics-section">
          <h4>Top Tools (Last 30 Days)</h4>
          <div class="top-tools">
            <div v-for="(count, tool) in stats.topTools" :key="tool" class="tool-bar">
              <span class="tool-name">{{ tool }}</span>
              <div class="bar-container">
                <div class="bar" :style="{ width: getBarWidth(count) }"></div>
              </div>
              <span class="tool-count">{{ count }}</span>
            </div>
          </div>
        </div>

        <div class="analytics-section">
          <h4>Daily Activity</h4>
          <div class="daily-chart">
            <div v-for="(count, date) in stats.dailyBreakdown" :key="date" class="day-bar">
              <div class="day-fill" :style="{ height: getDayHeight(count) }"></div>
              <span class="day-label">{{ formatDate(date) }}</span>
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- Create/Edit Permission Dialog -->
    <el-dialog v-model="showDialog" :title="editingPerm ? 'Edit Permission' : 'Create Permission'" width="600px">
      <el-form :model="permForm" label-position="top">
        <el-form-item label="Name" required>
          <el-input v-model="permForm.name" placeholder="Permission name" />
        </el-form-item>

        <el-form-item label="Description">
          <el-input v-model="permForm.description" type="textarea" :rows="2" />
        </el-form-item>

        <el-form-item label="Tool Pattern" required>
          <el-input v-model="permForm.tool_pattern" placeholder="e.g., file/*, mcp://github/*">
            <template #prepend>Pattern</template>
          </el-input>
          <div class="form-hint">Use * for wildcards. Examples: file/read*, mcp://*/*</div>
        </el-form-item>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="Scope">
              <el-select v-model="permForm.scope" style="width: 100%">
                <el-option label="Global" value="global" />
                <el-option label="Agent-specific" value="agent" />
                <el-option label="Workflow-specific" value="workflow" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Audit Level">
              <el-select v-model="permForm.audit_level" style="width: 100%">
                <el-option label="None" value="none" />
                <el-option label="Summary" value="summary" />
                <el-option label="Detailed" value="detailed" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider>Rate Limiting</el-divider>

        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="Max per Minute">
              <el-input-number v-model="permForm.rateLimit.maxPerMinute" :min="0" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Max per Hour">
              <el-input-number v-model="permForm.rateLimit.maxPerHour" :min="0" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Max per Day">
              <el-input-number v-model="permForm.rateLimit.maxPerDay" :min="0" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider>Security</el-divider>

        <el-form-item label="Blocked Patterns">
          <el-select 
            v-model="permForm.blockedPatterns" 
            multiple 
            filterable 
            allow-create
            placeholder="Add patterns to block"
            style="width: 100%"
          >
            <el-option label="rm -rf" value="rm -rf" />
            <el-option label="sudo" value="sudo" />
            <el-option label="> /dev" value="> /dev" />
            <el-option label="chmod 777" value="chmod 777" />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-checkbox v-model="permForm.requires_approval">Require human approval</el-checkbox>
        </el-form-item>

        <el-form-item>
          <el-checkbox v-model="permForm.is_enabled">Enabled</el-checkbox>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="showDialog = false">Cancel</el-button>
        <el-button type="primary" @click="savePermission" :loading="saving">Save</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { MoreFilled } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

const activeTab = ref('rules')
const permissions = ref<any[]>([])
const logs = ref<any[]>([])
const stats = ref({
  totalInvocations: 0,
  allowedCount: 0,
  deniedCount: 0,
  topTools: {} as Record<string, number>,
  dailyBreakdown: {} as Record<string, number>
})

const showDialog = ref(false)
const editingPerm = ref<any>(null)
const saving = ref(false)

const logFilter = reactive({
  status: '',
  toolName: ''
})

const permForm = reactive({
  name: '',
  description: '',
  tool_pattern: '',
  scope: 'global',
  audit_level: 'summary',
  requires_approval: false,
  is_enabled: true,
  rateLimit: {
    maxPerMinute: 0,
    maxPerHour: 0,
    maxPerDay: 0
  },
  blockedPatterns: [] as string[]
})

onMounted(async () => {
  await Promise.all([
    loadPermissions(),
    loadStats(),
    loadLogs()
  ])
})

const loadPermissions = async () => {
  try {
    const response = await fetch('/api/v1/agent-system/permissions', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      permissions.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load permissions:', error)
  }
}

const loadStats = async () => {
  try {
    const response = await fetch('/api/v1/agent-system/permissions/stats?days=30', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      stats.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

const loadLogs = async () => {
  try {
    let url = '/api/v1/agent-system/permissions/logs?limit=100'
    if (logFilter.status) url += `&status=${logFilter.status}`
    if (logFilter.toolName) url += `&tool_name=${logFilter.toolName}`

    const response = await fetch(url, {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      logs.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load logs:', error)
  }
}

const showCreateDialog = () => {
  editingPerm.value = null
  Object.assign(permForm, {
    name: '',
    description: '',
    tool_pattern: '',
    scope: 'global',
    audit_level: 'summary',
    requires_approval: false,
    is_enabled: true,
    rateLimit: { maxPerMinute: 0, maxPerHour: 0, maxPerDay: 0 },
    blockedPatterns: []
  })
  showDialog.value = true
}

const savePermission = async () => {
  if (!permForm.name.trim() || !permForm.tool_pattern.trim()) {
    ElMessage.error('Name and tool pattern are required')
    return
  }

  saving.value = true
  try {
    const url = editingPerm.value 
      ? `/api/v1/agent-system/permissions/${editingPerm.value.id}`
      : '/api/v1/agent-system/permissions'
    
    const method = editingPerm.value ? 'PUT' : 'POST'
    
    const response = await fetch(url, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({
        name: permForm.name,
        description: permForm.description,
        tool_pattern: permForm.tool_pattern,
        scope: permForm.scope,
        audit_level: permForm.audit_level,
        requires_approval: permForm.requires_approval,
        is_enabled: permForm.is_enabled,
        rate_limit: JSON.stringify(permForm.rateLimit),
        blocked_args: JSON.stringify(permForm.blockedPatterns)
      })
    })

    if (response.ok) {
      ElMessage.success(editingPerm.value ? 'Permission updated' : 'Permission created')
      showDialog.value = false
      await loadPermissions()
    } else {
      const error = await response.json()
      ElMessage.error(error.error || 'Failed to save permission')
    }
  } catch (error) {
    ElMessage.error('Failed to save permission')
  } finally {
    saving.value = false
  }
}

const togglePermission = async (perm: any) => {
  try {
    await fetch(`/api/v1/agent-system/permissions/${perm.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({ is_enabled: perm.is_enabled })
    })
  } catch (error) {
    ElMessage.error('Failed to update permission')
    perm.is_enabled = !perm.is_enabled
  }
}

const handlePermCommand = async (command: string, perm: any) => {
  switch (command) {
    case 'edit':
      editingPerm.value = perm
      Object.assign(permForm, {
        name: perm.name,
        description: perm.description,
        tool_pattern: perm.tool_pattern,
        scope: perm.scope,
        audit_level: perm.audit_level,
        requires_approval: perm.requires_approval,
        is_enabled: perm.is_enabled,
        rateLimit: perm.rate_limit ? JSON.parse(perm.rate_limit) : { maxPerMinute: 0, maxPerHour: 0, maxPerDay: 0 },
        blockedPatterns: perm.blocked_args ? JSON.parse(perm.blocked_args) : []
      })
      showDialog.value = true
      break

    case 'duplicate':
      editingPerm.value = null
      Object.assign(permForm, {
        name: perm.name + ' (Copy)',
        description: perm.description,
        tool_pattern: perm.tool_pattern,
        scope: perm.scope,
        audit_level: perm.audit_level,
        requires_approval: perm.requires_approval,
        is_enabled: true,
        rateLimit: perm.rate_limit ? JSON.parse(perm.rate_limit) : { maxPerMinute: 0, maxPerHour: 0, maxPerDay: 0 },
        blockedPatterns: perm.blocked_args ? JSON.parse(perm.blocked_args) : []
      })
      showDialog.value = true
      break

    case 'delete':
      try {
        await ElMessageBox.confirm('Delete this permission rule?', 'Confirm', { type: 'warning' })
        await fetch(`/api/v1/agent-system/permissions/${perm.id}`, {
          method: 'DELETE',
          headers: { 'Authorization': `Bearer ${authStore.token}` }
        })
        ElMessage.success('Permission deleted')
        await loadPermissions()
      } catch (e) {
        // Cancelled
      }
      break
  }
}

const createDefaults = async () => {
  try {
    const response = await fetch('/api/v1/agent-system/permissions/defaults', {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      ElMessage.success('Default permissions created')
      await loadPermissions()
    }
  } catch (error) {
    ElMessage.error('Failed to create defaults')
  }
}

const getRateLimit = (perm: any) => {
  if (!perm.rate_limit) return null
  try {
    const rl = JSON.parse(perm.rate_limit)
    return rl.maxPerMinute || null
  } catch (e) {
    return null
  }
}

const formatTime = (dateStr: string) => {
  return new Date(dateStr).toLocaleString()
}

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}/${date.getDate()}`
}

const getBarWidth = (count: number) => {
  const maxCount = Math.max(...Object.values(stats.value.topTools), 1)
  return `${(count / maxCount) * 100}%`
}

const getDayHeight = (count: number) => {
  const maxCount = Math.max(...Object.values(stats.value.dailyBreakdown), 1)
  return `${(count / maxCount) * 100}%`
}
</script>

<style scoped>
.permission-manager {
  padding: 24px;
}

.manager-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.manager-header h2 {
  margin: 0;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  text-align: center;
}

.stat-card.success .stat-value {
  color: var(--el-color-success);
}

.stat-card.danger .stat-value {
  color: var(--el-color-danger);
}

.stat-value {
  font-size: 32px;
  font-weight: 600;
  margin-bottom: 4px;
}

.stat-label {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.permissions-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.permission-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  overflow: hidden;
}

.perm-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--el-fill-color-light);
}

.perm-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.perm-name {
  font-weight: 500;
}

.perm-body {
  padding: 12px 16px;
}

.perm-pattern {
  margin-bottom: 8px;
}

.perm-pattern .label {
  color: var(--el-text-color-secondary);
  margin-right: 8px;
}

.perm-pattern code {
  background: var(--el-fill-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
}

.perm-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.perm-desc {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.logs-filter {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.analytics-section {
  margin-bottom: 32px;
}

.analytics-section h4 {
  margin: 0 0 16px 0;
  color: var(--el-text-color-secondary);
}

.top-tools {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tool-bar {
  display: flex;
  align-items: center;
  gap: 12px;
}

.tool-bar .tool-name {
  width: 150px;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.bar-container {
  flex: 1;
  height: 20px;
  background: var(--el-fill-color);
  border-radius: 4px;
  overflow: hidden;
}

.bar {
  height: 100%;
  background: var(--el-color-primary);
  border-radius: 4px;
  transition: width 0.3s;
}

.tool-count {
  width: 50px;
  text-align: right;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.daily-chart {
  display: flex;
  gap: 4px;
  height: 150px;
  align-items: flex-end;
}

.day-bar {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
}

.day-fill {
  width: 100%;
  background: var(--el-color-primary);
  border-radius: 2px 2px 0 0;
  transition: height 0.3s;
}

.day-label {
  font-size: 10px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.form-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}
</style>
