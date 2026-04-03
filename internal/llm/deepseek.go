// internal/llm/deepseek.go
package llm

import (
	"context"
	"fmt"
	"github.com/go-dev-frame/sponge/pkg/aicli/deepseek"
)

type DeepSeekClient struct {
	client *deepseek.Client
	model  string
}

type Config struct {
	APIKey string
	Model  string // deepseek-chat, deepseek-reasoner, deepseek-coder
}

func NewDeepSeekClient(cfg Config) (*DeepSeekClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY is required")
	}

	model := cfg.Model
	if model == "" {
		model = "deepseek-chat" // 默认模型
	}

	client, err := deepseek.NewClient(cfg.APIKey, deepseek.WithModel(model))
	if err != nil {
		return nil, fmt.Errorf("failed to create DeepSeek client: %w", err)
	}

	return &DeepSeekClient{
		client: client,
		model:  model,
	}, nil
}

// SummarizeIssuesAndPRs 使用 DeepSeek 总结 Issues 和 PRs
func (d *DeepSeekClient) SummarizeIssuesAndPRs(ctx context.Context, repoName, content string) (string, error) {
	prompt := d.buildSummaryPrompt(repoName, content)

	// 发送请求
	response, err := d.client.Send(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("DeepSeek API error: %w", err)
	}

	return response, nil
}

// GenerateDailyReport 生成 AI 增强的每日报告
func (d *DeepSeekClient) GenerateDailyReport(ctx context.Context, repoName, markdownContent string) (string, error) {
	prompt := d.buildReportPrompt(repoName, markdownContent)

	response, err := d.client.Send(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("DeepSeek API error: %w", err)
	}

	return response, nil
}

// buildSummaryPrompt 构建总结提示词
func (d *DeepSeekClient) buildSummaryPrompt(repoName, content string) string {
	return fmt.Sprintf(`你是一个专业的项目管理助手。请分析以下 %s 仓库的 Issues 和 Pull Requests 数据，并提供一份简洁的总结报告。

要求：
1. 按优先级分类：紧急、重要、一般
2. 识别关键问题和瓶颈
3. 提供可行的建议

数据内容：
%s

请用中文回复，格式要清晰易读。`, repoName, content)
}

// buildReportPrompt 构建报告生成提示词
func (d *DeepSeekClient) buildReportPrompt(repoName, content string) string {

	return fmt.Sprintf(`请基于以下 %s 仓库的每日进展数据，生成一份规范的项目每日报告。

报告格式要求：
1. 使用 Markdown 格式
2. 包含以下章节：
   - 今日概览（整体进展描述）
   - 关键进展（重要完成事项）
   - 问题与风险（需要关注的 issues）
   - 明日计划（基于当前进度的建议）

原始数据：
%s

请生成专业、清晰的项目报告。`, repoName, content)
}

// SendPrompt 发送自定义提示词
func (d *DeepSeekClient) SendPrompt(ctx context.Context, prompt string) (string, error) {
	return d.client.Send(ctx, prompt)
}

// SendPromptWithStream 流式响应（用于长文本生成）
func (d *DeepSeekClient) SendPromptWithStream(ctx context.Context, prompt string) (<-chan string, error) {
	reply := d.client.SendStream(ctx, prompt)
	return reply.Content, nil
}
