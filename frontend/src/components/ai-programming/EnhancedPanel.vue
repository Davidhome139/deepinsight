<template>
  <div class="enhanced-programming-panel">
    <!-- Tabs for different features -->
    <el-tabs v-model="activeTab" type="border-card">
      <!-- Codebase Context Tab -->
      <el-tab-pane label="Codebase Context" name="context">
        <div class="context-panel">
          <div class="context-header">
            <el-button type="primary" size="small" @click="loadCodebaseContext" :loading="loadingContext">
              Analyze Codebase
            </el-button>
            <el-button size="small" @click="reindexCodebase" :loading="reindexing">
              Reindex
            </el-button>
          </div>
          
          <div v-if="codebaseContext" class="context-content">
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item label="Project Type">{{ codebaseContext.project_type }}</el-descriptions-item>
              <el-descriptions-item label="Total Files">{{ codebaseContext.total_files }}</el-descriptions-item>
              <el-descriptions-item label="Total Lines">{{ codebaseContext.total_lines?.toLocaleString() }}</el-descriptions-item>
            </el-descriptions>
            
            <el-divider>Relevant Files</el-divider>
            <div class="relevant-files">
              <div v-for="file in codebaseContext.relevant_files" :key="file.path" class="file-item" @click="selectFile(file)">
                <el-icon><Document /></el-icon>
                <span class="file-path">{{ file.relative_path }}</span>
                <el-tag size="small">{{ file.language }}</el-tag>
                <span class="file-lines">{{ file.lines }} lines</span>
              </div>
            </div>
            
            <el-divider v-if="codebaseContext.related_symbols?.length">Related Symbols</el-divider>
            <div class="symbols-list" v-if="codebaseContext.related_symbols?.length">
              <el-tag v-for="symbol in codebaseContext.related_symbols.slice(0, 20)" :key="symbol.name" 
                      :type="symbol.type === 'function' ? 'primary' : 'success'" size="small" class="symbol-tag">
                {{ symbol.name }}
              </el-tag>
            </div>
          </div>
        </div>
      </el-tab-pane>
      
      <!-- Approvals Tab -->
      <el-tab-pane name="approvals">
        <template #label>
          <span>Approvals</span>
          <el-badge v-if="pendingApprovals.length" :value="pendingApprovals.length" class="approval-badge" />
        </template>
        
        <div class="approval-panel">
          <div class="approval-settings">
            <el-switch v-model="approvalRequired" @change="updateApprovalSettings" />
            <span>Require approval before file writes</span>
          </div>
          
          <div v-if="pendingApprovals.length === 0" class="no-approvals">
            <el-empty description="No pending approvals" :image-size="60" />
          </div>
          
          <div v-else class="approval-list">
            <el-card v-for="approval in pendingApprovals" :key="approval.id" class="approval-card" shadow="hover">
              <template #header>
                <div class="approval-header">
                  <span class="approval-type">{{ approval.type }}</span>
                  <span class="approval-file">{{ approval.file_path }}</span>
                </div>
              </template>
              
              <div class="approval-content">
                <p>{{ approval.description }}</p>
                <div v-if="approval.diff" class="diff-preview">
                  <pre>{{ approval.diff.substring(0, 500) }}{{ approval.diff.length > 500 ? '...' : '' }}</pre>
                </div>
              </div>
              
              <div class="approval-actions">
                <el-button type="success" size="small" @click="submitApproval(approval.id, true)">
                  Approve
                </el-button>
                <el-button type="danger" size="small" @click="submitApproval(approval.id, false)">
                  Reject
                </el-button>
                <el-button size="small" @click="viewFullDiff(approval)">
                  View Full Diff
                </el-button>
              </div>
            </el-card>
          </div>
        </div>
      </el-tab-pane>
      
      <!-- Changes Tab -->
      <el-tab-pane label="Proposed Changes" name="changes">
        <div class="changes-panel">
          <div class="changes-header" v-if="changesSummary">
            <el-descriptions :column="3" size="small">
              <el-descriptions-item label="Files">{{ changesSummary.total_files }}</el-descriptions-item>
              <el-descriptions-item label="Added">+{{ changesSummary.total_lines_added }}</el-descriptions-item>
              <el-descriptions-item label="Removed">-{{ changesSummary.total_lines_removed }}</el-descriptions-item>
            </el-descriptions>
            
            <div class="changes-actions">
              <el-button type="primary" size="small" @click="loadChanges">Refresh</el-button>
              <el-button type="success" size="small" @click="approveAllChanges">Approve All</el-button>
              <el-button size="small" @click="viewUnifiedDiff">View Unified Diff</el-button>
            </div>
          </div>
          
          <div class="changes-list">
            <el-collapse v-model="expandedChanges">
              <el-collapse-item v-for="change in proposedChanges" :key="change.file_path" :name="change.file_path">
                <template #title>
                  <div class="change-title">
                    <el-tag :type="getChangeTypeColor(change.change_type)" size="small">
                      {{ change.change_type }}
                    </el-tag>
                    <span class="change-file">{{ change.file_path }}</span>
                    <el-tag :type="getRiskColor(change.metadata?.risk_level)" size="small">
                      {{ change.metadata?.risk_level || 'low' }} risk
                    </el-tag>
                  </div>
                </template>
                
                <div class="change-content">
                  <div v-for="(hunk, idx) in change.hunks" :key="idx" class="diff-hunk">
                    <div class="hunk-header">@@ {{ hunk.old_start }},{{ hunk.old_lines }} {{ hunk.new_start }},{{ hunk.new_lines }} @@</div>
                    <pre class="hunk-content">{{ hunk.content }}</pre>
                  </div>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>
        </div>
      </el-tab-pane>
      
      <!-- Git Operations Tab -->
      <el-tab-pane label="Git" name="git">
        <div class="git-panel">
          <div class="git-status" v-if="gitStatus">
            <el-descriptions title="Repository Status" :column="2" border size="small">
              <el-descriptions-item label="Branch">{{ gitStatus.branch }}</el-descriptions-item>
              <el-descriptions-item label="Status">
                <el-tag :type="gitStatus.is_clean ? 'success' : 'warning'">
                  {{ gitStatus.is_clean ? 'Clean' : 'Modified' }}
                </el-tag>
              </el-descriptions-item>
            </el-descriptions>
            
            <div v-if="gitStatus.staged?.length" class="git-files">
              <h4>Staged ({{ gitStatus.staged.length }})</h4>
              <el-tag v-for="file in gitStatus.staged" :key="file" type="success" size="small" class="git-file">
                {{ file }}
              </el-tag>
            </div>
            
            <div v-if="gitStatus.modified?.length" class="git-files">
              <h4>Modified ({{ gitStatus.modified.length }})</h4>
              <el-tag v-for="file in gitStatus.modified" :key="file" type="warning" size="small" class="git-file">
                {{ file }}
              </el-tag>
            </div>
            
            <div v-if="gitStatus.untracked?.length" class="git-files">
              <h4>Untracked ({{ gitStatus.untracked.length }})</h4>
              <el-tag v-for="file in gitStatus.untracked" :key="file" type="info" size="small" class="git-file">
                {{ file }}
              </el-tag>
            </div>
          </div>
          
          <el-divider>Git Actions</el-divider>
          
          <div class="git-actions">
            <el-button @click="loadGitStatus" :loading="loadingGit">
              <el-icon><Refresh /></el-icon> Refresh Status
            </el-button>
            
            <el-button type="primary" @click="showBranchDialog = true">
              <el-icon><Plus /></el-icon> Create Branch
            </el-button>
            
            <el-button type="success" @click="showCommitDialog = true" :disabled="gitStatus?.is_clean">
              <el-icon><Check /></el-icon> Commit Changes
            </el-button>
            
            <el-button type="warning" @click="preparePR">
              <el-icon><Upload /></el-icon> Prepare PR
            </el-button>
          </div>
        </div>
      </el-tab-pane>
      
      <!-- Syntax Check Tab -->
      <el-tab-pane label="Diagnostics" name="diagnostics">
        <div class="diagnostics-panel">
          <div class="diagnostics-header">
            <el-button type="primary" @click="checkAllSyntax" :loading="checkingSyntax">
              Check All Files
            </el-button>
          </div>
          
          <div v-if="diagnosticsResults && Object.keys(diagnosticsResults).length" class="diagnostics-results">
            <el-collapse>
              <el-collapse-item v-for="(diagnostics, file) in diagnosticsResults" :key="file" :name="file">
                <template #title>
                  <el-icon :color="hasErrors(diagnostics) ? '#f56c6c' : '#e6a23c'"><Warning /></el-icon>
                  <span class="diagnostic-file">{{ file }}</span>
                  <el-badge :value="diagnostics.length" :type="hasErrors(diagnostics) ? 'danger' : 'warning'" />
                </template>
                
                <div class="diagnostics-list">
                  <div v-for="(d, idx) in diagnostics" :key="idx" :class="['diagnostic-item', d.severity]">
                    <span class="line">L{{ d.line }}</span>
                    <span class="message">{{ d.message }}</span>
                    <el-tag :type="d.severity === 'error' ? 'danger' : 'warning'" size="small">
                      {{ d.severity }}
                    </el-tag>
                  </div>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>
          
          <el-empty v-else-if="!checkingSyntax" description="No diagnostics. Click 'Check All Files' to run syntax checks." />
        </div>
      </el-tab-pane>
    </el-tabs>
    
    <!-- Create Branch Dialog -->
    <el-dialog v-model="showBranchDialog" title="Create Task Branch" width="400px">
      <el-form :model="branchForm" label-width="100px">
        <el-form-item label="Task ID">
          <el-input v-model="branchForm.taskId" placeholder="Task identifier" />
        </el-form-item>
        <el-form-item label="Description">
          <el-input v-model="branchForm.description" placeholder="Branch description" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showBranchDialog = false">Cancel</el-button>
        <el-button type="primary" @click="createBranch">Create</el-button>
      </template>
    </el-dialog>
    
    <!-- Commit Dialog -->
    <el-dialog v-model="showCommitDialog" title="Commit Changes" width="500px">
      <el-form :model="commitForm" label-width="100px">
        <el-form-item label="Message">
          <el-input v-model="commitForm.message" type="textarea" :rows="3" placeholder="Commit message" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCommitDialog = false">Cancel</el-button>
        <el-button type="primary" @click="commitChanges">Commit</el-button>
      </template>
    </el-dialog>
    
    <!-- Diff Viewer Dialog -->
    <el-dialog v-model="showDiffDialog" title="Diff Viewer" width="80%" top="5vh">
      <pre class="full-diff">{{ fullDiff }}</pre>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Document, Refresh, Plus, Check, Upload, Warning } from '@element-plus/icons-vue'

// State
const activeTab = ref('context')
const loadingContext = ref(false)
const reindexing = ref(false)
const loadingGit = ref(false)
const checkingSyntax = ref(false)

const codebaseContext = ref<any>(null)
const pendingApprovals = ref<any[]>([])
const proposedChanges = ref<any[]>([])
const changesSummary = ref<any>(null)
const expandedChanges = ref<string[]>([])
const gitStatus = ref<any>(null)
const diagnosticsResults = ref<Record<string, any[]>>({})
const approvalRequired = ref(false)

const showBranchDialog = ref(false)
const showCommitDialog = ref(false)
const showDiffDialog = ref(false)
const fullDiff = ref('')

const branchForm = ref({ taskId: '', description: '' })
const commitForm = ref({ message: '' })

const authToken = localStorage.getItem('token')
const headers = { 'Authorization': `Bearer ${authToken}` }

// API calls
const loadCodebaseContext = async () => {
  loadingContext.value = true
  try {
    const response = await fetch('/api/v1/agent-enhanced/codebase/context?task=current&max_files=15', { headers })
    const data = await response.json()
    codebaseContext.value = data.context
  } catch (error) {
    ElMessage.error('Failed to load codebase context')
  } finally {
    loadingContext.value = false
  }
}

const reindexCodebase = async () => {
  reindexing.value = true
  try {
    await fetch('/api/v1/agent-enhanced/codebase/reindex', { method: 'POST', headers })
    ElMessage.success('Codebase reindexed')
    await loadCodebaseContext()
  } catch (error) {
    ElMessage.error('Failed to reindex codebase')
  } finally {
    reindexing.value = false
  }
}

const loadApprovals = async () => {
  try {
    const response = await fetch('/api/v1/agent-enhanced/approvals', { headers })
    const data = await response.json()
    pendingApprovals.value = data.approvals || []
  } catch (error) {
    console.error('Failed to load approvals:', error)
  }
}

const submitApproval = async (id: string, approved: boolean) => {
  try {
    await fetch('/api/v1/agent-enhanced/approvals/submit', {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ request_id: id, approved })
    })
    ElMessage.success(approved ? 'Approved' : 'Rejected')
    await loadApprovals()
  } catch (error) {
    ElMessage.error('Failed to submit approval')
  }
}

const updateApprovalSettings = async () => {
  try {
    await fetch('/api/v1/agent-enhanced/approvals/settings', {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ required: approvalRequired.value })
    })
  } catch (error) {
    ElMessage.error('Failed to update settings')
  }
}

const loadChanges = async () => {
  try {
    const response = await fetch('/api/v1/agent-enhanced/changes', { headers })
    const data = await response.json()
    proposedChanges.value = data.changes || []
    changesSummary.value = data.summary
  } catch (error) {
    console.error('Failed to load changes:', error)
  }
}

const approveAllChanges = async () => {
  try {
    await fetch('/api/v1/agent-enhanced/changes/approve-all', { method: 'POST', headers })
    ElMessage.success('All changes approved')
    await loadChanges()
  } catch (error) {
    ElMessage.error('Failed to approve changes')
  }
}

const viewUnifiedDiff = async () => {
  try {
    const response = await fetch('/api/v1/agent-enhanced/changes/diff', { headers })
    const data = await response.json()
    fullDiff.value = data.diff
    showDiffDialog.value = true
  } catch (error) {
    ElMessage.error('Failed to load diff')
  }
}

const loadGitStatus = async () => {
  loadingGit.value = true
  try {
    const response = await fetch('/api/v1/agent-enhanced/git/status', { headers })
    const data = await response.json()
    gitStatus.value = data.status
  } catch (error) {
    ElMessage.error('Failed to load Git status')
  } finally {
    loadingGit.value = false
  }
}

const createBranch = async () => {
  try {
    await fetch('/api/v1/agent-enhanced/git/branch', {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify(branchForm.value)
    })
    ElMessage.success('Branch created')
    showBranchDialog.value = false
    await loadGitStatus()
  } catch (error) {
    ElMessage.error('Failed to create branch')
  }
}

const commitChanges = async () => {
  try {
    await fetch('/api/v1/agent-enhanced/git/commit', {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ message: commitForm.value.message, task_id: 'current' })
    })
    ElMessage.success('Changes committed')
    showCommitDialog.value = false
    await loadGitStatus()
  } catch (error) {
    ElMessage.error('Failed to commit changes')
  }
}

const preparePR = async () => {
  try {
    const response = await fetch('/api/v1/agent-enhanced/git/prepare-pr', {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ task_id: 'current', description: 'AI Generated Changes' })
    })
    const data = await response.json()
    ElMessage.success('PR info prepared')
    console.log('PR Info:', data.pull_request)
  } catch (error) {
    ElMessage.error('Failed to prepare PR')
  }
}

const checkAllSyntax = async () => {
  checkingSyntax.value = true
  try {
    const response = await fetch('/api/v1/agent-enhanced/syntax/check-all', { headers })
    const data = await response.json()
    diagnosticsResults.value = data.results || {}
  } catch (error) {
    ElMessage.error('Failed to check syntax')
  } finally {
    checkingSyntax.value = false
  }
}

const selectFile = (file: any) => {
  // Emit event to parent to open file in editor
  console.log('Selected file:', file.path)
}

const viewFullDiff = (approval: any) => {
  fullDiff.value = approval.diff || 'No diff available'
  showDiffDialog.value = true
}

const getChangeTypeColor = (type: string) => {
  const colors: Record<string, string> = { create: 'success', modify: 'warning', delete: 'danger' }
  return colors[type] || 'info'
}

const getRiskColor = (level: string) => {
  const colors: Record<string, string> = { low: 'success', medium: 'warning', high: 'danger' }
  return colors[level] || 'info'
}

const hasErrors = (diagnostics: any[]) => diagnostics.some(d => d.severity === 'error')

onMounted(() => {
  loadApprovals()
  loadGitStatus()
  // Poll for approvals
  setInterval(loadApprovals, 5000)
})
</script>

<style scoped>
.enhanced-programming-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.context-panel, .approval-panel, .changes-panel, .git-panel, .diagnostics-panel {
  padding: 10px;
}

.context-header, .changes-header, .diagnostics-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.relevant-files {
  max-height: 300px;
  overflow-y: auto;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px;
  cursor: pointer;
  border-radius: 4px;
}

.file-item:hover {
  background-color: var(--el-fill-color-light);
}

.file-path {
  flex: 1;
  font-family: monospace;
  font-size: 13px;
}

.file-lines {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.symbols-list {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.symbol-tag {
  margin: 2px;
}

.approval-settings {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 15px;
  padding: 10px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
}

.approval-badge {
  margin-left: 5px;
}

.approval-card {
  margin-bottom: 10px;
}

.approval-header {
  display: flex;
  justify-content: space-between;
}

.approval-type {
  font-weight: bold;
}

.approval-file {
  font-family: monospace;
  color: var(--el-text-color-secondary);
}

.diff-preview {
  background: var(--el-fill-color);
  padding: 10px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  max-height: 200px;
  overflow-y: auto;
}

.approval-actions {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}

.change-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.change-file {
  font-family: monospace;
}

.diff-hunk {
  margin: 10px 0;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}

.hunk-header {
  background: var(--el-fill-color-light);
  padding: 5px 10px;
  font-family: monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.hunk-content {
  padding: 10px;
  font-family: monospace;
  font-size: 12px;
  margin: 0;
  overflow-x: auto;
}

.git-files {
  margin: 15px 0;
}

.git-files h4 {
  margin-bottom: 8px;
  color: var(--el-text-color-secondary);
}

.git-file {
  margin: 2px;
}

.git-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.diagnostic-file {
  margin-left: 10px;
  font-family: monospace;
}

.diagnostics-list {
  padding: 10px;
}

.diagnostic-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 5px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.diagnostic-item.error {
  color: var(--el-color-danger);
}

.diagnostic-item.warning {
  color: var(--el-color-warning);
}

.diagnostic-item .line {
  font-family: monospace;
  color: var(--el-text-color-secondary);
  min-width: 50px;
}

.diagnostic-item .message {
  flex: 1;
  font-size: 13px;
}

.full-diff {
  font-family: monospace;
  font-size: 12px;
  background: var(--el-fill-color);
  padding: 15px;
  border-radius: 4px;
  overflow: auto;
  max-height: 70vh;
  white-space: pre-wrap;
}

.no-approvals {
  padding: 40px 0;
}
</style>
