package presets

import (
  "os"
  "path/filepath"
)

type Preset struct {
  ID string `json:"id"`
  Name string `json:"name"`
  Description string `json:"description"`
  Image string `json:"image"`
  SetupCommand string `json:"setupCommand"`
  DefaultMemory string `json:"defaultMemory"`
  DefaultCPUs string `json:"defaultCpus"`
  PackageManager string `json:"packageManager"`
}

var all = []Preset{
  {ID:"node-pnpm",Name:"Node.js + pnpm",Description:"Node LTS with Corepack and pnpm.",Image:"docker.io/library/node:22-bookworm",SetupCommand:"corepack enable && corepack prepare pnpm@latest --activate && git --version",DefaultMemory:"3g",DefaultCPUs:"2",PackageManager:"pnpm"},
  {ID:"bun",Name:"Bun",Description:"Bun runtime and package manager.",Image:"docker.io/oven/bun:1",SetupCommand:"bun --version && git --version",DefaultMemory:"3g",DefaultCPUs:"2",PackageManager:"bun"},
  {ID:"python-uv",Name:"Python + uv",Description:"Python 3.12 with uv package management.",Image:"ghcr.io/astral-sh/uv:python3.12-bookworm",SetupCommand:"uv --version && python --version && git --version",DefaultMemory:"3g",DefaultCPUs:"2",PackageManager:"uv"},
  {ID:"node-npm",Name:"Node.js + npm",Description:"Node LTS with npm.",Image:"docker.io/library/node:22-bookworm",SetupCommand:"node --version && npm --version && git --version",DefaultMemory:"3g",DefaultCPUs:"2",PackageManager:"npm"},
  {ID:"go",Name:"Go",Description:"Go toolchain for services and CLI projects.",Image:"docker.io/library/golang:1.23-bookworm",SetupCommand:"go version && git --version",DefaultMemory:"2g",DefaultCPUs:"2",PackageManager:"go"},
  {ID:"ubuntu-base",Name:"Ubuntu Base",Description:"Minimal Ubuntu environment for custom projects.",Image:"docker.io/library/ubuntu:24.04",SetupCommand:"apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y git ca-certificates curl",DefaultMemory:"2g",DefaultCPUs:"2",PackageManager:""},
}

func List() []Preset { out:=make([]Preset,len(all)); copy(out,all); return out }
func Get(id string) (Preset,bool) { for _,p:=range all { if p.ID==id{return p,true} }; return Preset{},false }

func Detect(dir string) (Preset,bool) {
  exists:=func(name string)bool{ _,err:=os.Stat(filepath.Join(dir,name)); return err==nil }
  switch {
  case exists("pnpm-lock.yaml"): p,_:=Get("node-pnpm"); return p,true
  case exists("bun.lock") || exists("bun.lockb"): p,_:=Get("bun"); return p,true
  case exists("uv.lock"): p,_:=Get("python-uv"); return p,true
  case exists("go.mod"): p,_:=Get("go"); return p,true
  case exists("pyproject.toml"): p,_:=Get("python-uv"); return p,true
  case exists("package-lock.json"): p,_:=Get("node-npm"); return p,true
  case exists("package.json"): p,_:=Get("node-pnpm"); return p,true
  default: p,_:=Get("ubuntu-base"); return p,false
  }
}
