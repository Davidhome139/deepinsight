<template>
  <div class="agent-system-view">
    <!-- Navigation Sidebar -->
    <div class="nav-sidebar">
      <div class="nav-header">
        <h2>🤖 Agent Studio</h2>
      </div>
      
      <el-menu :default-active="activeSection" @select="handleNavSelect">
        <el-menu-item index="agents">
          <el-icon><User /></el-icon>
          <span>My Agents</span>
        </el-menu-item>
        <el-menu-item index="workflows">
          <el-icon><Connection /></el-icon>
          <span>Workflows</span>
        </el-menu-item>
        <el-menu-item index="permissions">
          <el-icon><Lock /></el-icon>
          <span>Permissions</span>
        </el-menu-item>
        <el-menu-item index="marketplace">
          <el-icon><Shop /></el-icon>
          <span>Marketplace</span>
        </el-menu-item>
        <el-menu-item index="ab-tests">
          <el-icon><DataAnalysis /></el-icon>
          <span>A/B Tests</span>
        </el-menu-item>
      </el-menu>
    </div>

    <!-- Main Content Area -->
    <div class="main-content">
      <!-- My Agents Section -->
      <template v-if="activeSection === 'agents'">
        <AgentBuilder 
          v-if="showAgentBuilder" 
          :editing-agent="editingAgent"
          @back="closeAgentBuilder"
          @saved="handleAgentSaved"
        />
        <div v-else class="section-content">
          <div class="section-header">
            <h1>My Agents</h1>
            <el-button type="primary" @click="createNewAgent">
              <el-icon><Plus /></el-icon> Create Agent
            </el-button>
          </div>

          <div class="agents-grid">
            <el-empty v-if="agents.length === 0" description="No agents yet">
              <el-button type="primary" @click="createNewAgent">Create your first agent</el-button>
            </el-empty>

            <div v-for="agent in agents" :key="agent.id" class="agent-card">
              <div class="card-header">
                <span class="agent-icon">{{ agent.icon || '🤖' }}</span>
                <el-dropdown @command="handleAgentCommand($event, agent)">
                  <el-button :icon="MoreFilled" link />
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="edit">Edit</el-dropdown-item>
                      <el-dropdown-item command="duplicate">Duplicate</el-dropdown-item>
                      <el-dropdown-item command="export">Export</el-dropdown-item>
                      <el-dropdown-item command="publish">Publish</el-dropdown-item>
                      <el-dropdown-item command="delete" divided>Delete</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
              <div class="card-body">
                <div class="agent-name">{{ agent.name }}</div>
                <div class="agent-desc">{{ truncate(agent.description, 80) }}</div>
                <div class="agent-meta">
                  <el-tag size="small" effect="plain">{{ agent.cognitive_architecture }}</el-tag>
                  <el-tag :type="agent.status === 'active' ? 'success' : 'info'" size="small">
                    {{ agent.status }}
                  </el-tag>
                </div>
              </div>
              <div class="card-footer">
                <el-button size="small" @click="testAgent(agent)">Test</el-button>
                <el-button type="primary" size="small" @click="editAgent(agent)">Edit</el-button>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- Workflows Section -->
      <template v-else-if="activeSection === 'workflows'">
        <WorkflowDesigner 
          v-if="showWorkflowDesigner"
          :editing-workflow="editingWorkflow"
          @back="closeWorkflowDesigner"
          @saved="handleWorkflowSaved"
        />
        <div v-else class="section-content">
          <div class="section-header">
            <h1>Workflows</h1>
            <el-button type="primary" @click="createNewWorkflow">
              <el-icon><Plus /></el-icon> Create Workflow
            </el-button>
          </div>

          <div class="workflows-grid">
            <el-empty v-if="workflows.length === 0" description="No workflows yet">
              <el-button type="primary" @click="createNewWorkflow">Create your first workflow</el-button>
            </el-empty>

            <div v-for="workflow in workflows" :key="workflow.id" class="workflow-card">
              <div class="card-header">
                <span class="workflow-icon">{{ workflow.icon || '📋' }}</span>
                <el-dropdown @command="handleWorkflowCommand($event, workflow)">
                  <el-button :icon="MoreFilled" link />
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="edit">Edit</el-dropdown-item>
                      <el-dropdown-item command="duplicate">Duplicate</el-dropdown-item>
                      <el-dropdown-item command="export">Export</el-dropdown-item>
                      <el-dropdown-item command="publish">Publish</el-dropdown-item>
                      <el-dropdown-item command="delete" divided>Delete</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
              <div class="card-body">
                <div class="workflow-name">{{ workflow.name }}</div>
                <div class="workflow-desc">{{ truncate(workflow.description, 80) }}</div>
                <div class="workflow-meta">
                  <span>{{ getStepCount(workflow) }} steps</span>
                  <el-tag :type="workflow.status === 'active' ? 'success' : 'info'" size="small">
                    {{ workflow.status }}
                  </el-tag>
                </div>
              </div>
              <div class="card-footer">
                <el-button size="small" @click="runWorkflow(workflow)">Run</el-button>
                <el-button type="primary" size="small" @click="editWorkflow(workflow)">Edit</el-button>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- Permissions Section -->
      <template v-else-if="activeSection === 'permissions'">
        <PermissionManager />
      </template>

      <!-- Marketplace Section -->
      <template v-else-if="activeSection === 'marketplace'">
        <AgentMarketplace />
      </template>

      <!-- A/B Tests Section -->
      <template v-else-if="activeSection === 'ab-tests'">
        <div class="section-content">
          <div class="section-header">
            <h1>A/B Tests</h1>
            <el-button type="primary" @click="showABTestDialog = true">
              <el-icon><Plus /></el-icon> New Test
            </el-button>
          </div>

          <el-empty v-if="abTests.length === 0" description="No A/B tests yet">
            <p>Compare different agent versions to find the best performer</p>
          </el-empty>

          <div class="ab-tests-list">
            <div v-for="test in abTests" :key="test.id" class="ab-test-card">
              <div class="test-header">
                <span class="test-name">{{ test.name }}</span>
                <el-tag :type="getTestStatusType(test.status)" size="small">{{ test.status }}</el-tag>
              </div>
              <div class="test-body">
                <div class="test-variants">
                  <div class="variant">
                    <span class="variant-label">Variant A</span>
                    <span class="variant-id">{{ test.variant_a_id }}</span>
                  </div>
                  <span class="vs">VS</span>
                  <div class="variant">
                    <span class="variant-label">Variant B</span>
                    <span class="variant-id">{{ test.variant_b_id }}</span>
                  </div>
                </div>
                <div v-if="test.winner_id" class="test-winner">
                  Winner: {{ test.winner_id === test.variant_a_id ? 'A' : 'B' }}
                </div>
              </div>
              <div class="test-footer">
                <el-button v-if="test.status === 'draft'" type="primary" size="small" @click="startTest(test)">
                  Start Test
                </el-button>
                <el-button size="small" @click="viewTestDetails(test)">View Details</el-button>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>

    <!-- A/B Test Create Dialog -->
    <el-dialog v-model="showABTestDialog" title="Create A/B Test" width="500px">
      <el-form :model="abTestForm" label-position="top">
        <el-form-item label="Test Name">
          <el-input v-model="abTestForm.name" placeholder="e.g., Agent v1 vs v2" />
        </el-form-item>
        <el-form-item label="Type">
          <el-select v-model="abTestForm.type" style="width: 100%">
            <el-option label="Agent" value="agent" />
            <el-option label="Workflow" value="workflow" />
          </el-select>
        </el-form-item>
        <el-form-item label="Variant A">
          <el-select v-model="abTestForm.variant_a_id" style="width: 100%">
            <el-option 
              v-for="item in (abTestForm.type === 'agent' ? agents : workflows)" 
              :key="item.id" 
              :label="item.name" 
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Variant B">
          <el-select v-model="abTestForm.variant_b_id" style="width: 100%">
            <el-option 
              v-for="item in (abTestForm.type === 'agent' ? agents : workflows)" 
              :key="item.id" 
              :label="item.name" 
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Min Sample Size">
          <el-input-number v-model="abTestForm.min_sample_size" :min="10" :max="1000" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showABTestDialog = false">Cancel</el-button>
        <el-button type="primary" @click="createABTest" :loading="creatingTest">Create</el-button>
      </template>
    </el-dialog>

    <!-- Test Agent Dialog -->
    <el-dialog v-model="showTestDialog" title="Test Agent" width="600px">
      <div class="test-agent-dialog">
        <div class="test-conversation">
          <div v-for="(msg, idx) in testConversation" :key="idx" :class="['test-message', msg.role]">
            {{ msg.content }}
          </div>
        </div>
        <div class="test-input">
          <el-input v-model="testInput" placeholder="Enter test message..." @keyup.enter="sendTestMessage">
            <template #append>
              <el-button @click="sendTestMessage" :loading="testLoading">Send</el-button>
            </template>
          </el-input>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreFilled, User, Connection, Lock, Shop, DataAnalysis } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

import AgentBuilder from '@/components/agent-system/AgentBuilder.vue'
import WorkflowDesigner from '@/components/agent-system/WorkflowDesigner.vue'
import PermissionManager from '@/components/agent-system/PermissionManager.vue'
import AgentMarketplace from '@/components/agent-system/AgentMarketplace.vue'

const authStore = useAuthStore()

const activeSection = ref('agents')
const agents = ref<any[]>([])
const workflows = ref<any[]>([])
const abTests = ref<any[]>([])

const showAgentBuilder = ref(false)
const editingAgent = ref<any>(null)
const showWorkflowDesigner = ref(false)
const editingWorkflow = ref<any>(null)

const showABTestDialog = ref(false)
const creatingTest = ref(false)
const abTestForm = reactive({
  name: '',
  type: 'agent',
  variant_a_id: '',
  variant_b_id: '',
  min_sample_size: 100
})

const showTestDialog = ref(false)
const testingAgent = ref<any>(null)
const testConversation = ref<any[]>([])
const testInput = ref('')
const testLoading = ref(false)

onMounted(async () => {
  await Promise.all([
    loadAgents(),
    loadWorkflows(),
    loadABTests()
  ])
})

const handleNavSelect = (index: string) => {
  activeSection.value = index
  showAgentBuilder.value = false
  showWorkflowDesigner.value = false
}

// Agents
const loadAgents = async () => {
  try {
    const response = await fetch('/api/v1/agent-system/agents', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      agents.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load agents:', error)
  }
}

const createNewAgent = () => {
  editingAgent.value = null
  showAgentBuilder.value = true
}

const editAgent = (agent: any) => {
  editingAgent.value = agent
  showAgentBuilder.value = true
}

const closeAgentBuilder = () => {
  showAgentBuilder.value = false
  editingAgent.value = null
}

const handleAgentSaved = () => {
  closeAgentBuilder()
  loadAgents()
}

const handleAgentCommand = async (command: string, agent: any) => {
  switch (command) {
    case 'edit':
      editAgent(agent)
      break
    case 'duplicate':
      try {
        const response = await fetch(`/api/v1/agent-system/agents/${agent.id}/duplicate`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${authStore.token}`
          },
          body: JSON.stringify({ name: agent.name + ' (Copy)' })
        })
        if (response.ok) {
          ElMessage.success('Agent duplicated')
          loadAgents()
        }
      } catch (e) {
        ElMessage.error('Failed to duplicate')
      }
      break
    case 'export':
      try {
        const response = await fetch(`/api/v1/agent-system/agents/${agent.id}/export`, {
          headers: { 'Authorization': `Bearer ${authStore.token}` }
        })
        if (response.ok) {
          const data = await response.json()
          const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
          const url = URL.createObjectURL(blob)
          const a = document.createElement('a')
          a.href = url
          a.download = `${agent.name}.agent.json`
          a.click()
        }
      } catch (e) {
        ElMessage.error('Failed to export')
      }
      break
    case 'publish':
      ElMessage.info('Publishing coming soon')
      break
    case 'delete':
      try {
        await ElMessageBox.confirm('Delete this agent?', 'Confirm', { type: 'warning' })
        await fetch(`/api/v1/agent-system/agents/${agent.id}`, {
          method: 'DELETE',
          headers: { 'Authorization': `Bearer ${authStore.token}` }
        })
        ElMessage.success('Agent deleted')
        loadAgents()
      } catch (e) {
        // Cancelled
      }
      break
  }
}

const testAgent = (agent: any) => {
  testingAgent.value = agent
  testConversation.value = []
  showTestDialog.value = true
}

const sendTestMessage = async () => {
  if (!testInput.value.trim() || !testingAgent.value) return

  testConversation.value.push({ role: 'user', content: testInput.value })
  const input = testInput.value
  testInput.value = ''
  testLoading.value = true

  try {
    const response = await fetch(`/api/v1/agent-system/agents/${testingAgent.value.id}/execute`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({ input })
    })
    if (response.ok) {
      const result = await response.json()
      testConversation.value.push({ role: 'assistant', content: result.output })
    } else {
      testConversation.value.push({ role: 'assistant', content: 'Error: Failed to get response' })
    }
  } catch (error) {
    testConversation.value.push({ role: 'assistant', content: 'Error: Failed to get response' })
  } finally {
    testLoading.value = false
  }
}

// Workflows
const loadWorkflows = async () => {
  try {
    const response = await fetch('/api/v1/agent-system/workflows', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      workflows.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load workflows:', error)
  }
}

const createNewWorkflow = () => {
  editingWorkflow.value = null
  showWorkflowDesigner.value = true
}

const editWorkflow = (workflow: any) => {
  editingWorkflow.value = workflow
  showWorkflowDesigner.value = true
}

const closeWorkflowDesigner = () => {
  showWorkflowDesigner.value = false
  editingWorkflow.value = null
}

const handleWorkflowSaved = () => {
  closeWorkflowDesigner()
  loadWorkflows()
}

const handleWorkflowCommand = async (command: string, workflow: any) => {
  switch (command) {
    case 'edit':
      editWorkflow(workflow)
      break
    case 'delete':
      try {
        await ElMessageBox.confirm('Delete this workflow?', 'Confirm', { type: 'warning' })
        await fetch(`/api/v1/agent-system/workflows/${workflow.id}`, {
          method: 'DELETE',
          headers: { 'Authorization': `Bearer ${authStore.token}` }
        })
        ElMessage.success('Workflow deleted')
        loadWorkflows()
      } catch (e) {
        // Cancelled
      }
      break
    default:
      ElMessage.info('Feature coming soon')
  }
}

const runWorkflow = (workflow: any) => {
  editingWorkflow.value = workflow
  showWorkflowDesigner.value = true
}

const getStepCount = (workflow: any) => {
  return workflow.steps?.length || 0
}

// A/B Tests
const loadABTests = async () => {
  try {
    const response = await fetch('/api/v1/agent-system/ab-tests', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      abTests.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load A/B tests:', error)
  }
}

const createABTest = async () => {
  if (!abTestForm.name || !abTestForm.variant_a_id || !abTestForm.variant_b_id) {
    ElMessage.error('Please fill all fields')
    return
  }

  creatingTest.value = true
  try {
    const response = await fetch('/api/v1/agent-system/ab-tests', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify(abTestForm)
    })
    if (response.ok) {
      ElMessage.success('A/B test created')
      showABTestDialog.value = false
      loadABTests()
    }
  } catch (error) {
    ElMessage.error('Failed to create test')
  } finally {
    creatingTest.value = false
  }
}

const startTest = async (test: any) => {
  try {
    await fetch(`/api/v1/agent-system/ab-tests/${test.id}/start`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    ElMessage.success('Test started')
    loadABTests()
  } catch (error) {
    ElMessage.error('Failed to start test')
  }
}

const viewTestDetails = (test: any) => {
  ElMessage.info('Test details coming soon')
}

const getTestStatusType = (status: string) => {
  const types: Record<string, string> = {
    draft: 'info',
    running: 'warning',
    completed: 'success',
    cancelled: 'danger'
  }
  return types[status] || 'info'
}

const truncate = (str: string, len: number) => {
  if (!str) return ''
  return str.length > len ? str.substring(0, len) + '...' : str
}
</script>

<style scoped>
.agent-system-view {
  display: flex;
  height: 100vh;
  background: var(--el-bg-color-page);
}

.nav-sidebar {
  width: 220px;
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color);
  display: flex;
  flex-direction: column;
}

.nav-header {
  padding: 20px;
  border-bottom: 1px solid var(--el-border-color);
}

.nav-header h2 {
  margin: 0;
  font-size: 18px;
}

.nav-sidebar .el-menu {
  border-right: none;
  flex: 1;
}

.main-content {
  flex: 1;
  overflow-y: auto;
}

.section-content {
  padding: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.section-header h1 {
  margin: 0;
}

.agents-grid,
.workflows-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.agent-card,
.workflow-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 12px;
  overflow: hidden;
  transition: all 0.2s;
}

.agent-card:hover,
.workflow-card:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--el-fill-color-light);
}

.agent-icon,
.workflow-icon {
  font-size: 28px;
}

.card-body {
  padding: 16px;
}

.agent-name,
.workflow-name {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 8px;
}

.agent-desc,
.workflow-desc {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-bottom: 12px;
  line-height: 1.4;
}

.agent-meta,
.workflow-meta {
  display: flex;
  gap: 8px;
  align-items: center;
}

.card-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

/* A/B Tests */
.ab-tests-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.ab-test-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 12px;
  overflow: hidden;
}

.test-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--el-fill-color-light);
}

.test-name {
  font-weight: 600;
}

.test-body {
  padding: 16px;
}

.test-variants {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 24px;
}

.variant {
  text-align: center;
}

.variant-label {
  display: block;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.variant-id {
  font-family: monospace;
  font-size: 13px;
}

.vs {
  font-weight: 600;
  color: var(--el-color-primary);
}

.test-winner {
  text-align: center;
  margin-top: 12px;
  font-weight: 500;
  color: var(--el-color-success);
}

.test-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

/* Test Dialog */
.test-agent-dialog {
  display: flex;
  flex-direction: column;
  height: 400px;
}

.test-conversation {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  margin-bottom: 16px;
}

.test-message {
  padding: 8px 12px;
  border-radius: 8px;
  margin-bottom: 8px;
  max-width: 80%;
}

.test-message.user {
  background: var(--el-color-primary-light-9);
  margin-left: auto;
}

.test-message.assistant {
  background: var(--el-bg-color);
}
</style>
