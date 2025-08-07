package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type SyftOutput struct {
	Artifacts []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		PURL    string `json:"purl"`
	} `json:"artifacts"`
}

type Dependency struct {
	GroupID    string
	ArtifactID string
	Version    string
	Scope      string // "compile" or "test"
	Key        string // group:artifact to deduplicate
}

func runSyftJar(path string) (Dependency, error) {
	cmd := exec.Command("syft", path, "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return Dependency{}, fmt.Errorf("syft failed on %s: %v", path, err)
	}

	var result SyftOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return Dependency{}, fmt.Errorf("invalid syft output: %v", err)
	}

	if len(result.Artifacts) == 0 {
		return Dependency{}, fmt.Errorf("no metadata found in: %s", path)
	}

	a := result.Artifacts[0]
	groupID := "unknown.group"
	if strings.HasPrefix(a.PURL, "pkg:maven/") {
		parts := strings.Split(a.PURL, "/")
		if len(parts) >= 2 {
			groupID = parts[1]
		}
	}

	scope := "compile"
	if strings.Contains(path, "/test/") || strings.Contains(strings.ToLower(filepath.Base(path)), "test-") {
		scope = "test"
	}

	dep := Dependency{
		GroupID:    groupID,
		ArtifactID: a.Name,
		Version:    a.Version,
		Scope:      scope,
		Key:        groupID + ":" + a.Name,
	}

	return dep, nil
}

func findJarFiles(root string) ([]string, error) {
	var jars []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".jar") {
			jars = append(jars, path)
		}
		return nil
	})
	return jars, err
}

func generatePom(dependencies []Dependency) string {
	builder := strings.Builder{}
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0
         http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>

  <groupId>generated</groupId>
  <artifactId>sbom-project</artifactId>
  <version>1.0.0</version>

  <dependencies>
`)

	for _, dep := range dependencies {
		builder.WriteString(fmt.Sprintf("    <dependency>\n"))
		builder.WriteString(fmt.Sprintf("      <groupId>%s</groupId>\n", dep.GroupID))
		builder.WriteString(fmt.Sprintf("      <artifactId>%s</artifactId>\n", dep.ArtifactID))
		builder.WriteString(fmt.Sprintf("      <version>%s</version>\n", dep.Version))
		if dep.Scope == "test" {
			builder.WriteString("      <scope>test</scope>\n")
		}
		builder.WriteString("    </dependency>\n")
	}

	builder.WriteString(`  </dependencies>
</project>
`)
	return builder.String()
}

func main() {
	projectDir := "."

	jarFiles, err := findJarFiles(projectDir)
	if err != nil {
		fmt.Println("Error finding jars:", err)
		return
	}

	seen := make(map[string]bool)
	var dependencies []Dependency

	for _, jar := range jarFiles {
		dep, err := runSyftJar(jar)
		if err != nil {
			fmt.Printf("Skipping %s: %v\n", jar, err)
			continue
		}
		if !seen[dep.Key] {
			dependencies = append(dependencies, dep)
			seen[dep.Key] = true
		}
	}

	if len(dependencies) == 0 {
		fmt.Println("No valid dependencies found.")
		return
	}

	pomContent := generatePom(dependencies)

	err = os.WriteFile("generated-pom.xml", []byte(pomContent), 0644)
	if err != nil {
		fmt.Println("Failed to write generated-pom.xml:", err)
		return
	}

	// Rename to pom.xml
	err = os.Rename("generated-pom.xml", "pom.xml")
	if err != nil {
		fmt.Println("Failed to rename generated-pom.xml to pom.xml:", err)
		return
	}

	fmt.Println("✅ pom.xml has been successfully generated from .jar files.")
}
