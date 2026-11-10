package workspace

import (
  "context"
  "errors"
  "io"
  "os"
  "os/exec"
  "path/filepath"
  "strings"
)

var ignored=map[string]bool{".git":true,"node_modules":true,".next":true,"dist":true,"build":true,"target":true,".venv":true,"venv":true,"__pycache__":true,"vendor":true}

type Result struct { Path string; Remote string; BaseCommit string; Branch string }

func command(ctx context.Context,dir string,args ...string)(string,error){
  c:=exec.CommandContext(ctx,"git",args...);c.Dir=dir;out,err:=c.CombinedOutput();if err!=nil{return strings.TrimSpace(string(out)),err};return strings.TrimSpace(string(out)),nil
}

func Prepare(ctx context.Context,source,baseDir,taskID string)(Result,error){
  sourceAbs,err:=filepath.Abs(source);if err!=nil{return Result{},err}
  info,err:=os.Stat(sourceAbs);if err!=nil||!info.IsDir(){return Result{},errors.New("project path must be an existing directory")}
  if err=os.MkdirAll(baseDir,0700);err!=nil{return Result{},err}
  dest:=filepath.Join(baseDir,"task-"+taskID)
  if _,err=os.Stat(dest);err==nil{return Result{},errors.New("workspace already exists")}
  remote:=""
  if out,e:=command(ctx,sourceAbs,"remote","get-url","origin");e==nil{remote=out}
  base:=""
  if _,e:=command(ctx,sourceAbs,"rev-parse","--is-inside-work-tree");e==nil{
    if out,e:=command(ctx,sourceAbs,"rev-parse","HEAD");e==nil{base=out}
    if _,e=command(ctx,sourceAbs,"clone","--no-hardlinks",sourceAbs,dest);e!=nil{return Result{},errors.New("failed to clone local git repository: "+e.Error())}
    if remote!=""{_,_=command(ctx,dest,"remote","set-url","origin",remote)} else {_,_=command(ctx,dest,"remote","remove","origin")}
  }else{
    if err=os.MkdirAll(dest,0700);err!=nil{return Result{},err}
    if err=copyTree(sourceAbs,dest,sourceAbs);err!=nil{return Result{},err}
    if _,e:=command(ctx,dest,"init","-b","main");e!=nil{return Result{},errors.New("git init failed: "+e.Error())}
    _,_=command(ctx,dest,"config","user.name","go-mcp agent")
    _,_=command(ctx,dest,"config","user.email","agent@go-mcp.local")
    _,_=command(ctx,dest,"add","-A")
    _,_=command(ctx,dest,"commit","-m","chore: initialize task workspace")
    base,_=command(ctx,dest,"rev-parse","HEAD")
  }
  // Copy the current working tree over the clone, preserving uncommitted user changes.
  if err=copyTree(sourceAbs,dest,sourceAbs);err!=nil{return Result{},err}
  branch:="agent/"+taskID
  if _,err=command(ctx,dest,"checkout","-B",branch);err!=nil{return Result{},err}
  _,_=command(ctx,dest,"config","user.name","go-mcp agent")
  _,_=command(ctx,dest,"config","user.email","agent@go-mcp.local")
  return Result{Path:dest,Remote:remote,BaseCommit:base,Branch:branch},nil
}

func copyTree(src,dst,root string)error{
  entries,err:=os.ReadDir(src);if err!=nil{return err}
  for _,entry:=range entries{
    name:=entry.Name();if ignored[name]{continue}
    from:=filepath.Join(src,name);to:=filepath.Join(dst,name)
    fi,err:=os.Lstat(from);if err!=nil{return err}
    if fi.Mode()&os.ModeSymlink!=0{
      target,err:=filepath.EvalSymlinks(from);if err!=nil{return err}
      rel,err:=filepath.Rel(root,target);if err!=nil||rel==".."||strings.HasPrefix(rel,".."+string(filepath.Separator)){return errors.New("workspace source contains a symlink outside the project root")}
      link,err:=os.Readlink(from);if err!=nil{return err};_ = os.Remove(to);if err=os.Symlink(link,to);err!=nil{return err};continue
    }
    if fi.IsDir(){if err=os.MkdirAll(to,fi.Mode().Perm());err!=nil{return err};if err=copyTree(from,to,root);err!=nil{return err};continue}
    if !fi.Mode().IsRegular(){continue}
    in,err:=os.Open(from);if err!=nil{return err};out,err:=os.OpenFile(to,os.O_CREATE|os.O_TRUNC|os.O_WRONLY,fi.Mode().Perm());if err!=nil{in.Close();return err}
    _,copyErr:=io.Copy(out,in);in.Close();out.Close();if copyErr!=nil{return copyErr}
  }
  return nil
}
