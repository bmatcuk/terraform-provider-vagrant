package main

import "time"

// Default timeout values for vagrant VM resource creation
const (
	VagrantVMCreatedTimeout = 2 * time.Minute
)

// VagrantConfig is for provider-level configuration.
type VagrantConfig struct {
}
