package secrets

import (
	"context"
	"encoding/json"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Provider struct {
	client *secretsmanager.Client
}

type SlackGitHubSecret struct {
	GitHubToken      string `json:"github_token"`
	SlackWorkflowURL string `json:"slack_workflow_url"`
}

func NewProvider(ctx context.Context) (*Provider, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &Provider{client: secretsmanager.NewFromConfig(cfg)}, nil
}

func (p *Provider) GetSlackGitHubSecret(ctx context.Context, secretID string) (SlackGitHubSecret, error) {
	out, err := p.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &secretID})
	if err != nil {
		return SlackGitHubSecret{}, fmt.Errorf("get secret %s: %w", secretID, err)
	}
	var secret SlackGitHubSecret
	if err := json.Unmarshal([]byte(*out.SecretString), &secret); err != nil {
		return SlackGitHubSecret{}, fmt.Errorf("decode secret %s: %w", secretID, err)
	}
	if secret.GitHubToken == "" {
		return SlackGitHubSecret{}, fmt.Errorf("secret %s missing github_token", secretID)
	}
	if secret.SlackWorkflowURL == "" {
		return SlackGitHubSecret{}, fmt.Errorf("secret %s missing slack_workflow_url", secretID)
	}
	return secret, nil
}
