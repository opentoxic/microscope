package contract_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestOpenAPIPathsAreRegisteredInGoHandler(t *testing.T) {
	specPath := repoRoot(t) + "/core/api/openapi.yaml"
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read openapi: %v", err)
	}

	handlerPath := repoRoot(t) + "/adaptor/go/handler.go"
	handlerSource, err := os.ReadFile(handlerPath)
	if err != nil {
		t.Fatalf("read handler: %v", err)
	}

	for _, route := range parseOpenAPIRoutes(string(data)) {
		if !handlerImplementsRoute(string(handlerSource), route.method, route.path) {
			t.Errorf("openapi route %s %s is not registered in handler.go", route.method, route.path)
		}
	}
}

type openAPIRoute struct {
	method string
	path   string
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../"))
}

func handlerImplementsRoute(source, method, openAPIPath string) bool {
	apiPath := strings.TrimPrefix(openAPIPath, "/")
	needle := method + ` "+prefix+"/` + apiPath + `"`
	return strings.Contains(source, needle)
}

func parseOpenAPIRoutes(source string) []openAPIRoute {
	pathRe := regexp.MustCompile(`(?m)^  (/api/[^:]+):\s*$`)
	methodRe := regexp.MustCompile(`(?m)^    ([a-z]+):\s*$`)

	var routes []openAPIRoute
	lines := strings.Split(source, "\n")
	var currentPath string
	for _, line := range lines {
		if match := pathRe.FindStringSubmatch(line); len(match) == 2 {
			currentPath = match[1]
			continue
		}
		if currentPath == "" {
			continue
		}
		if match := methodRe.FindStringSubmatch(line); len(match) == 2 {
			routes = append(routes, openAPIRoute{
				method: strings.ToUpper(match[1]),
				path:   currentPath,
			})
			continue
		}
		if strings.HasPrefix(line, "  /") {
			currentPath = ""
		}
	}
	return routes
}
