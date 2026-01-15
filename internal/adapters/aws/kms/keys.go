package kms

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// KeysClient wraps the KMS client for key operations
type KeysClient struct {
	client *kms.Client
}

// NewKeysClient creates a new KMS keys client
func NewKeysClient(client *kms.Client) *KeysClient {
	return &KeysClient{client: client}
}

// Key represents a KMS key with its metadata
type Key struct {
	KeyID         string
	KeyARN        string
	AliasName     string
	Description   string
	KeyState      string
	KeyUsage      string
	KeySpec       string
	Origin        string
	MultiRegion   bool
	CreationDate  time.Time
	Enabled       bool
	CustomerOwned bool
	Tags          map[string]string
}

// ListKeys lists all KMS keys with their aliases
func (c *KeysClient) ListKeys(ctx context.Context) ([]Key, error) {
	// First, get all aliases to map them to keys
	aliasMap, err := c.getAliasMap(ctx)
	if err != nil {
		return nil, err
	}

	var keys []Key
	var nextMarker *string

	for {
		input := &kms.ListKeysInput{
			Marker: nextMarker,
		}

		output, err := c.client.ListKeys(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list KMS keys: %w", err)
		}

		for _, keyEntry := range output.Keys {
			keyID := aws.ToString(keyEntry.KeyId)

			// Get key details
			keyDetails, err := c.client.DescribeKey(ctx, &kms.DescribeKeyInput{
				KeyId: keyEntry.KeyId,
			})
			if err != nil {
				continue // Skip keys we can't describe
			}

			metadata := keyDetails.KeyMetadata
			key := Key{
				KeyID:         keyID,
				KeyARN:        aws.ToString(metadata.Arn),
				AliasName:     aliasMap[keyID],
				Description:   aws.ToString(metadata.Description),
				KeyState:      string(metadata.KeyState),
				KeyUsage:      string(metadata.KeyUsage),
				KeySpec:       string(metadata.KeySpec),
				Origin:        string(metadata.Origin),
				MultiRegion:   metadata.MultiRegion != nil && *metadata.MultiRegion,
				Enabled:       metadata.Enabled,
				CustomerOwned: metadata.KeyManager == types.KeyManagerTypeCustomer,
				Tags:          make(map[string]string),
			}

			if metadata.CreationDate != nil {
				key.CreationDate = *metadata.CreationDate
			}

			keys = append(keys, key)
		}

		if !output.Truncated {
			break
		}
		nextMarker = output.NextMarker
	}

	return keys, nil
}

// GetKey gets a single KMS key by ID
func (c *KeysClient) GetKey(ctx context.Context, keyID string) (*Key, error) {
	output, err := c.client.DescribeKey(ctx, &kms.DescribeKeyInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe key %s: %w", keyID, err)
	}

	metadata := output.KeyMetadata

	// Get aliases for this key
	aliasesOutput, _ := c.client.ListAliases(ctx, &kms.ListAliasesInput{
		KeyId: aws.String(keyID),
	})

	var aliasName string
	if aliasesOutput != nil && len(aliasesOutput.Aliases) > 0 {
		aliasName = aws.ToString(aliasesOutput.Aliases[0].AliasName)
	}

	// Get tags
	tags := make(map[string]string)
	tagsOutput, _ := c.client.ListResourceTags(ctx, &kms.ListResourceTagsInput{
		KeyId: aws.String(keyID),
	})
	if tagsOutput != nil {
		for _, tag := range tagsOutput.Tags {
			tags[aws.ToString(tag.TagKey)] = aws.ToString(tag.TagValue)
		}
	}

	key := &Key{
		KeyID:         aws.ToString(metadata.KeyId),
		KeyARN:        aws.ToString(metadata.Arn),
		AliasName:     aliasName,
		Description:   aws.ToString(metadata.Description),
		KeyState:      string(metadata.KeyState),
		KeyUsage:      string(metadata.KeyUsage),
		KeySpec:       string(metadata.KeySpec),
		Origin:        string(metadata.Origin),
		MultiRegion:   metadata.MultiRegion != nil && *metadata.MultiRegion,
		Enabled:       metadata.Enabled,
		CustomerOwned: metadata.KeyManager == types.KeyManagerTypeCustomer,
		Tags:          tags,
	}

	if metadata.CreationDate != nil {
		key.CreationDate = *metadata.CreationDate
	}

	return key, nil
}

// GetKeyPolicy gets the key policy for a KMS key
func (c *KeysClient) GetKeyPolicy(ctx context.Context, keyID string) (string, error) {
	output, err := c.client.GetKeyPolicy(ctx, &kms.GetKeyPolicyInput{
		KeyId:      aws.String(keyID),
		PolicyName: aws.String("default"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get key policy: %w", err)
	}

	return aws.ToString(output.Policy), nil
}

// GetKeyRotationStatus gets the rotation status for a KMS key
func (c *KeysClient) GetKeyRotationStatus(ctx context.Context, keyID string) (bool, error) {
	output, err := c.client.GetKeyRotationStatus(ctx, &kms.GetKeyRotationStatusInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return false, err // Rotation status not available for all key types
	}

	return output.KeyRotationEnabled, nil
}

func (c *KeysClient) getAliasMap(ctx context.Context) (map[string]string, error) {
	aliasMap := make(map[string]string)
	var nextMarker *string

	for {
		input := &kms.ListAliasesInput{
			Marker: nextMarker,
		}

		output, err := c.client.ListAliases(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list aliases: %w", err)
		}

		for _, alias := range output.Aliases {
			if alias.TargetKeyId != nil {
				keyID := aws.ToString(alias.TargetKeyId)
				// Only keep the first alias for each key
				if _, exists := aliasMap[keyID]; !exists {
					aliasMap[keyID] = aws.ToString(alias.AliasName)
				}
			}
		}

		if !output.Truncated {
			break
		}
		nextMarker = output.NextMarker
	}

	return aliasMap, nil
}
