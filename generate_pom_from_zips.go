package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Project struct {
	XMLName     xml.Name     `xml:"project"`
	Xmlns       string       `xml:"xmlns,attr"`
	ModelVer    string       `xml:"modelVersion"`
	GroupID     string       `xml:"groupId"`
	ArtifactID  string       `xml:"artifactId"`
	Version     string       `xml:"version"`
	Dependencies []Dependency `xml:"dependencies>dependency"`
}

type Dependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	SystemPath string `xml:"systemPath"`
}

func isTestZip(path string) bool {
	// Check if "test" is in any parent folder OR filename starts with "test-"
	lowerPath := strings.ToLower(path)
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(lowerPath, "/test/") || strings.HasPrefix(base, "test-")
}

func walkAndCollectZips(root string) ([]Dependency, error) {
	var deps []Dependency
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".zip") {
			return nil
		}
		scope := "compile"
		if isTestZip(path) {
			scope = "test"
		}
		artifact := strings.TrimSuffix(info.Name(), ".zip")
		dep := Dependency{
			GroupID:    "local.generated",
			ArtifactID: artifact,
			Version:    "1.0",
			Scope:      scope,
			SystemPath: path,
		}
		deps = append(deps, dep)
		return nil
	})
	return deps, err
}

func generatePomFile(deps []Dependency, output string) error {
	project := Project{
		Xmlns:       "http://maven.apache.org/POM/4.0.0",
		ModelVer:    "4.0.0",
		GroupID:     "org.example",
		ArtifactID:  "zip-to-pom-project",
		Version:     "1.0.0",
		Dependencies: deps,
	}

	xmlData, err := xml.MarshalIndent(project, "", "  ")
	if err != nil {
		return err
	}

	// Add header manually
	header := `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	content := header + string(xmlData)

	return os.WriteFile(output, []byte(content), 0644)
}

func main() {
	rootDir := "." // change to desired root
	outputFile := "generated-pom.xml"

	fmt.Println("Scanning for .zip files in:", rootDir)

	deps, err := walkAndCollectZips(rootDir)
	if err != nil {
		fmt.Println("Error during scan:", err)
		return
	}

	if len(deps) == 0 {
		fmt.Println("No .zip files found.")
		return
	}

	err = generatePomFile(deps, outputFile)
	if err != nil {
		fmt.Println("Error writing pom.xml:", err)
		return
	}

	fmt.Printf("✅ pom.xml generated successfully: %s with %d dependencies\n", outputFile, len(deps))
}
