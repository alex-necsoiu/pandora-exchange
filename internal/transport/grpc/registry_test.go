package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNewServiceRegistry(t *testing.T) {
	t.Run("creates registry with defaults", func(t *testing.T) {
		grpcServer := grpc.NewServer()
		defer grpcServer.Stop()

		registry := NewServiceRegistry(grpcServer)
		require.NotNil(t, registry)
		assert.NotNil(t, registry.services)
		assert.NotNil(t, registry.versions)
		assert.NotNil(t, registry.healthServer)
		assert.True(t, registry.enableReflection)
	})

	t.Run("respects WithReflection option", func(t *testing.T) {
		grpcServer := grpc.NewServer()
		defer grpcServer.Stop()

		registry := NewServiceRegistry(grpcServer, WithReflection(false))
		require.NotNil(t, registry)
		assert.False(t, registry.enableReflection)
	})
}

func TestServiceRegistry_RegisterService(t *testing.T) {
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()
	registry := NewServiceRegistry(grpcServer)

	t.Run("registers service successfully", func(t *testing.T) {
		info := &ServiceInfo{
			Name:        "pandora.user.v1.UserService",
			Version:     "v1",
			Description: "User management service",
			Methods:     []string{"GetUser", "CreateUser"},
			ProtoFile:   "proto/user_service.proto",
		}

		err := registry.RegisterService(info)
		require.NoError(t, err)

		// Verify service is registered
		retrieved, err := registry.GetService("pandora.user.v1.UserService", "v1")
		require.NoError(t, err)
		assert.Equal(t, info.Name, retrieved.Name)
		assert.Equal(t, info.Version, retrieved.Version)
		assert.Equal(t, info.Methods, retrieved.Methods)
	})

	t.Run("returns error for empty service name", func(t *testing.T) {
		info := &ServiceInfo{
			Name:    "",
			Version: "v1",
		}

		err := registry.RegisterService(info)
		assert.ErrorIs(t, err, ErrInvalidServiceName)
	})

	t.Run("returns error for empty version", func(t *testing.T) {
		info := &ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "",
		}

		err := registry.RegisterService(info)
		assert.ErrorIs(t, err, ErrInvalidVersion)
	})

	t.Run("returns error for duplicate service", func(t *testing.T) {
		info := &ServiceInfo{
			Name:    "pandora.payment.v1.PaymentService",
			Version: "v1",
			Methods: []string{"ProcessPayment"},
		}

		err := registry.RegisterService(info)
		require.NoError(t, err)

		// Try to register again
		err = registry.RegisterService(info)
		assert.ErrorIs(t, err, ErrServiceAlreadyRegistered)
	})

	t.Run("allows multiple versions of same service", func(t *testing.T) {
		infoV1 := &ServiceInfo{
			Name:    "pandora.order.v1.OrderService",
			Version: "v1",
			Methods: []string{"CreateOrder"},
		}

		infoV2 := &ServiceInfo{
			Name:    "pandora.order.v2.OrderService",
			Version: "v2",
			Methods: []string{"CreateOrder", "CancelOrder"},
		}

		err := registry.RegisterService(infoV1)
		require.NoError(t, err)

		err = registry.RegisterService(infoV2)
		require.NoError(t, err)
	})
}

func TestServiceRegistry_GetService(t *testing.T) {
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()
	registry := NewServiceRegistry(grpcServer)

	info := &ServiceInfo{
		Name:        "pandora.user.v1.UserService",
		Version:     "v1",
		Description: "User service",
		Methods:     []string{"GetUser"},
	}
	registry.RegisterService(info)

	t.Run("retrieves registered service", func(t *testing.T) {
		retrieved, err := registry.GetService("pandora.user.v1.UserService", "v1")
		require.NoError(t, err)
		assert.Equal(t, info.Name, retrieved.Name)
		assert.Equal(t, info.Version, retrieved.Version)
	})

	t.Run("returns error for non-existent service", func(t *testing.T) {
		_, err := registry.GetService("non.existent.Service", "v1")
		assert.ErrorIs(t, err, ErrServiceNotFound)
	})

	t.Run("returns error for empty service name", func(t *testing.T) {
		_, err := registry.GetService("", "v1")
		assert.ErrorIs(t, err, ErrInvalidServiceName)
	})

	t.Run("returns copy of service info", func(t *testing.T) {
		retrieved1, _ := registry.GetService("pandora.user.v1.UserService", "v1")
		retrieved2, _ := registry.GetService("pandora.user.v1.UserService", "v1")

		// Modifying one should not affect the other
		retrieved1.Description = "Modified"
		assert.NotEqual(t, retrieved1.Description, retrieved2.Description)
	})
}

func TestServiceRegistry_ListServices(t *testing.T) {
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()
	registry := NewServiceRegistry(grpcServer)

	t.Run("returns empty list when no services registered", func(t *testing.T) {
		services := registry.ListServices()
		assert.Empty(t, services)
	})

	t.Run("returns all registered services", func(t *testing.T) {
		info1 := &ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v1",
		}
		info2 := &ServiceInfo{
			Name:    "pandora.payment.v1.PaymentService",
			Version: "v1",
		}

		registry.RegisterService(info1)
		registry.RegisterService(info2)

		services := registry.ListServices()
		assert.Len(t, services, 2)
	})
}

func TestServiceRegistry_GetVersions(t *testing.T) {
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()
	registry := NewServiceRegistry(grpcServer)

	t.Run("returns all versions for a service", func(t *testing.T) {
		registry.RegisterService(&ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v1",
		})
		registry.RegisterService(&ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v2",
		})

		versions, err := registry.GetVersions("pandora.user.v1.UserService")
		require.NoError(t, err)
		assert.Len(t, versions, 2)
		assert.Contains(t, versions, "v1")
		assert.Contains(t, versions, "v2")
	})

	t.Run("returns error for non-existent service", func(t *testing.T) {
		_, err := registry.GetVersions("non.existent.Service")
		assert.ErrorIs(t, err, ErrServiceNotFound)
	})

	t.Run("returns error for empty service name", func(t *testing.T) {
		_, err := registry.GetVersions("")
		assert.ErrorIs(t, err, ErrInvalidServiceName)
	})
}

func TestServiceRegistry_GetLatestVersion(t *testing.T) {
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()
	registry := NewServiceRegistry(grpcServer)

	t.Run("returns latest version", func(t *testing.T) {
		registry.RegisterService(&ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v1",
		})
		registry.RegisterService(&ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v2",
		})

		latestVersion, err := registry.GetLatestVersion("pandora.user.v1.UserService")
		require.NoError(t, err)
		assert.Equal(t, "v2", latestVersion)
	})

	t.Run("returns error for non-existent service", func(t *testing.T) {
		_, err := registry.GetLatestVersion("non.existent.Service")
		assert.ErrorIs(t, err, ErrServiceNotFound)
	})
}

func TestServiceRegistry_SetServiceHealth(t *testing.T) {
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()
	registry := NewServiceRegistry(grpcServer)

	t.Run("sets service health status", func(t *testing.T) {
		info := &ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v1",
		}
		registry.RegisterService(info)

		// Should not panic
		registry.SetServiceHealth("pandora.user.v1.UserService", true)
		registry.SetServiceHealth("pandora.user.v1.UserService", false)
	})
}

func TestServiceRegistry_Shutdown(t *testing.T) {
	grpcServer := grpc.NewServer()
	registry := NewServiceRegistry(grpcServer)

	info := &ServiceInfo{
		Name:    "pandora.user.v1.UserService",
		Version: "v1",
	}
	registry.RegisterService(info)

	t.Run("shuts down gracefully within timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := registry.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestCompatibilityChecker_CheckCompatibility(t *testing.T) {
	checker := NewCompatibilityChecker()

	t.Run("compatible when adding new methods", func(t *testing.T) {
		oldInfo := &ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v1",
			Methods: []string{"GetUser", "CreateUser"},
		}

		newInfo := &ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v2",
			Methods: []string{"GetUser", "CreateUser", "DeleteUser"},
		}

		result, err := checker.CheckCompatibility(oldInfo, newInfo)
		require.NoError(t, err)
		assert.True(t, result.IsCompatible)
		assert.Len(t, result.BreakingChanges, 0)
		assert.Greater(t, len(result.Warnings), 0) // New method warning
	})

	t.Run("breaking change when removing methods", func(t *testing.T) {
		oldInfo := &ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v1",
			Methods: []string{"GetUser", "CreateUser", "DeleteUser"},
		}

		newInfo := &ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v2",
			Methods: []string{"GetUser", "CreateUser"},
		}

		result, err := checker.CheckCompatibility(oldInfo, newInfo)
		assert.ErrorIs(t, err, ErrBreakingChange)
		assert.False(t, result.IsCompatible)
		assert.Greater(t, len(result.BreakingChanges), 0)
	})

	t.Run("breaking change when service name changes", func(t *testing.T) {
		oldInfo := &ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v1",
			Methods: []string{"GetUser"},
		}

		newInfo := &ServiceInfo{
			Name:    "pandora.account.v1.AccountService",
			Version: "v2",
			Methods: []string{"GetUser"},
		}

		result, err := checker.CheckCompatibility(oldInfo, newInfo)
		assert.ErrorIs(t, err, ErrBreakingChange)
		assert.False(t, result.IsCompatible)
	})

	t.Run("compatible when no changes", func(t *testing.T) {
		oldInfo := &ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v1",
			Methods: []string{"GetUser", "CreateUser"},
		}

		newInfo := &ServiceInfo{
			Name:    "pandora.user.v1.UserService",
			Version: "v2",
			Methods: []string{"GetUser", "CreateUser"},
		}

		result, err := checker.CheckCompatibility(oldInfo, newInfo)
		require.NoError(t, err)
		assert.True(t, result.IsCompatible)
		assert.Len(t, result.BreakingChanges, 0)
	})
}

func TestCompatibilityChecker_GetCheckResult(t *testing.T) {
	checker := NewCompatibilityChecker()

	oldInfo := &ServiceInfo{
		Name:    "pandora.user.v1.UserService",
		Version: "v1",
		Methods: []string{"GetUser"},
	}

	newInfo := &ServiceInfo{
		Name:    "pandora.user.v1.UserService",
		Version: "v2",
		Methods: []string{"GetUser", "CreateUser"},
	}

	t.Run("retrieves stored check result", func(t *testing.T) {
		result, err := checker.CheckCompatibility(oldInfo, newInfo)
		require.NoError(t, err)

		retrieved, err := checker.GetCheckResult("v1", "v2")
		require.NoError(t, err)
		assert.Equal(t, result.OldVersion, retrieved.OldVersion)
		assert.Equal(t, result.NewVersion, retrieved.NewVersion)
		assert.Equal(t, result.IsCompatible, retrieved.IsCompatible)
	})

	t.Run("returns error for non-existent check", func(t *testing.T) {
		_, err := checker.GetCheckResult("v1", "v3")
		assert.Error(t, err)
	})
}

func TestServiceRegistry_ConcurrentAccess(t *testing.T) {
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()
	registry := NewServiceRegistry(grpcServer)

	t.Run("concurrent registration and retrieval", func(t *testing.T) {
		done := make(chan bool)

		// Concurrent registrations
		for i := 0; i < 10; i++ {
			go func(n int) {
				info := &ServiceInfo{
					Name:    "pandora.test.v1.TestService",
					Version: string(rune('a' + n)), // v1, v2, etc.
				}
				registry.RegisterService(info)
				done <- true
			}(i)
		}

		// Concurrent retrievals
		for i := 0; i < 10; i++ {
			go func() {
				registry.ListServices()
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 20; i++ {
			<-done
		}

		// Verify some services were registered
		services := registry.ListServices()
		assert.Greater(t, len(services), 0)
	})
}
