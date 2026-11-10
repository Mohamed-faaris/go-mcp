package git

import (
  "context"
  "errors"
  "net/url"
  "os/exec"
  "path/filepath"
  "regexp"
  "strings"
)

type Remote struct { Name string `json:"name"`; URL string `json:"url"`; Provider string `json:"provider"`; Owner string `json:"owner,omitempty"`; Repository string `json:"repository,omitempty"` }

var githubPath=regexp.MustCompile(`^(?:https://github\.com/|ssh://git@github\.com/|git@github\.com:)([^/]+)/([^/]+?)(?:\.git)?/?$`)

func ReadRemotes(ctx context.Context,dir string)([]Remote,error){
  abs,err:=filepath.Abs(dir);if err!=nil{return nil,err}
  cmd:=exec.CommandContext(ctx,"git","-C",abs,"remote","-v")
  out,err:=cmd.Output()
  if err!=nil {if _,e:=exec.LookPath("git");e!=nil{return nil,e};return []Remote{},nil}
  found:=map[string]Remote{}
  for _,line:=range strings.Split(string(out),"\n"){
    fields:=strings.Fields(line);if len(fields)<3||fields[2]!="(fetch)"{continue}
    r:=Remote{Name:fields[0],URL:fields[1],Provider:"unknown"}
    if m:=githubPath.FindStringSubmatch(r.URL);len(m)==3{r.Provider="github";r.Owner=m[1];r.Repository=strings.TrimSuffix(m[2],".git")} else if u,e:=url.Parse(r.URL);e==nil {switch strings.ToLower(u.Hostname()){case "gitlab.com":r.Provider="gitlab";case "bitbucket.org":r.Provider="bitbucket";default:r.Provider="unknown"}}
    found[r.Name]=r
  }
  result:=make([]Remote,0,len(found));for _,v:=range found{result=append(result,v)};return result,nil
}

func IsGitRepository(dir string) bool { cmd:=exec.Command("git","-C",dir,"rev-parse","--is-inside-work-tree");return cmd.Run()==nil }
func HeadCommit(ctx context.Context,dir string)(string,error){out,err:=exec.CommandContext(ctx,"git","-C",dir,"rev-parse","HEAD").Output();if err!=nil{return "",err};return strings.TrimSpace(string(out)),nil}
func Branch(ctx context.Context,dir string)(string,error){out,err:=exec.CommandContext(ctx,"git","-C",dir,"branch","--show-current").Output();return strings.TrimSpace(string(out)),err}
func ValidateGitHubRemote(r Remote)error{if r.Provider!="github"||r.Owner==""||r.Repository==""{return errors.New("remote is not a recognized GitHub repository")};return nil}
