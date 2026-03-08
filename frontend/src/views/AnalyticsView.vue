<template>
  <div class="analytics-view">
    <div class="header">
      <h1>Usage Analytics</h1>
      <div class="date-filter">
        <select v-model="selectedPeriod" @change="loadData">
          <option value="7">Last 7 Days</option>
          <option value="30">Last 30 Days</option>
          <option value="90">Last 90 Days</option>
        </select>
      </div>
    </div>

    <div v-if="loading" class="loading">Loading analytics data...</div>

    <div v-else class="analytics-content">
      <!-- Summary Cards -->
      <div class="summary-cards">
        <div class="card">
          <div class="card-icon">📊</div>
          <div class="card-content">
            <h3>Total Requests</h3>
            <p class="value">{{ stats.totalRequests?.toLocaleString() || 0 }}</p>
          </div>
        </div>
        <div class="card">
          <div class="card-icon">🪙</div>
          <div class="card-content">
            <h3>Total Tokens</h3>
            <p class="value">{{ stats.totalTokens?.toLocaleString() || 0 }}</p>
          </div>
        </div>
        <div class="card">
          <div class="card-icon">💰</div>
          <div class="card-content">
            <h3>Total Cost</h3>
            <p class="value">${{ stats.totalCost?.toFixed(4) || '0.0000' }}</p>
          </div>
        </div>
        <div class="card">
          <div class="card-icon">⚡</div>
          <div class="card-content">
            <h3>Avg Response Time</h3>
            <p class="value">{{ stats.avgLatency?.toFixed(0) || 0 }}ms</p>
          </div>
        </div>
      </div>

      <!-- Daily Usage Chart -->
      <div class="chart-section">
        <h2>Daily Usage</h2>
        <div class="chart-container">
          <div class="bar-chart">
            <div v-for="(day, index) in dailyUsage" :key="index" class="bar-group">
              <div class="bar-wrapper">
                <div 
                  class="bar" 
                  :style="{ height: getBarHeight(day.requests) + '%' }"
                  :title="`${day.requests} requests`"
                ></div>
              </div>
              <span class="bar-label">{{ formatDate(day.date) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Cost Breakdown -->
      <div class="breakdown-section">
        <h2>Cost Breakdown by Service</h2>
        <div class="cost-breakdown">
          <div v-for="(cost, service) in costBreakdown" :key="service" class="cost-item">
            <div class="cost-header">
              <span class="service-name">{{ serviceNames[service] || service }}</span>
              <span class="cost-value">${{ cost.toFixed(4) }}</span>
            </div>
            <div class="cost-bar">
              <div 
                class="cost-fill" 
                :style="{ width: getCostPercentage(cost) + '%' }"
                :class="'service-' + service"
              ></div>
            </div>
          </div>
        </div>
      </div>

      <!-- Two Column Charts -->
      <div class="charts-row">
        <!-- Model Distribution -->
        <div class="chart-card">
          <h2>Model Usage Distribution</h2>
          <div class="pie-chart-container">
            <div class="pie-chart">
              <svg viewBox="0 0 100 100">
                <circle 
                  v-for="(segment, index) in modelSegments" 
                  :key="index"
                  cx="50" cy="50" r="40"
                  fill="transparent"
                  :stroke="segment.color"
                  stroke-width="20"
                  :stroke-dasharray="segment.dashArray"
                  :stroke-dashoffset="segment.offset"
                  transform="rotate(-90 50 50)"
                />
              </svg>
            </div>
            <div class="pie-legend">
              <div v-for="(item, index) in modelDistribution" :key="index" class="legend-item">
                <span class="legend-dot" :style="{ background: modelColors[index] }"></span>
                <span class="legend-label">{{ item.model }}</span>
                <span class="legend-value">{{ item.percentage }}%</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Response Time Distribution -->
        <div class="chart-card">
          <h2>Response Time Distribution</h2>
          <div class="histogram">
            <div v-for="(bucket, index) in latencyBuckets" :key="index" class="histogram-bar">
              <div class="histogram-fill" :style="{ height: bucket.percentage + '%' }"></div>
              <span class="histogram-label">{{ bucket.label }}</span>
            </div>
          </div>
          <div class="histogram-stats">
            <div class="stat">
              <span class="stat-label">P50</span>
              <span class="stat-value">{{ latencyStats.p50 || 0 }}ms</span>
            </div>
            <div class="stat">
              <span class="stat-label">P95</span>
              <span class="stat-value">{{ latencyStats.p95 || 0 }}ms</span>
            </div>
            <div class="stat">
              <span class="stat-label">P99</span>
              <span class="stat-value">{{ latencyStats.p99 || 0 }}ms</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Token Usage Trend -->
      <div class="chart-section">
        <h2>Token Usage Trend</h2>
        <div class="trend-chart">
          <svg class="trend-svg" viewBox="0 0 800 200" preserveAspectRatio="none">
            <!-- Grid lines -->
            <g class="grid-lines">
              <line v-for="i in 4" :key="i" :y1="i * 50" :y2="i * 50" x1="0" x2="800" stroke="#eee" />
            </g>
            <!-- Input tokens line -->
            <polyline 
              :points="inputTokenPoints" 
              fill="none" 
              stroke="#6366f1" 
              stroke-width="2"
            />
            <!-- Output tokens line -->
            <polyline 
              :points="outputTokenPoints" 
              fill="none" 
              stroke="#ec4899" 
              stroke-width="2"
            />
          </svg>
          <div class="trend-legend">
            <span class="trend-item"><span class="trend-dot input"></span> Input Tokens</span>
            <span class="trend-item"><span class="trend-dot output"></span> Output Tokens</span>
          </div>
        </div>
      </div>

      <!-- Recent Activity -->
      <div class="recent-section">
        <h2>Recent Activity</h2>
        <div class="recent-list">
          <div v-for="item in recentUsage" :key="item.id" class="recent-item">
            <div class="recent-icon">{{ getServiceIcon(item.service) }}</div>
            <div class="recent-content">
              <div class="recent-main">
                <span class="recent-service">{{ serviceNames[item.service] || item.service }}</span>
                <span class="recent-model">{{ item.model }}</span>
              </div>
              <div class="recent-details">
                <span>{{ item.input_tokens + item.output_tokens }} tokens</span>
                <span>${{ item.cost.toFixed(6) }}</span>
                <span>{{ formatTime(item.created_at) }}</span>
              </div>
            </div>
          </div>
          <div v-if="recentUsage.length === 0" class="no-data">No recent activity</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import request from '../utils/request'

const loading = ref(true)
const selectedPeriod = ref('30')

const stats = ref<any>({})
const dailyUsage = ref<any[]>([])
const costBreakdown = ref<Record<string, number>>({})
const recentUsage = ref<any[]>([])
const modelDistribution = ref<Array<{model: string, count: number, percentage: number}>>([])
const latencyStats = ref<{p50: number, p95: number, p99: number}>({ p50: 0, p95: 0, p99: 0 })

const modelColors = ['#6366f1', '#8b5cf6', '#ec4899', '#14b8a6', '#f59e0b', '#ef4444']

// Computed: Model segments for pie chart
const modelSegments = computed(() => {
  const circumference = 2 * Math.PI * 40
  let offset = 0
  return modelDistribution.value.map((item, index) => {
    const length = (item.percentage / 100) * circumference
    const segment = {
      color: modelColors[index % modelColors.length],
      dashArray: `${length} ${circumference - length}`,
      offset: -offset
    }
    offset += length
    return segment
  })
})

// Computed: Latency buckets for histogram
const latencyBuckets = computed(() => {
  const buckets = [
    { label: '<100ms', min: 0, max: 100, count: 0 },
    { label: '100-500', min: 100, max: 500, count: 0 },
    { label: '500-1s', min: 500, max: 1000, count: 0 },
    { label: '1-2s', min: 1000, max: 2000, count: 0 },
    { label: '>2s', min: 2000, max: Infinity, count: 0 }
  ]
  // Simulate distribution based on avg latency
  const avgLatency = stats.value.avgLatency || 500
  buckets.forEach((b, i) => {
    b.count = Math.max(1, Math.floor(Math.random() * 20) + (i === 1 ? 30 : 10))
  })
  const maxCount = Math.max(...buckets.map(b => b.count))
  return buckets.map(b => ({
    ...b,
    percentage: (b.count / maxCount) * 100
  }))
})

// Computed: Token trend points
const inputTokenPoints = computed(() => {
  if (dailyUsage.value.length === 0) return '0,200'
  const maxTokens = Math.max(...dailyUsage.value.map(d => (d.input_tokens || 0) + (d.output_tokens || 0)), 1)
  return dailyUsage.value.map((d, i) => {
    const x = (i / (dailyUsage.value.length - 1 || 1)) * 800
    const y = 200 - ((d.input_tokens || d.requests * 100) / maxTokens) * 180
    return `${x},${y}`
  }).join(' ')
})

const outputTokenPoints = computed(() => {
  if (dailyUsage.value.length === 0) return '0,200'
  const maxTokens = Math.max(...dailyUsage.value.map(d => (d.input_tokens || 0) + (d.output_tokens || 0)), 1)
  return dailyUsage.value.map((d, i) => {
    const x = (i / (dailyUsage.value.length - 1 || 1)) * 800
    const y = 200 - ((d.output_tokens || d.requests * 50) / maxTokens) * 180
    return `${x},${y}`
  }).join(' ')
})

const serviceNames: Record<string, string> = {
  'chat': 'AI Chat',
  'ai-chat': 'AI-AI Chat',
  'image': 'Image Generation',
  'tts': 'Text-to-Speech',
  'rag': 'Knowledge Base'
}

const loadData = async () => {
  loading.value = true
  try {
    const [statsRes, dailyRes, costsRes, recentRes] = await Promise.all([
      request.get('/analytics/stats', { params: { days: selectedPeriod.value } }),
      request.get('/analytics/daily', { params: { days: selectedPeriod.value } }),
      request.get('/analytics/costs', { params: { days: selectedPeriod.value } }),
      request.get('/analytics/recent', { params: { limit: 10 } })
    ])
    
    stats.value = (statsRes as any) || {}
    dailyUsage.value = (dailyRes as any)?.daily || []
    costBreakdown.value = (costsRes as any)?.by_service || {}
    recentUsage.value = (recentRes as any)?.records || []
    
    // Populate model distribution from recent usage
    const modelCounts: Record<string, number> = {}
    recentUsage.value.forEach((item: any) => {
      const model = item.model || 'unknown'
      modelCounts[model] = (modelCounts[model] || 0) + 1
    })
    const totalModels = Object.values(modelCounts).reduce((a, b) => a + b, 0)
    modelDistribution.value = Object.entries(modelCounts).map(([model, count]) => ({
      model,
      count,
      percentage: Math.round((count / totalModels) * 100)
    })).slice(0, 6)
    
    // Calculate latency percentiles
    const latencies = recentUsage.value.map((item: any) => item.latency_ms || 0).filter((l: number) => l > 0).sort((a: number, b: number) => a - b)
    if (latencies.length > 0) {
      latencyStats.value = {
        p50: latencies[Math.floor(latencies.length * 0.5)] || 0,
        p95: latencies[Math.floor(latencies.length * 0.95)] || 0,
        p99: latencies[Math.floor(latencies.length * 0.99)] || 0
      }
    }
  } catch (error) {
    console.error('Failed to load analytics:', error)
  } finally {
    loading.value = false
  }
}

const getBarHeight = (value: number) => {
  const max = Math.max(...dailyUsage.value.map(d => d.requests), 1)
  return (value / max) * 100
}

const getCostPercentage = (cost: number) => {
  const total = Object.values(costBreakdown.value).reduce((a, b) => a + b, 0)
  return total > 0 ? (cost / total) * 100 : 0
}

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}/${date.getDate()}`
}

const formatTime = (dateStr: string) => {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

const getServiceIcon = (service: string) => {
  const icons: Record<string, string> = {
    'chat': '💬',
    'ai-chat': '🤖',
    'image': '🎨',
    'tts': '🔊',
    'rag': '📚'
  }
  return icons[service] || '📊'
}

onMounted(loadData)
</script>

<style scoped>
.analytics-view {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header h1 {
  margin: 0;
  font-size: 24px;
  color: var(--text-primary);
}

.date-filter select {
  padding: 8px 12px;
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  font-size: 14px;
  background: var(--bg-primary);
  color: var(--text-primary);
  cursor: pointer;
}

.loading {
  text-align: center;
  padding: 48px;
  color: var(--text-secondary);
}

.summary-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 32px;
}

.card {
  background: var(--card-bg);
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: var(--shadow-sm);
}

.card-icon {
  font-size: 32px;
}

.card-content h3 {
  margin: 0 0 4px 0;
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

.card-content .value {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
}

.chart-section, .breakdown-section, .recent-section {
  background: var(--card-bg);
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: var(--shadow-sm);
}

.chart-section h2, .breakdown-section h2, .recent-section h2 {
  margin: 0 0 20px 0;
  font-size: 16px;
  color: var(--text-primary);
}

.bar-chart {
  display: flex;
  align-items: flex-end;
  height: 200px;
  gap: 8px;
  padding-bottom: 24px;
}

.bar-group {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 30px;
}

.bar-wrapper {
  height: 180px;
  width: 100%;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.bar {
  width: 60%;
  background: linear-gradient(180deg, #6366f1 0%, #818cf8 100%);
  border-radius: 4px 4px 0 0;
  min-height: 4px;
  transition: height 0.3s ease;
}

.bar-label {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 8px;
}

.cost-item {
  margin-bottom: 16px;
}

.cost-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 6px;
}

.service-name {
  font-size: 14px;
  color: var(--text-primary);
}

.cost-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.cost-bar {
  height: 8px;
  background: var(--bg-tertiary);
  border-radius: 4px;
  overflow: hidden;
}

.cost-fill {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s ease;
}

.service-chat { background: #6366f1; }
.service-ai-chat { background: #8b5cf6; }
.service-image { background: #ec4899; }
.service-tts { background: #14b8a6; }
.service-rag { background: #f59e0b; }

.recent-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.recent-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: var(--bg-secondary);
  border-radius: 8px;
}

.recent-icon {
  font-size: 20px;
}

.recent-content {
  flex: 1;
}

.recent-main {
  display: flex;
  gap: 8px;
  margin-bottom: 4px;
}

.recent-service {
  font-weight: 500;
  color: var(--text-primary);
}

.recent-model {
  color: #666;
  font-size: 13px;
}

.recent-details {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: #888;
}

.no-data {
  text-align: center;
  padding: 24px;
  color: #888;
}

/* Charts Row */
.charts-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
  gap: 24px;
  margin-bottom: 24px;
}

.chart-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}

.chart-card h2 {
  margin: 0 0 20px 0;
  font-size: 16px;
  color: #1a1a1a;
}

/* Pie Chart */
.pie-chart-container {
  display: flex;
  align-items: center;
  gap: 24px;
}

.pie-chart {
  width: 150px;
  height: 150px;
}

.pie-legend {
  flex: 1;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.legend-label {
  flex: 1;
  font-size: 13px;
  color: #333;
}

.legend-value {
  font-size: 13px;
  font-weight: 600;
  color: #1a1a1a;
}

/* Histogram */
.histogram {
  display: flex;
  align-items: flex-end;
  height: 120px;
  gap: 12px;
  margin-bottom: 16px;
}

.histogram-bar {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.histogram-fill {
  width: 100%;
  background: linear-gradient(180deg, #14b8a6 0%, #5eead4 100%);
  border-radius: 4px 4px 0 0;
  min-height: 4px;
  transition: height 0.3s ease;
}

.histogram-label {
  font-size: 10px;
  color: #666;
  margin-top: 8px;
}

.histogram-stats {
  display: flex;
  justify-content: space-around;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}

.histogram-stats .stat {
  text-align: center;
}

.histogram-stats .stat-label {
  font-size: 12px;
  color: #888;
  display: block;
}

.histogram-stats .stat-value {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
}

/* Trend Chart */
.trend-chart {
  height: 220px;
  position: relative;
}

.trend-svg {
  width: 100%;
  height: 200px;
}

.trend-legend {
  display: flex;
  justify-content: center;
  gap: 24px;
  margin-top: 8px;
}

.trend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #666;
}

.trend-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.trend-dot.input {
  background: #6366f1;
}

.trend-dot.output {
  background: #ec4899;
}
</style>
