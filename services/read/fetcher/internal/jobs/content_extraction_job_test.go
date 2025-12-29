package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContentExtractionJob_NewContentExtractionJob_WithNilConfig(t *testing.T) {
	job := NewContentExtractionJob(nil, nil)

	assert.NotNil(t, job)
	assert.NotNil(t, job.config)
	assert.Equal(t, 20, job.config.BatchSize)
	assert.Equal(t, 30*time.Second, job.config.ProcessInterval)
}

func TestContentExtractionJob_NewContentExtractionJob_WithCustomConfig(t *testing.T) {
	customConfig := &ContentExtractionJobConfig{
		BatchSize:       50,
		ProcessInterval: 1 * time.Minute,
	}

	job := NewContentExtractionJob(customConfig, nil)

	assert.NotNil(t, job)
	assert.Equal(t, customConfig, job.config)
	assert.Equal(t, 50, job.config.BatchSize)
	assert.Equal(t, 1*time.Minute, job.config.ProcessInterval)
}

func TestDefaultContentExtractionJobConfig(t *testing.T) {
	config := DefaultContentExtractionJobConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 20, config.BatchSize)
	assert.Equal(t, 30*time.Second, config.ProcessInterval)
}
