#!/bin/bash
# scripts/run.sh

set -e

echo "===> Setting up GitHub Sentinel..."

# 设置GitHub Token（请替换为你的token）
export GITHUB_TOKEN="your_github_token_here"

# 创建数据目录
mkdir -p data

# 运行程序
echo "===> Starting GitHub Sentinel..."
go run cmd/server/main.go