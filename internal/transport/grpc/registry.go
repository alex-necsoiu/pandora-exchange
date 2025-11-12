package grpc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var (
	// ErrServiceAlreadyRegistered indicates a service is already registered
	ErrServiceAlreadyRegistered = errors.New("service already registered")

	// ErrServiceNotFound indicates the requested service was not found
	ErrServiceNotFound = errors.New("service not found")

	// ErrInvalidServiceName indicates an invalid service name
	ErrInvalidServiceName = errors.New("invalid service name")

	// ErrInvalidVersion indicates an invalid version format
	ErrInvalidVersion = errors.New("invalid version format")

	// ErrBreakingChange indicates a backward incompatible change
	ErrBreakingChange = errors.New("breaking change detected")
)

// ServiceInfo contains metadata about a registered gRPC service
type ServiceInfo struct {
	Name        string            // Fully qualified service name (e.g., "pandora.user.v1.UserService")
	Version     string            // Semantic version (e.g., "v1", "v2")
	Description string            // Service description
	Methods     []string          // List of RPC method names
	ProtoFile   string            // Path to .proto file
	Dependencies []string         // List of dependent service names
	Metadata    map[string]string // Additional metadata
}

// ServiceRegistry manages gRPC service registration and discovery
type ServiceRegistry struct {
	mu            sync.RWMutex
	services      map[string]*ServiceInfo // serviceName -> ServiceInfo
	versions      map[string][]string     // serviceName -> list of versions
	grpcServer    *grpc.Server
	healthServer  *health.Server
	enableReflection bool
}

// RegistryOption is a functional option for configuring ServiceRegistry
type RegistryOption func(*ServiceRegistry)

// WithReflection enables gRPC server reflection
func WithReflection(enable bool) RegistryOption {
	return func(r *ServiceRegistry) {
		r.enableReflection = enable
	}
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(grpcServer *grpc.Server, opts ...RegistryOption) *ServiceRegistry {
	registry := &ServiceRegistry{
		services:         make(map[string]*ServiceInfo),
		versions:         make(map[string][]string),
		grpcServer:       grpcServer,
		healthServer:     health.NewServer(),
		enableReflection: true, // Default: enabled for development
	}

	// Apply options
	for _, opt := range opts {
		opt(registry)
	}

	// Register health check service
	grpc_health_v1.RegisterHealthServer(grpcServer, registry.healthServer)

	// Enable reflection if configured
	if registry.enableReflection {
		reflection.Register(grpcServer)
	}

	return registry
}

// RegisterService registers a gRPC service with metadata
func (r *ServiceRegistry) RegisterService(info *ServiceInfo) error {
	if info.Name == "" {
		return ErrInvalidServiceName
	}

	if info.Version == "" {
		return ErrInvalidVersion
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if this exact service version is already registered
	fullName := fmt.Sprintf("%s.%s", info.Name, info.Version)
	if _, exists := r.services[fullName]; exists {
		return fmt.Errorf("%w: %s", ErrServiceAlreadyRegistered, fullName)
	}

	// Store service info
	r.services[fullName] = info

	// Track versions for this service
	baseName := extractBaseName(info.Name)
	r.versions[baseName] = append(r.versions[baseName], info.Version)

	// Set service as healthy
	r.healthServer.SetServingStatus(info.Name, grpc_health_v1.HealthCheckResponse_SERVING)

	return nil
}

// GetService retrieves service information by name and version
func (r *ServiceRegistry) GetService(name, version string) (*ServiceInfo, error) {
	if name == "" {
		return nil, ErrInvalidServiceName
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	fullName := fmt.Sprintf("%s.%s", name, version)
	info, exists := r.services[fullName]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, fullName)
	}

	// Return a copy to prevent external modification
	infoCopy := *info
	return &infoCopy, nil
}

// ListServices returns all registered services
func (r *ServiceRegistry) ListServices() []*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]*ServiceInfo, 0, len(r.services))
	for _, info := range r.services {
		infoCopy := *info
		services = append(services, &infoCopy)
	}

	return services
}

// GetVersions returns all registered versions for a service
func (r *ServiceRegistry) GetVersions(serviceName string) ([]string, error) {
	if serviceName == "" {
		return nil, ErrInvalidServiceName
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	versions, exists := r.versions[serviceName]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, serviceName)
	}

	// Return a copy
	versionsCopy := make([]string, len(versions))
	copy(versionsCopy, versions)

	return versionsCopy, nil
}

// GetLatestVersion returns the latest version of a service
func (r *ServiceRegistry) GetLatestVersion(serviceName string) (string, error) {
	versions, err := r.GetVersions(serviceName)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("%w: %s", ErrServiceNotFound, serviceName)
	}

	// Return the last version (assumes versions are added in order)
	return versions[len(versions)-1], nil
}

// SetServiceHealth updates the health status of a service
func (r *ServiceRegistry) SetServiceHealth(serviceName string, serving bool) {
	var status grpc_health_v1.HealthCheckResponse_ServingStatus
	if serving {
		status = grpc_health_v1.HealthCheckResponse_SERVING
	} else {
		status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	r.healthServer.SetServingStatus(serviceName, status)
}

// Shutdown gracefully shuts down all services
func (r *ServiceRegistry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Mark all services as not serving
	for serviceName := range r.services {
		r.healthServer.SetServingStatus(serviceName, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}

	// Graceful stop with context
	stopped := make(chan struct{})
	go func() {
		r.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		// Force stop if context times out
		r.grpcServer.Stop()
		return ctx.Err()
	case <-stopped:
		return nil
	}
}

// extractBaseName extracts the base service name without version
// e.g., "pandora.user.v1.UserService" -> "UserService"
func extractBaseName(fullName string) string {
	// Simple implementation - can be enhanced with proper parsing
	return fullName
}

// CompatibilityChecker checks backward compatibility between service versions
type CompatibilityChecker struct {
	mu     sync.RWMutex
	checks map[string]*CompatibilityResult // version pair -> result
}

// CompatibilityResult represents the result of a compatibility check
type CompatibilityResult struct {
	OldVersion      string
	NewVersion      string
	IsCompatible    bool
	BreakingChanges []string
	Warnings        []string
	CheckedAt       string // Timestamp
}

// NewCompatibilityChecker creates a new compatibility checker
func NewCompatibilityChecker() *CompatibilityChecker {
	return &CompatibilityChecker{
		checks: make(map[string]*CompatibilityResult),
	}
}

// CheckCompatibility checks if newInfo is backward compatible with oldInfo
func (c *CompatibilityChecker) CheckCompatibility(oldInfo, newInfo *ServiceInfo) (*CompatibilityResult, error) {
	result := &CompatibilityResult{
		OldVersion:      oldInfo.Version,
		NewVersion:      newInfo.Version,
		IsCompatible:    true,
		BreakingChanges: []string{},
		Warnings:        []string{},
	}

	// Check 1: Method removal is a breaking change
	oldMethods := make(map[string]bool)
	for _, method := range oldInfo.Methods {
		oldMethods[method] = true
	}

	for _, method := range oldInfo.Methods {
		found := false
		for _, newMethod := range newInfo.Methods {
			if method == newMethod {
				found = true
				break
			}
		}
		if !found {
			result.IsCompatible = false
			result.BreakingChanges = append(result.BreakingChanges, 
				fmt.Sprintf("Method '%s' removed", method))
		}
	}

	// Check 2: New methods are OK (non-breaking)
	for _, newMethod := range newInfo.Methods {
		if !oldMethods[newMethod] {
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("New method '%s' added", newMethod))
		}
	}

	// Check 3: Service name should match (excluding version)
	if oldInfo.Name != newInfo.Name {
		result.IsCompatible = false
		result.BreakingChanges = append(result.BreakingChanges, 
			fmt.Sprintf("Service name changed from '%s' to '%s'", oldInfo.Name, newInfo.Name))
	}

	// Store result
	key := fmt.Sprintf("%s->%s", oldInfo.Version, newInfo.Version)
	c.mu.Lock()
	c.checks[key] = result
	c.mu.Unlock()

	if !result.IsCompatible {
		return result, ErrBreakingChange
	}

	return result, nil
}

// GetCheckResult retrieves a previously stored compatibility check result
func (c *CompatibilityChecker) GetCheckResult(oldVersion, newVersion string) (*CompatibilityResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("%s->%s", oldVersion, newVersion)
	result, exists := c.checks[key]
	if !exists {
		return nil, fmt.Errorf("no compatibility check found for %s", key)
	}

	// Return a copy
	resultCopy := *result
	return &resultCopy, nil
}
