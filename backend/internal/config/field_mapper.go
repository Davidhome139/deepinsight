package config

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// FieldMapper 字段映射器，用于处理不同命名格式的字段
type FieldMapper struct {
	fieldMap map[string]string // 归一化后的字段名 -> 结构体字段名
}

// NewFieldMapper 创建字段映射器
func NewFieldMapper(target interface{}) *FieldMapper {
	mapper := &FieldMapper{
		fieldMap: make(map[string]string),
	}

	// 解析目标结构体的字段
	t := reflect.TypeOf(target)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// 获取 mapstructure 标签
		tag := field.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}

		// 支持多个标签值（逗号分隔，第一个为主）
		parts := strings.Split(tag, ",")
		primaryTag := parts[0]

		// 注册多种变体
		mapper.registerVariants(field.Name, primaryTag)
	}

	return mapper
}

// registerVariants 注册字段名的多种变体
func (fm *FieldMapper) registerVariants(structField, tag string) {
	// 原始标签
	fm.fieldMap[tag] = structField

	// 小写版本
	fm.fieldMap[strings.ToLower(tag)] = structField

	// 大写版本
	fm.fieldMap[strings.ToUpper(tag)] = structField

	// 下划线版本（如果标签没有下划线）
	if !strings.Contains(tag, "_") {
		underscored := toUnderscore(tag)
		fm.fieldMap[underscored] = structField
		fm.fieldMap[strings.ToLower(underscored)] = structField
	}

	// 驼峰版本（如果标签有下划线）
	if strings.Contains(tag, "_") {
		camel := toCamelCase(tag)
		fm.fieldMap[camel] = structField
		fm.fieldMap[strings.ToLower(camel)] = structField
	}
}

// MapField 将输入字段名映射到结构体字段名
func (fm *FieldMapper) MapField(input string) (string, bool) {
	// 归一化处理
	normalized := normalizeFieldName(input)

	// 尝试直接匹配
	if field, ok := fm.fieldMap[normalized]; ok {
		return field, true
	}

	// 尝试小写匹配
	if field, ok := fm.fieldMap[strings.ToLower(normalized)]; ok {
		return field, true
	}

	// 尝试移除下划线匹配
	noUnderscore := strings.ReplaceAll(normalized, "_", "")
	if field, ok := fm.fieldMap[noUnderscore]; ok {
		return field, true
	}

	return "", false
}

// normalizeFieldName 归一化字段名
func normalizeFieldName(name string) string {
	// 移除空格
	name = strings.TrimSpace(name)
	// 统一处理：转为小写
	name = strings.ToLower(name)
	return name
}

// toUnderscore 将驼峰命名转为下划线命名
func toUnderscore(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// toCamelCase 将下划线命名转为驼峰命名
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if i > 0 && len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}
	return strings.Join(parts, "")
}

// NormalizeMap 归一化 map 的键名
func NormalizeMap(input map[string]interface{}, target interface{}) (map[string]interface{}, error) {
	mapper := NewFieldMapper(target)

	result := make(map[string]interface{})
	for key, value := range input {
		// 首先尝试解析字段别名（支持下划线格式）
		standardKey := ResolveFieldAlias(key)

		if fieldName, ok := mapper.MapField(standardKey); ok {
			result[fieldName] = value
		} else if fieldName, ok := mapper.MapField(key); ok {
			// 尝试直接映射
			result[fieldName] = value
		} else {
			// 如果无法映射，使用标准化后的键名
			result[standardKey] = value
		}
	}

	return result, nil
}

// DecodeWithFieldMapping 使用字段映射解码配置，并将未定义字段存储到 Custom 字段
func DecodeWithFieldMapping(input map[string]interface{}, output interface{}) error {
	// 首先归一化输入
	normalized, err := NormalizeMap(input, output)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] DecodeWithFieldMapping: Normalized input: %+v", normalized)

	// 获取目标结构体的类型信息
	outputType := reflect.TypeOf(output).Elem()
	if outputType.Kind() != reflect.Struct {
		return fmt.Errorf("output must be a pointer to struct")
	}

	// 提取已定义的字段名及其所有大小写变体
	definedFields := make(map[string]bool)
	for i := 0; i < outputType.NumField(); i++ {
		field := outputType.Field(i)
		// 处理mapstructure标签
		if tag, ok := field.Tag.Lookup("mapstructure"); ok {
			tagParts := strings.Split(tag, ",")
			if len(tagParts) > 0 && tagParts[0] != "" {
				primaryTag := tagParts[0]
				// 注册原始标签及其所有变体
				definedFields[primaryTag] = true
				definedFields[strings.ToLower(primaryTag)] = true
				definedFields[strings.ToUpper(primaryTag)] = true
				// 处理驼峰式变体
				camelCase := toCamelCase(primaryTag)
				if camelCase != primaryTag {
					definedFields[camelCase] = true
					definedFields[strings.ToLower(camelCase)] = true
					definedFields[strings.ToUpper(camelCase)] = true
				}
				// 添加首字母大写的变体
				var firstUpper string
				if len(primaryTag) > 0 {
					firstUpper = strings.ToUpper(primaryTag[:1]) + strings.ToLower(primaryTag[1:])
					definedFields[firstUpper] = true
				}
				log.Printf("[DEBUG] DecodeWithFieldMapping: Registered field '%s' with tag '%s' (variants: %s, %s, %s, %s, %s, %s, %s)",
					field.Name, primaryTag, primaryTag, strings.ToLower(primaryTag), strings.ToUpper(primaryTag),
					camelCase, strings.ToLower(camelCase), strings.ToUpper(camelCase), firstUpper)
			}
		}
		// 处理json标签作为备选
		if tag, ok := field.Tag.Lookup("json"); ok {
			tagParts := strings.Split(tag, ",")
			if len(tagParts) > 0 && tagParts[0] != "" {
				primaryTag := tagParts[0]
				// 注册原始标签及其所有变体
				definedFields[primaryTag] = true
				definedFields[strings.ToLower(primaryTag)] = true
				definedFields[strings.ToUpper(primaryTag)] = true
				// 处理驼峰式变体
				camelCase := toCamelCase(primaryTag)
				if camelCase != primaryTag {
					definedFields[camelCase] = true
					definedFields[strings.ToLower(camelCase)] = true
					definedFields[strings.ToUpper(camelCase)] = true
				}
				// 添加首字母大写的变体
				if len(primaryTag) > 0 {
					firstUpper := strings.ToUpper(primaryTag[:1]) + strings.ToLower(primaryTag[1:])
					definedFields[firstUpper] = true
				}
			}
		}
	}

	log.Printf("[DEBUG] DecodeWithFieldMapping: Defined fields: %+v", definedFields)

	// 分离已定义字段和未定义字段
	definedData := make(map[string]interface{})
	customData := make(map[string]interface{})

	for key, value := range normalized {
		if definedFields[key] {
			// 已定义的字段
			definedData[key] = value
			log.Printf("[DEBUG] DecodeWithFieldMapping: Key '%s' is defined, adding to definedData", key)
		} else {
			// 未定义的字段，存储到customData
			customData[key] = value
			log.Printf("[DEBUG] DecodeWithFieldMapping: Key '%s' is NOT defined, adding to customData", key)
		}
	}

	log.Printf("[DEBUG] DecodeWithFieldMapping: definedData: %+v", definedData)
	log.Printf("[DEBUG] DecodeWithFieldMapping: customData: %+v", customData)

	// 使用 mapstructure 解码已定义字段
	config := &mapstructure.DecoderConfig{
		Result:           output,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
		MatchName: func(mapKey, fieldName string) bool {
			// 自定义匹配逻辑
			normalizedMapKey := normalizeFieldName(mapKey)
			normalizedFieldName := normalizeFieldName(fieldName)
			matched := strings.EqualFold(normalizedMapKey, normalizedFieldName)
			// 添加调试日志
			log.Printf("[DEBUG] MatchName: mapKey='%s', fieldName='%s', normalizedMapKey='%s', normalizedFieldName='%s', matched=%v\n",
				mapKey, fieldName, normalizedMapKey, normalizedFieldName, matched)
			return matched
		},
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	// 执行解码
	if err := decoder.Decode(definedData); err != nil {
		return err
	}

	// 如果输出结构体有 Custom 字段，将未定义字段存储进去
	customField, exists := outputType.FieldByName("Custom")
	if exists && customField.Type == reflect.TypeOf(map[string]interface{}{}) {
		customFieldValue := reflect.ValueOf(output).Elem().FieldByName("Custom")

		// 如果 Custom 字段还未初始化，先初始化
		if customFieldValue.IsNil() {
			customFieldValue.Set(reflect.MakeMap(customFieldValue.Type()))
		}

		// 将未定义字段存储到 Custom 字段
		for key, value := range customData {
			customFieldValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		}
	}

	return nil
}

// CommonFieldAliases 常用字段别名映射
var CommonFieldAliases = map[string][]string{
	"api_key":               {"apikey", "apiKey", "API_KEY", "api-key"},
	"secret_key":            {"secretkey", "secretKey", "SECRET_KEY", "secret-key"},
	"secret_id":             {"secretid", "secretId", "SECRET_ID", "secret-id"},
	"group_id":              {"groupid", "groupId", "GROUP_ID", "group-id"},
	"base_url":              {"baseurl", "baseUrl", "BASE_URL", "base-url"},
	"auth_type":             {"authtype", "authType", "AUTH_TYPE", "auth-type"},
	"token_url":             {"tokenurl", "tokenUrl", "TOKEN_URL", "token-url"},
	"max_completion_tokens": {"maxcompletiontokens", "maxCompletionTokens", "MAX_COMPLETION_TOKENS"},
	"top_p":                 {"topp", "topP", "TOP_P"},
	"default_model":         {"defaultmodel", "defaultModel", "DEFAULT_MODEL"},
	"max_tokens":            {"maxtokens", "maxTokens", "MAX_TOKENS"},
	"retry_times":           {"retrytimes", "retryTimes", "RETRY_TIMES"},
	"retry_delay":           {"retrydelay", "retryDelay", "RETRY_DELAY"},
	"enable_thinking":       {"enablethinking", "enableThinking", "ENABLE_THINKING", "enable-thinking"},
	"type":                  {"server_type", "serverType", "SERVER_TYPE", "server-type", "Type", "ServerType"},
	"allowed_paths":         {"allowedPaths", "ALLOWED_PATHS", "allowed-paths"},
}

// ResolveFieldAlias 解析字段别名，返回标准化的字段名
func ResolveFieldAlias(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	input = strings.ReplaceAll(input, "-", "_")

	for standard, aliases := range CommonFieldAliases {
		if input == standard {
			return standard
		}
		for _, alias := range aliases {
			if strings.ToLower(alias) == input {
				return standard
			}
		}
	}

	return input
}
