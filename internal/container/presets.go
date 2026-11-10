package container

import (
	"path/filepath"
	"sort"
	"strings"
)

type Preset struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Image          string         `json:"image"`
	SetupCommands  [][]string     `json:"setupCommands,omitempty"`
	DefaultLimits  ResourceLimits `json:"defaultLimits"`
	DetectionFiles []string       `json:"detectionFiles,omitempty"`
}

var builtInPresets = []Preset{
	{ID:"node-pnpm", Name:"Node.js + pnpm", Description:"Node LTS, Corepack and pnpm for React, Next.js and Node services.", Image:"docker.io/library/node:22-bookworm", SetupCommands:[][]string{{"corepack","enable"}}, DefaultLimits:ResourceLimits{MemoryBytes:2<<30,CPUCores:2,PidsLimit:512}, DetectionFiles:[]string{"pnpm-lock.yaml"}},
	{ID:"bun", Name:"Bun", Description:"Bun runtime and package manager for Bun-native projects.", Image:"docker.io/oven/bun:1", DefaultLimits:ResourceLimits{MemoryBytes:2<<30,CPUCores:2,PidsLimit:512}, DetectionFiles:[]string{"bun.lock","bun.lockb"}},
	{ID:"python-uv", Name:"Python + uv", Description:"Python 3.12 and uv for Python applications and tooling.", Image:"ghcr.io/astral-sh/uv:python3.12-bookworm", DefaultLimits:ResourceLimits{MemoryBytes:2<<30,CPUCores:2,PidsLimit:512}, DetectionFiles:[]string{"uv.lock","pyproject.toml"}},
	{ID:"node-npm", Name:"Node.js + npm", Description:"Node LTS and npm for standard JavaScript projects.", Image:"docker.io/library/node:22-bookworm", DefaultLimits:ResourceLimits{MemoryBytes:2<<30,CPUCores:2,PidsLimit:512}, DetectionFiles:[]string{"package-lock.json"}},
	{ID:"go", Name:"Go", Description:"Go toolchain for Go services and CLI applications.", Image:"docker.io/library/golang:1.23-bookworm", DefaultLimits:ResourceLimits{MemoryBytes:2<<30,CPUCores:2,PidsLimit:512}, DetectionFiles:[]string{"go.mod"}},
	{ID:"ubuntu-base", Name:"Ubuntu Base", Description:"Ubuntu LTS base for custom or mixed-language environments.", Image:"docker.io/library/ubuntu:24.04", DefaultLimits:ResourceLimits{MemoryBytes:2<<30,CPUCores:2,PidsLimit:512}},
}

func BuiltInPresets() []Preset {
	out := make([]Preset, len(builtInPresets))
	for i, preset := range builtInPresets {
		out[i] = preset
		out[i].SetupCommands = cloneCommands(preset.SetupCommands)
		out[i].DetectionFiles = append([]string(nil), preset.DetectionFiles...)
	}
	return out
}

func PresetByID(id string) (Preset, bool) {
	for _, preset := range builtInPresets {
		if preset.ID == id {
			copy := preset
			copy.SetupCommands = cloneCommands(preset.SetupCommands)
			copy.DetectionFiles = append([]string(nil), preset.DetectionFiles...)
			return copy, true
		}
	}
	return Preset{}, false
}

// DetectPreset recommends a runtime from repository marker files. Explicit user
// selection always wins over this recommendation.
func DetectPreset(paths []string) (Preset, bool) {
	found := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		found[strings.ToLower(filepath.Base(path))] = struct{}{}
	}
	for _, id := range []string{"bun","node-pnpm","python-uv","node-npm","go","ubuntu-base"} {
		preset, _ := PresetByID(id)
		for _, marker := range preset.DetectionFiles {
			if _, ok := found[strings.ToLower(marker)]; ok {
				return preset, true
			}
		}
	}
	return Preset{}, false
}

func AllowedPresetImages() []string {
	set := make(map[string]struct{}, len(builtInPresets))
	for _, preset := range builtInPresets { set[preset.Image] = struct{}{} }
	out := make([]string, 0, len(set))
	for image := range set { out = append(out, image) }
	sort.Strings(out)
	return out
}

func cloneCommands(in [][]string) [][]string {
	out := make([][]string, len(in))
	for i := range in { out[i] = append([]string(nil), in[i]...) }
	return out
}
