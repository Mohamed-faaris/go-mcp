package podman

import (
  "bufio"
  "context"
  "encoding/binary"
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "net/url"
  "regexp"
  "strings"
  "time"
)

type CommandResult struct {
  Stdout string `json:"stdout"`
  Stderr string `json:"stderr"`
  ExitCode int `json:"exitCode"`
}

var exitMarker = regexp.MustCompile(`(?m)^__GOMCP_EXIT__([0-9]+)\r?$`)

// RunCommand executes a non-interactive command in a container using the
// Podman exec API. It never exposes the Podman socket to the workload.
func (c *Client) RunCommand(ctx context.Context, containerID string, command string, timeout time.Duration) (CommandResult,error) {
  if timeout<=0 || timeout>10*time.Minute { timeout=2*time.Minute }
  ctx,cancel:=context.WithTimeout(ctx,timeout);defer cancel()
  wrapped := "set +e\n{\n" + command + "\n}\n__gomcp_rc=$?\nprintf '\\n__GOMCP_EXIT__%s\\n' \"$__gomcp_rc\"\nexit 0"
  createPayload:=map[string]interface{}{
    "AttachStdout":true,"AttachStderr":true,"AttachStdin":false,"Tty":false,
    "Cmd":[]string{"/bin/sh","-lc",wrapped},
    "WorkingDir":"/workspace",
  }
  raw,err:=json.Marshal(createPayload);if err!=nil{return CommandResult{},err}
  createURL:=fmt.Sprintf("%s/containers/%s/exec",c.baseURL,url.PathEscape(containerID))
  req,err:=http.NewRequestWithContext(ctx,http.MethodPost,createURL,strings.NewReader(string(raw)));if err!=nil{return CommandResult{},err}
  req.Header.Set("Content-Type","application/json")
  resp,err:=c.httpClient.Do(req);if err!=nil{return CommandResult{},err}
  var created ExecCreateResponse
  if resp.StatusCode!=http.StatusCreated&&resp.StatusCode!=http.StatusOK { body,_:=io.ReadAll(io.LimitReader(resp.Body,8192));resp.Body.Close();return CommandResult{},fmt.Errorf("exec create failed (%d): %s",resp.StatusCode,string(body)) }
  err=json.NewDecoder(resp.Body).Decode(&created);resp.Body.Close();if err!=nil{return CommandResult{},err}
  if created.ID==""{return CommandResult{},fmt.Errorf("Podman exec returned an empty session id")}

  conn,err:=c.DialUnixConn();if err!=nil{return CommandResult{},err}
  defer conn.Close()
  if deadline,ok:=ctx.Deadline();ok{_ = conn.SetDeadline(deadline)}
  stopClose:=context.AfterFunc(ctx,func(){_ = conn.Close()});defer stopClose()
  startPayload:=`{"Detach":false,"Tty":false}`
  path:=fmt.Sprintf("/v4.0.0/libpod/exec/%s/start",url.PathEscape(created.ID))
  header:=fmt.Sprintf("POST %s HTTP/1.1\r\nHost: d\r\nContent-Type: application/json\r\nConnection: Upgrade\r\nUpgrade: tcp\r\nContent-Length: %d\r\n\r\n%s",path,len(startPayload),startPayload)
  if _,err=io.WriteString(conn,header);err!=nil{return CommandResult{},err}
  reader:=bufio.NewReader(conn)
  statusLine,err:=reader.ReadString('\n');if err!=nil{return CommandResult{},fmt.Errorf("read exec response: %w",err)}
  if !strings.Contains(statusLine,"101")&&!strings.Contains(statusLine,"200"){return CommandResult{},fmt.Errorf("exec start rejected: %s",strings.TrimSpace(statusLine))}
  for { line,e:=reader.ReadString('\n');if e!=nil{return CommandResult{},e};if strings.TrimSpace(line)==""{break} }

  result:=CommandResult{}
  first,peekErr:=reader.Peek(8)
  if peekErr==nil && (first[0]==1||first[0]==2||first[0]==3) && first[1]==0&&first[2]==0&&first[3]==0 {
    for {
      var h [8]byte
      if _,err=io.ReadFull(reader,h[:]);err!=nil{if err==io.EOF||err==io.ErrUnexpectedEOF{break};return result,err}
      n:=binary.BigEndian.Uint32(h[4:])
      if n>8*1024*1024{return result,fmt.Errorf("exec output frame exceeds 8 MiB")}
      chunk:=make([]byte,n);if _,err=io.ReadFull(reader,chunk);err!=nil{return result,err}
      if h[0]==2{result.Stderr+=string(chunk)}else{result.Stdout+=string(chunk)}
      if len(result.Stdout)+len(result.Stderr)>8*1024*1024{return result,fmt.Errorf("exec output exceeds 8 MiB")}
    }
  } else {
    b,e:=io.ReadAll(io.LimitReader(reader,8*1024*1024));result.Stdout=string(b);if e!=nil&&ctx.Err()==nil{return result,e}
  }
  combined:=result.Stdout+"\n"+result.Stderr
  matches:=exitMarker.FindStringSubmatch(combined)
  if len(matches)==2 { fmt.Sscanf(matches[1],"%d",&result.ExitCode); result.Stdout=exitMarker.ReplaceAllString(result.Stdout,"");result.Stderr=exitMarker.ReplaceAllString(result.Stderr,"") }
  if ctx.Err()!=nil{return result,ctx.Err()}
  return result,nil
}
