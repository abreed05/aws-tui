package secretsmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// SecretsClient wraps the Secrets Manager client
type SecretsClient struct {
	client *secretsmanager.Client
}

// NewSecretsClient creates a new secrets client
func NewSecretsClient(client *secretsmanager.Client) *SecretsClient {
	return &SecretsClient{client: client}
}

// Secret represents a secret with its metadata
type Secret struct {
	Name               string
	ARN                string
	Description        string
	KmsKeyID           string
	RotationEnabled    bool
	RotationLambdaARN  string
	LastChangedDate    time.Time
	LastAccessedDate   time.Time
	LastRotatedDate    time.Time
	DeletedDate        time.Time
	OwningService      string
	PrimaryRegion      string
	Tags               map[string]string
}

// ListSecrets lists all secrets
func (c *SecretsClient) ListSecrets(ctx context.Context) ([]Secret, error) {
	var secrets []Secret
	var nextToken *string

	for {
		input := &secretsmanager.ListSecretsInput{
			NextToken: nextToken,
		}

		output, err := c.client.ListSecrets(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, entry := range output.SecretList {
			secret := Secret{
				Name:              aws.ToString(entry.Name),
				ARN:               aws.ToString(entry.ARN),
				Description:       aws.ToString(entry.Description),
				KmsKeyID:          aws.ToString(entry.KmsKeyId),
				RotationEnabled:   entry.RotationEnabled != nil && *entry.RotationEnabled,
				RotationLambdaARN: aws.ToString(entry.RotationLambdaARN),
				OwningService:     aws.ToString(entry.OwningService),
				PrimaryRegion:     aws.ToString(entry.PrimaryRegion),
				Tags:              make(map[string]string),
			}

			if entry.LastChangedDate != nil {
				secret.LastChangedDate = *entry.LastChangedDate
			}
			if entry.LastAccessedDate != nil {
				secret.LastAccessedDate = *entry.LastAccessedDate
			}
			if entry.LastRotatedDate != nil {
				secret.LastRotatedDate = *entry.LastRotatedDate
			}
			if entry.DeletedDate != nil {
				secret.DeletedDate = *entry.DeletedDate
			}

			for _, tag := range entry.Tags {
				secret.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
			}

			secrets = append(secrets, secret)
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return secrets, nil
}

// GetSecret gets a single secret by name or ARN
func (c *SecretsClient) GetSecret(ctx context.Context, secretID string) (*Secret, error) {
	output, err := c.client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe secret %s: %w", secretID, err)
	}

	secret := &Secret{
		Name:              aws.ToString(output.Name),
		ARN:               aws.ToString(output.ARN),
		Description:       aws.ToString(output.Description),
		KmsKeyID:          aws.ToString(output.KmsKeyId),
		RotationEnabled:   output.RotationEnabled != nil && *output.RotationEnabled,
		RotationLambdaARN: aws.ToString(output.RotationLambdaARN),
		OwningService:     aws.ToString(output.OwningService),
		PrimaryRegion:     aws.ToString(output.PrimaryRegion),
		Tags:              make(map[string]string),
	}

	if output.LastChangedDate != nil {
		secret.LastChangedDate = *output.LastChangedDate
	}
	if output.LastAccessedDate != nil {
		secret.LastAccessedDate = *output.LastAccessedDate
	}
	if output.LastRotatedDate != nil {
		secret.LastRotatedDate = *output.LastRotatedDate
	}
	if output.DeletedDate != nil {
		secret.DeletedDate = *output.DeletedDate
	}

	for _, tag := range output.Tags {
		secret.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	return secret, nil
}

// GetSecretResourcePolicy gets the resource policy for a secret
func (c *SecretsClient) GetSecretResourcePolicy(ctx context.Context, secretID string) (string, error) {
	output, err := c.client.GetResourcePolicy(ctx, &secretsmanager.GetResourcePolicyInput{
		SecretId: aws.String(secretID),
	})
	if err != nil {
		return "", err // Not all secrets have policies
	}

	return aws.ToString(output.ResourcePolicy), nil
}

// GetSecretVersionIDs gets all version IDs for a secret
func (c *SecretsClient) GetSecretVersionIDs(ctx context.Context, secretID string) ([]string, error) {
	var versionIDs []string
	var nextToken *string

	for {
		input := &secretsmanager.ListSecretVersionIdsInput{
			SecretId:  aws.String(secretID),
			NextToken: nextToken,
		}

		output, err := c.client.ListSecretVersionIds(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list secret versions: %w", err)
		}

		for _, version := range output.Versions {
			versionIDs = append(versionIDs, aws.ToString(version.VersionId))
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return versionIDs, nil
}

// GetSecretValue retrieves the actual secret value
func (c *SecretsClient) GetSecretValue(ctx context.Context, secretID string) (string, error) {
	output, err := c.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get secret value: %w", err)
	}

	// Return string value (most common)
	if output.SecretString != nil {
		return aws.ToString(output.SecretString), nil
	}

	// Handle binary secrets (less common)
	if output.SecretBinary != nil {
		return string(output.SecretBinary), nil
	}

	return "", fmt.Errorf("secret has no value")
}

// UpdateSecretValue updates a secret's value
func (c *SecretsClient) UpdateSecretValue(ctx context.Context, secretID, secretValue string) error {
	_, err := c.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(secretID),
		SecretString: aws.String(secretValue),
	})
	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	return nil
}

// CreateSecretParams contains parameters for creating a secret
type CreateSecretParams struct {
	Name        string
	SecretValue string
	Description string
	Tags        map[string]string
}

// CreateSecret creates a new secret
func (c *SecretsClient) CreateSecret(ctx context.Context, params CreateSecretParams) (*Secret, error) {
	input := &secretsmanager.CreateSecretInput{
		Name:         aws.String(params.Name),
		SecretString: aws.String(params.SecretValue),
	}

	if params.Description != "" {
		input.Description = aws.String(params.Description)
	}

	if len(params.Tags) > 0 {
		tags := make([]types.Tag, 0, len(params.Tags))
		for k, v := range params.Tags {
			tags = append(tags, types.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}
		input.Tags = tags
	}

	output, err := c.client.CreateSecret(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}

	// Fetch the created secret to return full details
	return c.GetSecret(ctx, aws.ToString(output.Name))
}

// DeleteSecret schedules a secret for deletion with recovery window
func (c *SecretsClient) DeleteSecret(ctx context.Context, secretID string, recoveryWindowDays int32) error {
	if recoveryWindowDays < 7 || recoveryWindowDays > 30 {
		return fmt.Errorf("recovery window must be between 7 and 30 days")
	}

	_, err := c.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:             aws.String(secretID),
		RecoveryWindowInDays: aws.Int64(int64(recoveryWindowDays)),
	})
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}
