import json

# 验证JSON文件格式
try:
    with open('d:\\apps\\newDouBao\\backend\\plans\\agent_configs.json', 'r', encoding='utf-8') as f:
        data = json.load(f)
    print('JSON格式正确')
    print(f'文件包含 {len(data)} 个Agent配置')
except json.JSONDecodeError as e:
    print(f'JSON格式错误: {e}')
except Exception as e:
    print(f'其他错误: {e}')