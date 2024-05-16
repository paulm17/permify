package yugabyte

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockYBDatabase simulates the behavior of a real PQDatabase.
type MockYBDatabase struct {
	mock.Mock
}

func (m *MockYBDatabase) GetEngineType() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockYBDatabase) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockYBDatabase) IsReady(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}
