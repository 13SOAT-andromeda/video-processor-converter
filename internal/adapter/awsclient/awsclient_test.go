package awsclient_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/awsclient"
)

func TestLoad(t *testing.T) {
	cfg, err := awsclient.Load(context.Background(), "us-east-1")
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", cfg.Region)
}

func TestBaseEndpoint(t *testing.T) {
	assert.Nil(t, awsclient.BaseEndpoint(""))

	got := awsclient.BaseEndpoint("http://localhost:4566")
	require.NotNil(t, got)
	assert.Equal(t, "http://localhost:4566", *got)
}
