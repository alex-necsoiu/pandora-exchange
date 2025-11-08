package ci_checks_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDomainLayerImportBoundaries ensures the domain layer maintains clean architecture
// by preventing it from importing infrastructure or transport layer packages.
//
// Domain layer should only depend on:
//   - Standard library packages
//   - External/third-party packages
//   - Other domain packages
//
// Domain layer should NOT import:
//   - internal/postgres (infrastructure/data layer)
//   - internal/transport (HTTP/gRPC handlers)
//   - internal/repository (concrete implementations)
//   - internal/service (concrete implementations)
//
// This test ensures compile-time enforcement of architectural boundaries.
func TestDomainLayerImportBoundaries(t *testing.T) {
	// Get the project root (2 levels up from internal/ci_checks)
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	domainPath := filepath.Join(projectRoot, "internal", "domain")

	// Forbidden imports for domain layer
	forbiddenImports := []string{
		"github.com/alex-necsoiu/pandora-exchange/internal/postgres",
		"github.com/alex-necsoiu/pandora-exchange/internal/transport",
		"github.com/alex-necsoiu/pandora-exchange/internal/repository",
		"github.com/alex-necsoiu/pandora-exchange/internal/service",
	}

	// Walk through all .go files in domain package
	err = filepath.Walk(domainPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files (they may need mocks from other packages)
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the Go file
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("Failed to parse %s: %v", path, err)
			return nil
		}

		// Check all imports in this file
		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			// Check if this import is forbidden
			for _, forbidden := range forbiddenImports {
				if importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/") {
					relPath, _ := filepath.Rel(projectRoot, path)
					t.Errorf(
						"ARCHITECTURE VIOLATION: Domain layer file '%s' imports forbidden package '%s'\n"+
							"Domain layer must not depend on infrastructure or transport layers.\n"+
							"Consider:\n"+
							"  1. Moving shared types to domain layer\n"+
							"  2. Using interfaces defined in domain\n"+
							"  3. Inverting the dependency (dependency inversion principle)",
						relPath,
						importPath,
					)
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Error walking domain directory: %v", err)
	}
}

// TestRepositoryLayerImportBoundaries ensures repository implementations
// don't import transport layer (HTTP/gRPC handlers).
//
// Repository layer can import:
//   - internal/domain (interfaces and types)
//   - internal/postgres (database layer)
//   - Standard library
//   - External packages
//
// Repository layer should NOT import:
//   - internal/transport (HTTP/gRPC handlers)
func TestRepositoryLayerImportBoundaries(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	repositoryPath := filepath.Join(projectRoot, "internal", "repository")

	// Check if repository directory exists
	if _, err := os.Stat(repositoryPath); os.IsNotExist(err) {
		t.Skip("Repository directory does not exist yet")
		return
	}

	forbiddenImports := []string{
		"github.com/alex-necsoiu/pandora-exchange/internal/transport",
	}

	err = filepath.Walk(repositoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("Failed to parse %s: %v", path, err)
			return nil
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			for _, forbidden := range forbiddenImports {
				if importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/") {
					relPath, _ := filepath.Rel(projectRoot, path)
					t.Errorf(
						"ARCHITECTURE VIOLATION: Repository layer file '%s' imports forbidden package '%s'\n"+
							"Repository layer must not depend on transport layer.",
						relPath,
						importPath,
					)
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Error walking repository directory: %v", err)
	}
}

// TestServiceLayerImportBoundaries ensures service implementations
// don't import transport layer (HTTP/gRPC handlers).
//
// Service layer can import:
//   - internal/domain (interfaces and types)
//   - internal/repository (for dependency injection)
//   - Standard library
//   - External packages
//
// Service layer should NOT import:
//   - internal/transport (HTTP/gRPC handlers)
//   - internal/postgres directly (should use repository interfaces)
func TestServiceLayerImportBoundaries(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	servicePath := filepath.Join(projectRoot, "internal", "service")

	// Check if service directory exists
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		t.Skip("Service directory does not exist yet")
		return
	}

	forbiddenImports := []string{
		"github.com/alex-necsoiu/pandora-exchange/internal/transport",
		"github.com/alex-necsoiu/pandora-exchange/internal/postgres",
	}

	err = filepath.Walk(servicePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("Failed to parse %s: %v", path, err)
			return nil
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			for _, forbidden := range forbiddenImports {
				if importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/") {
					relPath, _ := filepath.Rel(projectRoot, path)
					t.Errorf(
						"ARCHITECTURE VIOLATION: Service layer file '%s' imports forbidden package '%s'\n"+
							"Service layer must not depend on transport or postgres layers directly.",
						relPath,
						importPath,
					)
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Error walking service directory: %v", err)
	}
}
