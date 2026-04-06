#!/bin/bash

# Test the GitHub API endpoint
curl -s 'https://api.github.com/repos/deepseek-ai/deepseek-coder/releases'

# Test with jq
curl -s 'https://api.github.com/repos/deepseek-ai/deepseek-coder/releases' | jq -r '[.[0].tag_name, .[0].published_at] | join(", ")'
