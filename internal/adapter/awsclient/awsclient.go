package awsclient

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

func Load(ctx context.Context, region string) (aws.Config, error) {
	return awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
}

// BaseEndpoint retorna um *string p/ o campo BaseEndpoint dos Options dos clients,
// ou nil quando endpoint vazio (produção usa o endpoint real da AWS).
func BaseEndpoint(endpoint string) *string {
	if endpoint == "" {
		return nil
	}
	return &endpoint
}
