<template>
  <div class="branch-panel">
    <div class="branch-header">
      <h3>
        <el-icon><Share /></el-icon>
        Conversation Branches
      </h3>
      <div class="branch-actions-header">
        <el-button size="small" @click="showGraphDialog = true" :disabled="branches.length === 0">
          <el-icon><DataLine /></el-icon>
        </el-button>
        <el-button size="small" type="primary" @click="showCreateDialog = true">
          <el-icon><Plus /></el-icon>
          New Branch
        </el-button>
      </div>
    </div>

    <div class="branch-tree" v-if="branches.length > 0">
      <div 
        v-for="branch in branchTree" 
        :key="branch.id"
        class="branch-node"
        :class="{ 
          active: branch.id === activeBranchId,
          main: branch.is_main 
        }"
      >
        <div class="branch-item" @click="switchBranch(branch.id)">
          <div class="branch-info">
            <span class="branch-name">
              <el-icon v-if="branch.is_main"><HomeFilled /></el-icon>
              <el-icon v-else><FolderOpened /></el-icon>
              {{ branch.name }}
            </span>
            <span class="branch-meta">
              {{ branch.message_count }} messages
              <span v-if="branch.score" class="branch-score">
                Score: {{ branch.score.toFixed(1) }}
              </span>
            </span>
          </div>
          <div class="branch-actions" @click.stop>
            <el-dropdown trigger="click" @command="handleBranchAction($event, branch)">
              <el-button size="small" circle>
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="rename">
                    <el-icon><Edit /></el-icon> Rename
                  </el-dropdown-item>
                  <el-dropdown-item command="compare" v-if="!branch.is_main">
                    <el-icon><Switch /></el-icon> Compare with Main
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" v-if="!branch.is_main" divided>
                    <el-icon><Delete /></el-icon> Delete
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>

        <!-- Child branches (indented) -->
        <div v-if="branch.children && branch.children.length" class="branch-children">
          <div 
            v-for="child in branch.children" 
            :key="child.id"
            class="branch-node child"
            :class="{ active: child.id === activeBranchId }"
            @click="switchBranch(child.id)"
          >
            <div class="branch-item">
              <div class="branch-info">
                <span class="branch-name">
                  <el-icon><Document /></el-icon>
                  {{ child.name }}
                </span>
                <span class="branch-meta">{{ child.message_count }} msgs</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="empty-branches">
      <el-empty description="No branches yet" :image-size="60">
        <el-button type="primary" size="small" @click="createMainBranch">
          Initialize Branches
        </el-button>
      </el-empty>
    </div>

    <!-- Create Branch Dialog -->
    <el-dialog v-model="showCreateDialog" title="Create New Branch" width="400px">
      <el-form :model="newBranch" label-width="100px">
        <el-form-item label="Name">
          <el-input v-model="newBranch.name" placeholder="Branch name" />
        </el-form-item>
        <el-form-item label="Description">
          <el-input 
            v-model="newBranch.description" 
            type="textarea" 
            :rows="2"
            placeholder="Optional description" 
          />
        </el-form-item>
        <el-form-item label="Fork From">
          <el-select v-model="newBranch.forkPoint" placeholder="Current position" clearable style="width: 100%">
            <el-option 
              v-for="msg in recentMessages" 
              :key="msg.id" 
              :label="`#${msg.id}: ${msg.content.substring(0, 40)}...`"
              :value="msg.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">Cancel</el-button>
        <el-button type="primary" @click="createBranch">Create</el-button>
      </template>
    </el-dialog>

    <!-- Branch Comparison Dialog -->
    <el-dialog v-model="showCompareDialog" title="Branch Comparison" width="800px">
      <div class="compare-container" v-if="comparingBranch">
        <div class="compare-header">
          <div class="compare-column">
            <el-tag type="primary">Main Branch</el-tag>
          </div>
          <div class="compare-column">
            <el-tag type="success">{{ comparingBranch.name }}</el-tag>
          </div>
        </div>
        <div class="compare-body">
          <div class="compare-column">
            <div v-for="msg in mainBranchMessages" :key="msg.id" class="compare-message" :class="msg.role">
              <div class="msg-role">{{ msg.role }}</div>
              <div class="msg-content">{{ msg.content }}</div>
            </div>
          </div>
          <div class="compare-column">
            <div v-for="msg in compareBranchMessages" :key="msg.id" class="compare-message" :class="msg.role">
              <div class="msg-role">{{ msg.role }}</div>
              <div class="msg-content">{{ msg.content }}</div>
            </div>
          </div>
        </div>
        <div class="compare-footer">
          <div class="compare-stats">
            <span>Main: {{ mainBranchMessages.length }} messages</span>
            <span>{{ comparingBranch.name }}: {{ compareBranchMessages.length }} messages</span>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- Visual Branch Graph -->
    <el-dialog v-model="showGraphDialog" title="Branch Graph" width="600px">
      <div class="branch-graph">
        <svg class="graph-svg" width="100%" height="300">
          <g v-for="(node, i) in graphNodes" :key="i">
            <circle 
              :cx="node.x" 
              :cy="node.y" 
              r="12"
              :fill="node.isMain ? '#409eff' : '#67c23a'"
              @click="switchBranch(node.id)"
              style="cursor: pointer"
            />
            <text 
              :x="node.x" 
              :y="node.y + 28" 
              text-anchor="middle" 
              font-size="12"
            >
              {{ node.name }}
            </text>
            <line 
              v-if="node.parentX" 
              :x1="node.parentX" 
              :y1="node.parentY"
              :x2="node.x" 
              :y2="node.y - 12"
              stroke="#ddd"
              stroke-width="2"
            />
          </g>
        </svg>
      </div>
      <template #footer>
        <el-button @click="showGraphDialog = false">Close</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Share, Plus, HomeFilled, FolderOpened, Document, 
  MoreFilled, Edit, Switch, Delete, DataLine 
} from '@element-plus/icons-vue'

interface Branch {
  id: string
  name: string
  description: string
  is_main: boolean
  is_active: boolean
  message_count: number
  score?: number
  parent_branch_id?: string
  children?: Branch[]
}

const props = defineProps<{
  conversationId: number
  messages: any[]
}>()

const emit = defineEmits(['branch-switched', 'refresh-messages'])

const branches = ref<Branch[]>([])
const activeBranchId = ref<string | null>(null)
const showCreateDialog = ref(false)
const showCompareDialog = ref(false)
const showGraphDialog = ref(false)
const comparingBranch = ref<Branch | null>(null)
const mainBranchMessages = ref<any[]>([])
const compareBranchMessages = ref<any[]>([])
const newBranch = ref({
  name: '',
  description: '',
  forkPoint: null as number | null
})

// Graph nodes for visual branch graph
const graphNodes = computed(() => {
  const nodes: Array<{id: string, name: string, x: number, y: number, isMain: boolean, parentX?: number, parentY?: number}> = []
  const startX = 300
  const startY = 50
  const xGap = 100
  const yGap = 80
  
  let col = 0
  branches.value.forEach((branch, index) => {
    const isMain = branch.is_main
    const x = isMain ? startX : startX + (col % 4 - 2) * xGap
    const y = isMain ? startY : startY + Math.floor(index / 4 + 1) * yGap
    
    let parentX, parentY
    if (branch.parent_branch_id) {
      const parent = nodes.find(n => n.id === branch.parent_branch_id)
      if (parent) {
        parentX = parent.x
        parentY = parent.y + 12
      }
    }
    
    nodes.push({ id: branch.id, name: branch.name, x, y, isMain, parentX, parentY })
    if (!isMain) col++
  })
  
  return nodes
})

const branchTree = computed(() => {
  // Build tree structure from flat list
  const tree: Branch[] = []
  const map = new Map<string, Branch>()
  
  branches.value.forEach(b => {
    map.set(b.id, { ...b, children: [] })
  })
  
  branches.value.forEach(b => {
    const node = map.get(b.id)!
    if (b.parent_branch_id && map.has(b.parent_branch_id)) {
      map.get(b.parent_branch_id)!.children!.push(node)
    } else {
      tree.push(node)
    }
  })
  
  return tree
})

const recentMessages = computed(() => {
  return props.messages.slice(-10).reverse()
})

const fetchBranches = async () => {
  if (!props.conversationId) return
  
  try {
    const response = await fetch(`/api/v1/chat/conversations/${props.conversationId}/branches`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    if (response.ok) {
      branches.value = await response.json()
      // Set active branch
      const active = branches.value.find(b => b.is_active)
      if (active) {
        activeBranchId.value = active.id
      }
    }
  } catch (error) {
    console.error('Failed to fetch branches:', error)
  }
}

const createMainBranch = async () => {
  try {
    const response = await fetch(`/api/v1/chat/conversations/${props.conversationId}/branches`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({
        name: 'Main',
        description: 'Main conversation branch'
      })
    })
    if (response.ok) {
      await fetchBranches()
      ElMessage.success('Branch initialized')
    }
  } catch (error) {
    ElMessage.error('Failed to create branch')
  }
}

const createBranch = async () => {
  if (!newBranch.value.name.trim()) {
    ElMessage.warning('Please enter a branch name')
    return
  }
  
  try {
    const response = await fetch(`/api/v1/chat/conversations/${props.conversationId}/branches`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({
        name: newBranch.value.name,
        description: newBranch.value.description,
        fork_point_message_id: newBranch.value.forkPoint
      })
    })
    if (response.ok) {
      showCreateDialog.value = false
      newBranch.value = { name: '', description: '', forkPoint: null }
      await fetchBranches()
      ElMessage.success('Branch created')
    }
  } catch (error) {
    ElMessage.error('Failed to create branch')
  }
}

const switchBranch = async (branchId: string) => {
  if (branchId === activeBranchId.value) return
  
  try {
    const response = await fetch(
      `/api/v1/chat/conversations/${props.conversationId}/branches/${branchId}/switch`,
      {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      }
    )
    if (response.ok) {
      activeBranchId.value = branchId
      emit('branch-switched', branchId)
      emit('refresh-messages')
      ElMessage.success('Switched to branch')
    }
  } catch (error) {
    ElMessage.error('Failed to switch branch')
  }
}

const handleBranchAction = async (action: string, branch: Branch) => {
  switch (action) {
    case 'rename':
      try {
        const { value } = await ElMessageBox.prompt('Enter new name', 'Rename Branch', {
          inputValue: branch.name
        }) as { value: string }
        if (value) {
          // Implement rename API
          ElMessage.info('Rename feature coming soon')
        }
      } catch {
        // User cancelled
      }
      break
    case 'compare':
      emit('branch-switched', { compare: true, branchId: branch.id })
      break
    case 'delete':
      await ElMessageBox.confirm('Delete this branch? This cannot be undone.', 'Delete Branch')
      try {
        await fetch(
          `/api/v1/chat/conversations/${props.conversationId}/branches/${branch.id}`,
          {
            method: 'DELETE',
            headers: {
              'Authorization': `Bearer ${localStorage.getItem('token')}`
            }
          }
        )
        await fetchBranches()
        ElMessage.success('Branch deleted')
      } catch (error) {
        ElMessage.error('Failed to delete branch')
      }
      break
  }
}

watch(() => props.conversationId, fetchBranches, { immediate: true })

defineExpose({ fetchBranches })
</script>

<style scoped>
.branch-panel {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 12px;
}

.branch-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.branch-header h3 {
  margin: 0;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.branch-tree {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.branch-node {
  border-radius: 6px;
  transition: all 0.2s;
}

.branch-node.active .branch-item {
  background: #e6f4ff;
  border-color: #1890ff;
}

.branch-node.main .branch-item {
  background: #f6ffed;
  border-color: #52c41a;
}

.branch-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  cursor: pointer;
  background: white;
  transition: all 0.2s;
}

.branch-item:hover {
  border-color: #1890ff;
}

.branch-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.branch-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  font-size: 13px;
}

.branch-meta {
  font-size: 11px;
  color: #8c8c8c;
}

.branch-score {
  margin-left: 8px;
  color: #1890ff;
}

.branch-actions {
  opacity: 0;
  transition: opacity 0.2s;
}

.branch-item:hover .branch-actions {
  opacity: 1;
}

.branch-children {
  margin-left: 20px;
  margin-top: 4px;
  padding-left: 12px;
  border-left: 2px solid #e0e0e0;
}

.branch-node.child .branch-item {
  padding: 6px 10px;
}

.branch-node.child .branch-name {
  font-size: 12px;
}

.empty-branches {
  padding: 20px;
  text-align: center;
}

/* Compare Dialog Styles */
.compare-container {
  max-height: 500px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.compare-header {
  display: flex;
  gap: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #ebeef5;
}

.compare-column {
  flex: 1;
}

.compare-body {
  display: flex;
  gap: 16px;
  flex: 1;
  overflow-y: auto;
  padding: 12px 0;
  max-height: 400px;
}

.compare-body .compare-column {
  flex: 1;
  overflow-y: auto;
  max-height: 100%;
}

.compare-message {
  padding: 8px 12px;
  margin-bottom: 8px;
  border-radius: 8px;
  font-size: 13px;
}

.compare-message.user {
  background: #ecf5ff;
}

.compare-message.assistant {
  background: #f5f7fa;
}

.msg-role {
  font-size: 11px;
  font-weight: 600;
  color: #909399;
  margin-bottom: 4px;
  text-transform: uppercase;
}

.msg-content {
  line-height: 1.5;
}

.compare-footer {
  padding-top: 12px;
  border-top: 1px solid #ebeef5;
}

.compare-stats {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: #606266;
}

/* Branch Graph Styles */
.branch-graph {
  background: #f5f7fa;
  border-radius: 8px;
  padding: 16px;
}

.graph-svg {
  display: block;
}
</style>
