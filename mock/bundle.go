// Package mock provide gozix mocks.
package mock

import (
	"github.com/sarulabs/di"
	"github.com/stretchr/testify/mock"
)

// Bundle mock.
type Bundle struct {
	mock.Mock
}

// BundleDependsOn mock.
type BundleDependsOn struct {
	Bundle
}

// Name implementation.
func (b *Bundle) Name() string {
	return b.Called().String(0)
}

// Build implementation.
func (b *Bundle) Build(builder *di.Builder) error {
	return b.Called(builder).Error(0)
}

// DependsOn implementation.
func (b *BundleDependsOn) DependsOn() []string {
	return b.Called().Get(0).([]string)
}
