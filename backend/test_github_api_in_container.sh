#!/bin/bash

# Test the GitHub API endpoint directly in the container
docker exec newdoubao-backend-1 sh -c "curl -s 'https://api.github.com/repos/deepseek-ai/deepseek-coder/releases'"

echo "\n--- Testing with error handling ---"
docker exec newdoubao-backend-1 sh -c "curl -v 'https://api.github.com/repos/deepseek-ai/deepseek-coder/releases' 2>&1 | head -50"
