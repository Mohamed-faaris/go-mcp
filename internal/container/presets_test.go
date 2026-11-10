package container

import "testing"

func TestBuiltInPresetsHaveUniqueIDsAndImages(t *testing.T){
	seen:=map[string]bool{}
	for _,p:=range BuiltInPresets(){if p.ID==""||p.Name==""||p.Image==""{t.Fatalf("incomplete preset: %#v",p)};if seen[p.ID]{t.Fatalf("duplicate preset %q",p.ID)};seen[p.ID]=true}
	if len(seen)<6{t.Fatalf("expected six presets, got %d",len(seen))}
}
func TestDetectPresetPrioritizesBunLock(t *testing.T){p,ok:=DetectPreset([]string{"package.json","pnpm-lock.yaml","bun.lock"});if !ok||p.ID!="bun"{t.Fatalf("expected bun preset, got %q",p.ID)}}
func TestDetectPresetRecognizesUV(t *testing.T){p,ok:=DetectPreset([]string{"pyproject.toml","uv.lock"});if !ok||p.ID!="python-uv"{t.Fatalf("expected python-uv, got %q",p.ID)}}
func TestDetectPresetDoesNotGuess(t *testing.T){if _,ok:=DetectPreset([]string{"README.md"});ok{t.Fatal("should not guess from README")}}
