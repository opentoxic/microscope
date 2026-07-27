package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed templates/*
var templates embed.FS

func main() {
	dir := flag.String("dir", ".", "target project directory")
	force := flag.Bool("force", false, "overwrite existing scaffold files")
	noEnv := flag.Bool("no-env", false, "skip .env.example updates")
	skipGet := flag.Bool("skip-get", false, "skip go get github.com/opentoxic/microscope")
	flag.Parse()

	root, err := filepath.Abs(*dir)
	if err != nil {
		fail(err)
	}

	if !*skipGet {
		cmd := exec.Command("go", "get", "github.com/opentoxic/microscope@latest")
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fail(fmt.Errorf("go get: %w", err))
		}
	}

	wrote := []string{}

	cfgPath := filepath.Join(root, "config", "microscope.yaml")
	if err := writeTemplate("templates/microscope.yaml.tmpl", cfgPath, *force); err != nil {
		fail(err)
	}
	wrote = append(wrote, cfgPath)

	wirePath := filepath.Join(root, "internal", "microscope", "wire.go")
	if err := writeTemplate("templates/wire.go.tmpl", wirePath, *force); err != nil {
		fail(err)
	}
	wrote = append(wrote, wirePath)

	if !*noEnv {
		envExample := filepath.Join(root, ".env.example")
		if err := appendSnippet(envExample, "templates/env.example.snippet"); err != nil {
			fail(err)
		}
		wrote = append(wrote, envExample)
	}

	fmt.Println("Microscope install complete.")
	fmt.Println()
	fmt.Println("Wrote:")
	for _, p := range wrote {
		fmt.Printf("  - %s\n", p)
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Set APP_ENV=development and DATABASE_URL in your environment")
	fmt.Println("  2. Call internal/microscope.Wire from your app bootstrap")
	fmt.Println("  3. Run the app and open http://localhost:8080/microscope")
	fmt.Println()
	fmt.Println("Docs: https://github.com/opentoxic/microscope/blob/main/docs/go-integration.md")
}

func writeTemplate(name, dest string, force bool) error {
	if _, err := os.Stat(dest); err == nil && !force {
		fmt.Printf("skip (exists): %s\n", dest)
		return nil
	}
	data, err := fs.ReadFile(templates, name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

func appendSnippet(dest, templateName string) error {
	snippet, err := fs.ReadFile(templates, templateName)
	if err != nil {
		return err
	}
	existing, err := os.ReadFile(dest)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(existing)
	if strings.Contains(content, "MICROSCOPE_ENABLED") {
		fmt.Printf("skip (already present): %s\n", dest)
		return nil
	}
	merged := strings.TrimRight(content, "\n") + string(snippet)
	return os.WriteFile(dest, []byte(merged), 0o644)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "microscope install: %v\n", err)
	os.Exit(1)
}
