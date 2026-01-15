package aws

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ClientManager manages AWS service clients with profile/region switching
type ClientManager struct {
	mu            sync.RWMutex
	currentConfig aws.Config
	profile       string
	region        string
	accountID     string

	// Lazily initialized service clients
	iamClient    *iam.Client
	ec2Client    *ec2.Client
	kmsClient    *kms.Client
	smClient     *secretsmanager.Client
	stsClient    *sts.Client
	rdsClient    *rds.Client
	ecsClient    *ecs.Client
	lambdaClient *lambda.Client
	s3Client     *s3.Client
}

// NewClientManager creates a new AWS client manager
func NewClientManager() *ClientManager {
	return &ClientManager{
		region: "us-east-1",
	}
}

// Configure initializes the client manager with a specific profile and region
func (cm *ClientManager) Configure(ctx context.Context, profile, region string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	opts := []func(*config.LoadOptions) error{}

	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	if profile != "" && profile != "default" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config for profile '%s': %w. Check your ~/.aws/config and ~/.aws/credentials files, or set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables", profile, err)
	}

	cm.currentConfig = cfg
	cm.profile = profile
	if region != "" {
		cm.region = region
	} else if cfg.Region != "" {
		cm.region = cfg.Region
	} else {
		cm.region = "us-east-1" // Fallback
	}

	// Reset cached clients so they get recreated with new config
	cm.iamClient = nil
	cm.ec2Client = nil
	cm.kmsClient = nil
	cm.smClient = nil
	cm.stsClient = nil
	cm.rdsClient = nil
	cm.ecsClient = nil
	cm.lambdaClient = nil
	cm.s3Client = nil
	cm.accountID = ""

	return nil
}

// GetAccountID returns the current AWS account ID
func (cm *ClientManager) GetAccountID(ctx context.Context) (string, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.accountID != "" {
		return cm.accountID, nil
	}

	client := cm.getSTS()
	result, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("AWS credentials not configured or invalid. Profile: '%s'. Error: %w", cm.profile, err)
	}

	cm.accountID = aws.ToString(result.Account)
	return cm.accountID, nil
}

// SwitchProfile changes the AWS profile while keeping the same region
func (cm *ClientManager) SwitchProfile(ctx context.Context, profile string) error {
	cm.mu.RLock()
	region := cm.region
	cm.mu.RUnlock()

	return cm.Configure(ctx, profile, region)
}

// SwitchRegion changes the AWS region while keeping the same profile
func (cm *ClientManager) SwitchRegion(ctx context.Context, region string) error {
	cm.mu.RLock()
	profile := cm.profile
	cm.mu.RUnlock()

	return cm.Configure(ctx, profile, region)
}

// GetCurrentContext returns the current profile and region
func (cm *ClientManager) GetCurrentContext() (profile, region string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.profile, cm.region
}

// Profile returns the current profile name
func (cm *ClientManager) Profile() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.profile
}

// Region returns the current region
func (cm *ClientManager) Region() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.region
}

// IAM returns the IAM client (lazily initialized)
func (cm *ClientManager) IAM() *iam.Client {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.iamClient == nil {
		cm.iamClient = iam.NewFromConfig(cm.currentConfig)
	}
	return cm.iamClient
}

// EC2 returns the EC2 client (lazily initialized)
func (cm *ClientManager) EC2() *ec2.Client {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.ec2Client == nil {
		cm.ec2Client = ec2.NewFromConfig(cm.currentConfig)
	}
	return cm.ec2Client
}

// KMS returns the KMS client (lazily initialized)
func (cm *ClientManager) KMS() *kms.Client {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.kmsClient == nil {
		cm.kmsClient = kms.NewFromConfig(cm.currentConfig)
	}
	return cm.kmsClient
}

// SecretsManager returns the Secrets Manager client (lazily initialized)
func (cm *ClientManager) SecretsManager() *secretsmanager.Client {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.smClient == nil {
		cm.smClient = secretsmanager.NewFromConfig(cm.currentConfig)
	}
	return cm.smClient
}

// RDS returns the RDS client (lazily initialized)
func (cm *ClientManager) RDS() *rds.Client {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.rdsClient == nil {
		cm.rdsClient = rds.NewFromConfig(cm.currentConfig)
	}
	return cm.rdsClient
}

// ECS returns the ECS client (lazily initialized)
func (cm *ClientManager) ECS() *ecs.Client {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.ecsClient == nil {
		cm.ecsClient = ecs.NewFromConfig(cm.currentConfig)
	}
	return cm.ecsClient
}

// Lambda returns the Lambda client (lazily initialized)
func (cm *ClientManager) Lambda() *lambda.Client {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.lambdaClient == nil {
		cm.lambdaClient = lambda.NewFromConfig(cm.currentConfig)
	}
	return cm.lambdaClient
}

// S3 returns the S3 client (lazily initialized)
func (cm *ClientManager) S3() *s3.Client {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.s3Client == nil {
		cm.s3Client = s3.NewFromConfig(cm.currentConfig)
	}
	return cm.s3Client
}

// STS returns the STS client (lazily initialized)
func (cm *ClientManager) STS() *sts.Client {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.getSTS()
}

func (cm *ClientManager) getSTS() *sts.Client {
	if cm.stsClient == nil {
		cm.stsClient = sts.NewFromConfig(cm.currentConfig)
	}
	return cm.stsClient
}

// ValidateCredentials checks if the current credentials are valid
func (cm *ClientManager) ValidateCredentials(ctx context.Context) error {
	client := cm.STS()
	_, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("invalid AWS credentials: %w", err)
	}
	return nil
}
