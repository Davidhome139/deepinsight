<template>
  <div class="output-format-selector" @click.stop>
    <el-select 
      v-model="localFormat" 
      :placeholder="t('promptEngineering.selectOutputFormat')" 
      style="width: 100%; margin-bottom: 10px;" 
      :append-to-body="false"
      :popper-append-to-body="false"
      @click.stop
    >
      <el-option :label="t('promptEngineering.defaultFormat')" value="default" />
      <el-option label="JSON" value="json" />
      <el-option label="XML" value="xml" />
      <el-option label="YAML" value="yaml" />
      <el-option label="Markdown" value="markdown" />
    </el-select>
    
    <div v-if="localFormat !== 'default'" class="schema-section">
      <div class="preset-buttons">
        <el-button v-if="localFormat === 'json'" size="small" @click="applyJsonPreset">JSON 示例</el-button>
        <el-button v-if="localFormat === 'xml'" size="small" @click="applyXmlPreset">XML 示例</el-button>
        <el-button v-if="localFormat === 'yaml'" size="small" @click="applyYamlPreset">YAML 示例</el-button>
        <el-button v-if="localFormat === 'markdown'" size="small" @click="applyMarkdownPreset">Markdown 示例</el-button>
      </div>
      <el-input
        v-if="localFormat !== 'markdown'"
        v-model="localSchema"
        type="textarea"
        :rows="4"
        :placeholder="t('promptEngineering.enterSchema')"
        style="width: 100%; margin-top: 10px;"
        @input="handleSchemaChanged"
      />
      <div class="schema-hint">
        {{ t('promptEngineering.schemaHint') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  selectedFormat: string
  schema: string
}>()

const emit = defineEmits<{
  (e: 'format-changed', format: string): void
  (e: 'schema-changed', schema: string): void
}>()

const localFormat = ref(props.selectedFormat)
const localSchema = ref(props.schema)

// Watch for changes in props
watch(() => props.selectedFormat, (newFormat) => {
  localFormat.value = newFormat
})

watch(() => props.schema, (newSchema) => {
  localSchema.value = newSchema
})

watch(localFormat, (newFormat) => {
  emit('format-changed', newFormat)
})

const handleSchemaChanged = () => {
  emit('schema-changed', localSchema.value)
}

const applyJsonPreset = () => {
  localSchema.value = `{
  "name": "string",
  "age": "number",
  "email": "string",
  "hobbies": ["string"],
  "address": {
    "street": "string",
    "city": "string",
    "country": "string"
  }
}`
  handleSchemaChanged()
}

const applyXmlPreset = () => {
  localSchema.value = `<person>
  <name></name>
  <age></age>
  <email></email>
  <hobbies>
    <hobby></hobby>
  </hobbies>
  <address>
    <street></street>
    <city></city>
    <country></country>
  </address>
</person>`
  handleSchemaChanged()
}

const applyYamlPreset = () => {
  localSchema.value = `name: string
age: number
email: string
hobbies:
  - string
address:
  street: string
  city: string
  country: string`
  handleSchemaChanged()
}

const applyMarkdownPreset = () => {
  localSchema.value = `# 标题

## 二级标题

### 三级标题

**粗体文字** 和 *斜体文字*

- 列表项 1
- 列表项 2
- 列表项 3

1. 有序列表项 1
2. 有序列表项 2
3. 有序列表项 3

> 引用文字

\`\`\`
代码块
\`\`\`

[链接文字](https://example.com)

![图片描述](https://example.com/image.jpg)

| 列1 | 列2 | 列3 |
|-----|-----|-----|
| 数据1 | 数据2 | 数据3 |
| 数据4 | 数据5 | 数据6 |`
  handleSchemaChanged()
}
</script>

<style scoped>
.output-format-selector {
  width: 100%;
}

.schema-section {
  margin-top: 10px;
}

.preset-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.schema-hint {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-top: 8px;
  line-height: 1.6;
  white-space: pre-line;
}
</style>
