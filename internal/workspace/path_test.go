package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAcceptsDirectoryBelowRoot(t *testing.T){root:=t.TempDir();child:=filepath.Join(root,"task");if err:=os.Mkdir(child,0o700);err!=nil{t.Fatal(err)};got,err:=Validate(root,child);if err!=nil{t.Fatal(err)};if got!=child{t.Fatalf("got %q want %q",got,child)}}
func TestValidateRejectsRootItself(t *testing.T){root:=t.TempDir();if _,err:=Validate(root,root);!errors.Is(err,ErrOutsideRoot){t.Fatalf("expected ErrOutsideRoot, got %v",err)}}
func TestValidateRejectsSibling(t *testing.T){root:=t.TempDir();sibling:=t.TempDir();if _,err:=Validate(root,sibling);!errors.Is(err,ErrOutsideRoot){t.Fatalf("expected ErrOutsideRoot, got %v",err)}}
func TestValidateRejectsSymlinkEscape(t *testing.T){root:=t.TempDir();outside:=t.TempDir();link:=filepath.Join(root,"escape");if err:=os.Symlink(outside,link);err!=nil{t.Skip(err)};if _,err:=Validate(root,link);!errors.Is(err,ErrOutsideRoot){t.Fatalf("expected ErrOutsideRoot, got %v",err)}}
