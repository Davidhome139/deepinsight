<template>
  <div class="workflow-designer">
    <!-- Header -->
    <div class="designer-header">
      <div class="header-left">
        <el-button :icon="ArrowLeft" text @click="$emit('back')">Back</el-button>
        <el-input 
          v-model="workflow.name" 
          placeholder="Workflow name"
          style="width: 200px"
        />
      </div>
      <div class="header-center">
        <el-button-group>
          <el-button :type="mode === 'edit' ? 'primary' : ''" @click="mode = 'edit'">Edit</el-button>
          <el-button :type="mode === 'run' ? 'primary' : ''" @click="mode = 'run'">Run</el-button>
        </el-button-group>
      </div>
      <div class="header-right">
        <el-button-group class="zoom-controls">
          <el-button size="small" @click="zoomOut">
            <el-icon><ZoomOut /></el-icon>
          </el-button>
          <el-button size="small" disabled>{{ Math.round(zoom * 100) }}%</el-button>
          <el-button size="small" @click="zoomIn">
            <el-icon><ZoomIn /></el-icon>
          </el-button>
          <el-button size="small" @click="resetZoom" title="Reset Zoom">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </el-button-group>
        <el-button @click="autoLayout" title="Auto Layout">
          <el-icon><Grid /></el-icon> Auto Layout
        </el-button>
        <el-button @click="handleExport">Export</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">Save</el-button>
      </div>
    </div>

    <!-- Main Content -->
    <div class="designer-content">
      <!-- Left Sidebar - Step Types -->
      <div class="step-palette">
        <h4>Add Steps</h4>
        <div 
          v-for="stepType in stepTypes" 
          :key="stepType.type"
          class="step-type-item"
          draggable="true"
          @dragstart="handleDragStart(stepType)"
        >
          <div class="step-icon">{{ stepType.icon }}</div>
          <div class="step-info">
            <div class="step-name">{{ stepType.name }}</div>
            <div class="step-desc">{{ stepType.description }}</div>
          </div>
        </div>

        <el-divider>My Agents</el-divider>
        <div 
          v-for="agent in myAgents" 
          :key="agent.id"
          class="step-type-item agent-item"
          draggable="true"
          @dragstart="handleAgentDragStart(agent)"
        >
          <div class="step-icon">{{ agent.icon || '🤖' }}</div>
          <div class="step-info">
            <div class="step-name">{{ agent.name }}</div>
          </div>
        </div>
      </div>

      <!-- Canvas -->
      <div 
        class="workflow-canvas"
        @dragover.prevent
        @drop="handleDrop"
        ref="canvasRef"
      >
        <div class="canvas-inner" :style="{ transform: `scale(${zoom})`, transformOrigin: 'top left' }">
        <svg class="connections-layer" ref="svgRef">
          <defs>
            <marker 
              id="arrowhead" 
              markerWidth="10" 
              markerHeight="7" 
              refX="9" 
              refY="3.5" 
              orient="auto"
            >
              <polygon points="0 0, 10 3.5, 0 7" fill="var(--el-color-primary)" />
            </marker>
          </defs>
          <path 
            v-for="edge in workflow.edges" 
            :key="edge.id"
            :d="getEdgePath(edge)"
            stroke="var(--el-color-primary)"
            stroke-width="2"
            fill="none"
            marker-end="url(#arrowhead)"
            class="edge-path"
            @click="selectEdge(edge)"
            :class="{ selected: selectedEdge?.id === edge.id }"
          />
          <!-- Drawing edge -->
          <path 
            v-if="drawingEdge"
            :d="drawingEdgePath"
            stroke="var(--el-color-primary)"
            stroke-width="2"
            stroke-dasharray="5,5"
            fill="none"
          />
        </svg>

        <!-- Steps -->
        <div 
          v-for="step in workflow.steps" 
          :key="step.id"
          class="workflow-step"
          :class="{ selected: selectedStep?.id === step.id, running: runningSteps.has(step.id), completed: completedSteps.has(step.id) }"
          :style="{ left: step.position_x + 'px', top: step.position_y + 'px' }"
          @mousedown="handleStepMouseDown($event, step)"
          @click.stop="selectStep(step)"
        >
          <div class="step-header">
            <span class="step-type-icon">{{ getStepIcon(step.type) }}</span>
            <span class="step-title">{{ step.name || step.type }}</span>
            <el-dropdown trigger="click" @command="handleStepCommand($event, step)">
              <el-button :icon="MoreFilled" link size="small" />
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">Edit</el-dropdown-item>
                  <el-dropdown-item command="duplicate">Duplicate</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>Delete</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
          <div class="step-body">
            <template v-if="step.type === 'agent'">
              <span>{{ getAgentName(step.agent_id) }}</span>
            </template>
            <template v-else-if="step.type === 'condition'">
              <span class="condition-label">If/Else</span>
            </template>
            <template v-else-if="step.type === 'checkpoint'">
              <span class="checkpoint-label">⏸️ Human Review</span>
            </template>
            <template v-else>
              <span>{{ step.type }}</span>
            </template>
          </div>
          <!-- Connection Points -->
          <div 
            class="connection-point output"
            @mousedown.stop="startDrawingEdge($event, step, 'output')"
          />
          <div 
            class="connection-point input"
            @mouseup="finishDrawingEdge(step)"
          />
        </div>

        <!-- Empty State -->
        <div v-if="workflow.steps.length === 0" class="empty-canvas">
          <div class="empty-icon">📋</div>
          <div class="empty-text">Drag steps here to build your workflow</div>
        </div>
        </div><!-- close canvas-inner -->
      </div>

      <!-- Right Sidebar - Properties -->
      <div class="properties-panel">
        <template v-if="selectedStep">
          <h4>Step Properties</h4>
          <el-form :model="selectedStep" label-position="top" size="small">
            <el-form-item label="Name">
              <el-input v-model="selectedStep.name" />
            </el-form-item>

            <template v-if="selectedStep.type === 'agent'">
              <el-form-item label="Agent">
                <el-select v-model="selectedStep.agent_id" style="width: 100%">
                  <el-option 
                    v-for="agent in myAgents" 
                    :key="agent.id" 
                    :label="agent.name" 
                    :value="agent.id"
                  />
                </el-select>
              </el-form-item>
            </template>

            <template v-if="selectedStep.type === 'condition'">
              <el-form-item label="Field">
                <el-input v-model="selectedStep.condition.field" placeholder="e.g., output.status" />
              </el-form-item>
              <el-form-item label="Operator">
                <el-select v-model="selectedStep.condition.operator" style="width: 100%">
                  <el-option label="Equals (==)" value="eq" />
                  <el-option label="Not Equals (!=)" value="ne" />
                  <el-option label="Greater Than (>)" value="gt" />
                  <el-option label="Less Than (<)" value="lt" />
                  <el-option label="Contains" value="contains" />
                </el-select>
              </el-form-item>
              <el-form-item label="Value">
                <el-input v-model="selectedStep.condition.value" />
              </el-form-item>
            </template>

            <el-form-item label="Timeout (seconds)">
              <el-input-number v-model="selectedStep.timeout_seconds" :min="10" :max="3600" />
            </el-form-item>
          </el-form>
        </template>

        <template v-else-if="selectedEdge">
          <h4>Edge Properties</h4>
          <el-form :model="selectedEdge" label-position="top" size="small">
            <el-form-item label="Label">
              <el-select v-model="selectedEdge.label" style="width: 100%">
                <el-option label="Success (default)" value="success" />
                <el-option label="True" value="true" />
                <el-option label="False" value="false" />
                <el-option label="Error" value="error" />
              </el-select>
            </el-form-item>
            <el-button type="danger" size="small" @click="deleteEdge">Delete Edge</el-button>
          </el-form>
        </template>

        <template v-else>
          <h4>Workflow Properties</h4>
          <el-form :model="workflow" label-position="top" size="small">
            <el-form-item label="Description">
              <el-input v-model="workflow.description" type="textarea" :rows="3" />
            </el-form-item>
            <el-form-item label="Tags">
              <el-select v-model="workflow.tags" multiple filterable allow-create style="width: 100%">
                <el-option v-for="tag in suggestedTags" :key="tag" :label="tag" :value="tag" />
              </el-select>
            </el-form-item>
          </el-form>
        </template>

        <!-- Run Panel (when in run mode) -->
        <template v-if="mode === 'run'">
          <el-divider />
          <h4>Run Workflow</h4>
          <el-form label-position="top" size="small">
            <el-form-item label="Input">
              <el-input v-model="runInput" type="textarea" :rows="3" placeholder='{"key": "value"}' />
            </el-form-item>
            <el-button 
              type="primary" 
              @click="handleRun" 
              :loading="running"
              :disabled="workflow.steps.length === 0"
            >
              Start Workflow
            </el-button>
          </el-form>

          <!-- Run Status -->
          <div v-if="currentRun" class="run-status">
            <div class="status-header">
              <span :class="['status-badge', currentRun.status]">{{ currentRun.status }}</span>
            </div>
            <div class="run-events">
              <div v-for="event in runEvents" :key="event.timestamp" class="run-event">
                <span class="event-type">{{ event.type }}</span>
                <span class="event-step" v-if="event.step_name">{{ event.step_name }}</span>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowLeft, MoreFilled, ZoomIn, ZoomOut, Refresh, Grid } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  editingWorkflow?: any
}>()

const emit = defineEmits(['back', 'saved'])
const authStore = useAuthStore()

const mode = ref<'edit' | 'run'>('edit')
const saving = ref(false)
const running = ref(false)
const runInput = ref('{}')
const currentRun = ref<any>(null)
const runEvents = ref<any[]>([])
const runningSteps = ref(new Set<string>())
const completedSteps = ref(new Set<string>())

const canvasRef = ref<HTMLElement | null>(null)
const svgRef = ref<SVGElement | null>(null)
const selectedStep = ref<any>(null)
const selectedEdge = ref<any>(null)
const myAgents = ref<any[]>([])

const drawingEdge = ref(false)
const drawingEdgeStart = ref<any>(null)
const drawingEdgeEnd = ref({ x: 0, y: 0 })

// Zoom state
const zoom = ref(1)
const zoomIn = () => {
  zoom.value = Math.min(zoom.value + 0.1, 2)
}
const zoomOut = () => {
  zoom.value = Math.max(zoom.value - 0.1, 0.5)
}
const resetZoom = () => {
  zoom.value = 1
}

// Auto layout function
const autoLayout = () => {
  if (workflow.steps.length === 0) return
  
  const startX = 50
  const startY = 50
  const stepWidth = 200
  const stepHeight = 80
  const horizontalGap = 100
  const verticalGap = 60
  
  // Simple left-to-right layout
  workflow.steps.forEach((step: any, index: number) => {
    const row = Math.floor(index / 3)
    const col = index % 3
    step.position_x = startX + col * (stepWidth + horizontalGap)
    step.position_y = startY + row * (stepHeight + verticalGap)
  })
  
  ElMessage.success('Layout applied')
}

const suggestedTags = ['automation', 'analysis', 'data-processing', 'content', 'integration']

const stepTypes = [
  { type: 'agent', name: 'Agent', description: 'Run a custom agent', icon: '🤖' },
  { type: 'condition', name: 'Condition', description: 'If/else branching', icon: '🔀' },
  { type: 'transform', name: 'Transform', description: 'Transform data', icon: '🔄' },
  { type: 'checkpoint', name: 'Checkpoint', description: 'Human approval', icon: '⏸️' },
  { type: 'tool', name: 'Tool', description: 'Run MCP tool', icon: '🔧' }
]

const workflow = reactive({
  id: '',
  name: 'New Workflow',
  description: '',
  icon: '📋',
  status: 'draft',
  tags: [] as string[],
  steps: [] as any[],
  edges: [] as any[]
})

let stepIdCounter = 0
let edgeIdCounter = 0

onMounted(async () => {
  await loadAgents()
  
  if (props.editingWorkflow) {
    Object.assign(workflow, props.editingWorkflow)
    if (workflow.steps) {
      stepIdCounter = workflow.steps.length
    }
    if (workflow.edges) {
      edgeIdCounter = workflow.edges.length
    }
  }

  // Mouse move listener for edge drawing
  document.addEventListener('mousemove', handleMouseMove)
  document.addEventListener('mouseup', handleMouseUp)
})

const loadAgents = async () => {
  try {
    const response = await fetch('/api/v1/agent-system/agents', {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      myAgents.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load agents:', error)
  }
}

const handleDragStart = (stepType: any) => {
  if (canvasRef.value) {
    (canvasRef.value as any).__dragData = { type: 'stepType', data: stepType }
  }
}

const handleAgentDragStart = (agent: any) => {
  if (canvasRef.value) {
    (canvasRef.value as any).__dragData = { type: 'agent', data: agent }
  }
}

const handleDrop = (event: DragEvent) => {
  const dragData = (canvasRef.value as any)?.__dragData
  if (!dragData) return

  const rect = canvasRef.value!.getBoundingClientRect()
  const x = event.clientX - rect.left
  const y = event.clientY - rect.top

  if (dragData.type === 'stepType') {
    addStep(dragData.data.type, x, y)
  } else if (dragData.type === 'agent') {
    addStep('agent', x, y, { agent_id: dragData.data.id })
  }

  delete (canvasRef.value as any).__dragData
}

const addStep = (type: string, x: number, y: number, extra: any = {}) => {
  const step = {
    id: `step_${++stepIdCounter}`,
    name: `${type.charAt(0).toUpperCase() + type.slice(1)} ${stepIdCounter}`,
    type,
    position_x: x - 80,
    position_y: y - 30,
    timeout_seconds: 300,
    condition: type === 'condition' ? { field: '', operator: 'eq', value: '' } : undefined,
    ...extra
  }
  workflow.steps.push(step)
  selectedStep.value = step
  selectedEdge.value = null
}

const handleStepMouseDown = (event: MouseEvent, step: any) => {
  if (event.button !== 0) return // Only left click
  
  const startX = event.clientX
  const startY = event.clientY
  const startPosX = step.position_x
  const startPosY = step.position_y

  const handleDrag = (e: MouseEvent) => {
    step.position_x = startPosX + (e.clientX - startX)
    step.position_y = startPosY + (e.clientY - startY)
  }

  const handleDragEnd = () => {
    document.removeEventListener('mousemove', handleDrag)
    document.removeEventListener('mouseup', handleDragEnd)
  }

  document.addEventListener('mousemove', handleDrag)
  document.addEventListener('mouseup', handleDragEnd)
}

const selectStep = (step: any) => {
  selectedStep.value = step
  selectedEdge.value = null
}

const selectEdge = (edge: any) => {
  selectedEdge.value = edge
  selectedStep.value = null
}

const handleStepCommand = (command: string, step: any) => {
  switch (command) {
    case 'edit':
      selectedStep.value = step
      break
    case 'duplicate':
      const newStep = { ...step, id: `step_${++stepIdCounter}`, position_x: step.position_x + 20, position_y: step.position_y + 20 }
      workflow.steps.push(newStep)
      break
    case 'delete':
      workflow.steps = workflow.steps.filter(s => s.id !== step.id)
      workflow.edges = workflow.edges.filter(e => e.source_step_id !== step.id && e.target_step_id !== step.id)
      if (selectedStep.value?.id === step.id) selectedStep.value = null
      break
  }
}

const deleteEdge = () => {
  if (selectedEdge.value) {
    workflow.edges = workflow.edges.filter(e => e.id !== selectedEdge.value.id)
    selectedEdge.value = null
  }
}

const startDrawingEdge = (event: MouseEvent, step: any, _type: string) => {
  event.preventDefault()
  drawingEdge.value = true
  drawingEdgeStart.value = step
  drawingEdgeEnd.value = { x: event.clientX, y: event.clientY }
}

const handleMouseMove = (event: MouseEvent) => {
  if (drawingEdge.value) {
    drawingEdgeEnd.value = { x: event.clientX, y: event.clientY }
  }
}

const handleMouseUp = () => {
  if (drawingEdge.value) {
    drawingEdge.value = false
    drawingEdgeStart.value = null
  }
}

const finishDrawingEdge = (targetStep: any) => {
  if (drawingEdge.value && drawingEdgeStart.value && targetStep.id !== drawingEdgeStart.value.id) {
    // Check if edge already exists
    const exists = workflow.edges.some(
      e => e.source_step_id === drawingEdgeStart.value.id && e.target_step_id === targetStep.id
    )
    if (!exists) {
      workflow.edges.push({
        id: `edge_${++edgeIdCounter}`,
        source_step_id: drawingEdgeStart.value.id,
        target_step_id: targetStep.id,
        label: 'success'
      })
    }
  }
  drawingEdge.value = false
  drawingEdgeStart.value = null
}

const getStepIcon = (type: string) => {
  const found = stepTypes.find(s => s.type === type)
  return found?.icon || '📦'
}

const getAgentName = (agentId: string) => {
  const agent = myAgents.value.find(a => a.id === agentId)
  return agent?.name || 'Select agent'
}

const getStepPosition = (stepId: string) => {
  const step = workflow.steps.find(s => s.id === stepId)
  if (!step) return { x: 0, y: 0 }
  return { x: step.position_x + 80, y: step.position_y + 30 }
}

const getEdgePath = (edge: any) => {
  const source = getStepPosition(edge.source_step_id)
  const target = getStepPosition(edge.target_step_id)
  
  const dx = target.x - source.x
  const dy = target.y - source.y
  const cx = source.x + dx / 2
  
  return `M ${source.x + 80} ${source.y} C ${cx} ${source.y}, ${cx} ${target.y}, ${target.x - 80} ${target.y}`
}

const drawingEdgePath = computed(() => {
  if (!drawingEdge.value || !drawingEdgeStart.value || !canvasRef.value) return ''
  
  const rect = canvasRef.value.getBoundingClientRect()
  const source = getStepPosition(drawingEdgeStart.value.id)
  const target = { x: drawingEdgeEnd.value.x - rect.left, y: drawingEdgeEnd.value.y - rect.top }
  
  return `M ${source.x + 80} ${source.y} L ${target.x} ${target.y}`
})

const handleSave = async () => {
  if (!workflow.name.trim()) {
    ElMessage.error('Workflow name is required')
    return
  }

  saving.value = true
  try {
    const url = workflow.id 
      ? `/api/v1/agent-system/workflows/${workflow.id}`
      : '/api/v1/agent-system/workflows'
    
    const method = workflow.id ? 'PUT' : 'POST'
    
    const response = await fetch(url, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({
        name: workflow.name,
        description: workflow.description,
        icon: workflow.icon,
        status: workflow.status,
        tags: JSON.stringify(workflow.tags)
      })
    })

    if (response.ok) {
      const savedWorkflow = await response.json()
      workflow.id = savedWorkflow.id

      // Save steps and edges
      for (const step of workflow.steps) {
        if (!step.id.startsWith('step_')) continue
        await fetch(`/api/v1/agent-system/workflows/${workflow.id}/steps`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${authStore.token}`
          },
          body: JSON.stringify(step)
        })
      }

      for (const edge of workflow.edges) {
        if (!edge.id.startsWith('edge_')) continue
        await fetch(`/api/v1/agent-system/workflows/${workflow.id}/edges`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${authStore.token}`
          },
          body: JSON.stringify(edge)
        })
      }

      ElMessage.success('Workflow saved')
      emit('saved')
    } else {
      const error = await response.json()
      ElMessage.error(error.error || 'Failed to save workflow')
    }
  } catch (error) {
    ElMessage.error('Failed to save workflow')
  } finally {
    saving.value = false
  }
}

const handleExport = async () => {
  if (!workflow.id) {
    ElMessage.warning('Please save the workflow first')
    return
  }

  try {
    const response = await fetch(`/api/v1/agent-system/workflows/${workflow.id}/export`, {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      const data = await response.json()
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${workflow.name}.workflow.json`
      a.click()
      URL.revokeObjectURL(url)
    }
  } catch (error) {
    ElMessage.error('Failed to export workflow')
  }
}

const handleRun = async () => {
  if (!workflow.id) {
    ElMessage.warning('Please save the workflow first')
    return
  }

  running.value = true
  runningSteps.value.clear()
  completedSteps.value.clear()
  runEvents.value = []

  try {
    let input = {}
    try {
      input = JSON.parse(runInput.value)
    } catch (e) {
      ElMessage.error('Invalid JSON input')
      running.value = false
      return
    }

    const response = await fetch(`/api/v1/agent-system/workflows/${workflow.id}/start`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({ input })
    })

    if (response.ok) {
      currentRun.value = await response.json()
      ElMessage.success('Workflow started')
      
      // Poll for status
      pollRunStatus()
    } else {
      const error = await response.json()
      ElMessage.error(error.error || 'Failed to start workflow')
    }
  } catch (error) {
    ElMessage.error('Failed to start workflow')
  } finally {
    running.value = false
  }
}

const pollRunStatus = async () => {
  if (!currentRun.value) return

  try {
    const response = await fetch(`/api/v1/agent-system/workflows/runs/${currentRun.value.id}`, {
      headers: { 'Authorization': `Bearer ${authStore.token}` }
    })
    if (response.ok) {
      currentRun.value = await response.json()
      
      if (['running', 'paused', 'pending'].includes(currentRun.value.status)) {
        setTimeout(pollRunStatus, 2000)
      }
    }
  } catch (error) {
    console.error('Failed to poll status:', error)
  }
}
</script>

<style scoped>
.workflow-designer {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
}

.designer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color);
}

.header-left, .header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.designer-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.step-palette {
  width: 200px;
  padding: 16px;
  border-right: 1px solid var(--el-border-color);
  overflow-y: auto;
}

.step-palette h4 {
  margin: 0 0 12px 0;
  color: var(--el-text-color-secondary);
}

.step-type-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  margin-bottom: 8px;
  cursor: grab;
  transition: all 0.2s;
}

.step-type-item:hover {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.step-icon {
  font-size: 20px;
}

.step-info {
  flex: 1;
  min-width: 0;
}

.step-name {
  font-weight: 500;
  font-size: 13px;
}

.step-desc {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.workflow-canvas {
  flex: 1;
  position: relative;
  overflow: auto;
  background: 
    linear-gradient(90deg, var(--el-border-color-lighter) 1px, transparent 1px),
    linear-gradient(var(--el-border-color-lighter) 1px, transparent 1px);
  background-size: 20px 20px;
}

.connections-layer {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.edge-path {
  pointer-events: stroke;
  cursor: pointer;
}

.edge-path.selected {
  stroke: var(--el-color-danger);
  stroke-width: 3;
}

.workflow-step {
  position: absolute;
  width: 160px;
  background: var(--el-bg-color);
  border: 2px solid var(--el-border-color);
  border-radius: 8px;
  cursor: move;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.workflow-step:hover {
  border-color: var(--el-color-primary);
}

.workflow-step.selected {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px var(--el-color-primary-light-8);
}

.workflow-step.running {
  border-color: var(--el-color-warning);
  animation: pulse 1s infinite;
}

.workflow-step.completed {
  border-color: var(--el-color-success);
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.step-header {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  background: var(--el-fill-color-light);
  border-radius: 6px 6px 0 0;
}

.step-type-icon {
  font-size: 14px;
}

.step-title {
  flex: 1;
  font-size: 12px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.step-body {
  padding: 8px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.connection-point {
  position: absolute;
  width: 12px;
  height: 12px;
  background: var(--el-color-primary);
  border: 2px solid var(--el-bg-color);
  border-radius: 50%;
  cursor: crosshair;
}

.connection-point.output {
  right: -6px;
  top: 50%;
  transform: translateY(-50%);
}

.connection-point.input {
  left: -6px;
  top: 50%;
  transform: translateY(-50%);
}

.properties-panel {
  width: 280px;
  padding: 16px;
  border-left: 1px solid var(--el-border-color);
  overflow-y: auto;
}

.properties-panel h4 {
  margin: 0 0 12px 0;
  color: var(--el-text-color-secondary);
}

.empty-canvas {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;
  color: var(--el-text-color-secondary);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 8px;
}

.run-status {
  margin-top: 16px;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}

.status-header {
  margin-bottom: 8px;
}

.status-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.status-badge.running { background: var(--el-color-warning-light-9); color: var(--el-color-warning); }
.status-badge.completed { background: var(--el-color-success-light-9); color: var(--el-color-success); }
.status-badge.failed { background: var(--el-color-danger-light-9); color: var(--el-color-danger); }
.status-badge.paused { background: var(--el-color-info-light-9); color: var(--el-color-info); }

.run-events {
  max-height: 200px;
  overflow-y: auto;
}

.run-event {
  padding: 4px 0;
  font-size: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.event-type {
  font-weight: 500;
}

.event-step {
  color: var(--el-text-color-secondary);
  margin-left: 8px;
}
</style>
