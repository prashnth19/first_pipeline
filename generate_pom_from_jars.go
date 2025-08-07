package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type SyftDocument struct {
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
	Scope      string
}

func main() {
	projectDir := "./" // change or pass via os.Args[1]

	var jarFiles []string
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".jar") {
			jarFiles = append(jarFiles, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	dependencies := make([]Dependency, 0)

	for _, jar := range jarFiles {
		// Run syft -o json <jar>
		cmd := exec.Command("syft", jar, "-o", "json")
		output, err := cmd.Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to scan %s: %v\n", jar, err)
			continue
		}

		var doc SyftDocument
		err = json.Unmarshal(output, &doc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse syft output: %v\n", err)
			continue
		}

		for _, art := range doc.Artifacts {
			if art.PURL == "" {
				continue
			}

			groupID, artifactID, version := parsePurl(art.PURL)
			if groupID == "" || artifactID == "" || version == "" {
				continue
			}

			scope := "compile"
			lower := strings.ToLower(jar)
			if strings.Contains(lower, "/test/") || strings.Contains(lower, "\\test\\") || strings.Contains(lower, "test-") {
				scope = "test"
			}

			dep := Dependency{
				GroupID:    groupID,
				ArtifactID: artifactID,
				Version:    version,
				Scope:      scope,
			}
			dependencies = append(dependencies, dep)
		}
	}

	generatePom(dependencies)
}

func parsePurl(purl string) (string, string, string) {
	// Example: pkg:maven/org.apache.commons/commons-lang3@3.12.0
	if !strings.HasPrefix(purl, "pkg:maven/") {
		return "", "", ""
	}
	purl = strings.TrimPrefix(purl, "pkg:maven/")
	parts := strings.SplitN(purl, "@", 2)
	if len(parts) != 2 {
		return "", "", ""
	}
	coords := strings.Split(parts[0], "/")
	if len(coords) != 2 {
		return "", "", ""
	}
	return coords[0], coords[1], parts[1]
}

func generatePom(deps []Dependency) {
	f, err := os.Create("generated-pom.xml")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.WriteString("<project xmlns=\"http://maven.apache.org/POM/4.0.0\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\"\n")
	f.WriteString("         xsi:schemaLocation=\"http://maven.apache.org/POM/4.0.0\n")
	f.WriteString("                             http://maven.apache.org/xsd/maven-4.0.0.xsd\">\n")
	f.WriteString("  <modelVersion>4.0.0</modelVersion>\n")
	f.WriteString("  <groupId>generated</groupId>\n")
	f.WriteString("  <artifactId>sbom-artifact</artifactId>\n")
	f.WriteString("  <version>1.0.0</version>\n")
	f.WriteString("  <dependencies>\n")

	for _, d := range deps {
		f.WriteString("    <dependency>\n")
		f.WriteString(fmt.Sprintf("      <groupId>%s</groupId>\n", d.GroupID))
		f.WriteString(fmt.Sprintf("      <artifactId>%s</artifactId>\n", d.ArtifactID))
		f.WriteString(fmt.Sprintf("      <version>%s</version>\n", d.Version))
		if d.Scope != "compile" {
			f.WriteString(fmt.Sprintf("      <scope>%s</scope>\n", d.Scope))
		}
		f.WriteString("    </dependency>\n")
	}

	f.WriteString("  </dependencies>\n")
	f.WriteString("</project>\n")

	fmt.Println("✅ generated-pom.xml created with", len(deps), "dependencies.")
}
