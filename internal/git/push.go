package git

import (
  "context"
  "encoding/base64"
  "errors"
  "os"
  "os/exec"
  "strings"
  "time"

  "github.com/Mohamed-faaris/go-mcp/internal/credentials"
)

// PushWithGitHubApp performs the push on the host. The installation token is
// kept in host process memory/environment and is never mounted into a task container.
func PushWithGitHubApp(ctx context.Context,dir,owner,repository,branch string,broker *credentials.GitHubAppBroker)error{
  if broker==nil{return errors.New("GitHub App credential broker is not configured")}
  if strings.TrimSpace(branch)==""{return errors.New("branch is required")}
  token,expires,err:=broker.InstallationToken(ctx,owner,repository);if err!=nil{return err}
  if !expires.IsZero()&&time.Until(expires)<30*time.Second{return errors.New("GitHub installation token is too close to expiry")}
  header:="AUTHORIZATION: basic "+base64.StdEncoding.EncodeToString([]byte("x-access-token:"+token))
  cmd:=exec.CommandContext(ctx,"git","push","origin",branch)
  cmd.Dir=dir
  cmd.Env=append(os.Environ(),"GIT_CONFIG_COUNT=1","GIT_CONFIG_KEY_0=http.extraheader","GIT_CONFIG_VALUE_0="+header,"GIT_TERMINAL_PROMPT=0")
  output,err:=cmd.CombinedOutput()
  if err!=nil{
    // Do not include transport output, which may contain credential-related details.
    return errors.New("GitHub push failed; inspect host-side Git diagnostics (credentials redacted)")
  }
  _=output
  return nil
}
