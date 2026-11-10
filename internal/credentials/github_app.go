package credentials

import (
  "context"
  "crypto"
  "crypto/rand"
  "crypto/rsa"
  "crypto/sha256"
  "crypto/x509"
  "encoding/base64"
  "encoding/json"
  "encoding/pem"
  "errors"
  "fmt"
  "net/http"
  "os"
  "strings"
  "time"
)

type GitHubAppBroker struct {
  AppID string
  InstallationID string
  PrivateKeyPath string
  HTTPClient *http.Client
}
type installationTokenResponse struct { Token string `json:"token"`; ExpiresAt time.Time `json:"expires_at"` }

func NewGitHubAppBrokerFromEnv() (*GitHubAppBroker,error) {
  app:=strings.TrimSpace(os.Getenv("GITHUB_APP_ID"))
  installation:=strings.TrimSpace(os.Getenv("GITHUB_INSTALLATION_ID"))
  keyPath:=strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH"))
  if app==""||installation==""||keyPath=="" { return nil,errors.New("GitHub App not configured: set GITHUB_APP_ID, GITHUB_INSTALLATION_ID and GITHUB_APP_PRIVATE_KEY_PATH") }
  return &GitHubAppBroker{AppID:app,InstallationID:installation,PrivateKeyPath:keyPath,HTTPClient:&http.Client{Timeout:20*time.Second}},nil
}

func (b *GitHubAppBroker) parseKey()(*rsa.PrivateKey,error){
  data,err:=os.ReadFile(b.PrivateKeyPath);if err!=nil{return nil,err}
  block,_:=pem.Decode(data);if block==nil{return nil,errors.New("GitHub App private key is not valid PEM")}
  if key,e:=x509.ParsePKCS1PrivateKey(block.Bytes);e==nil{return key,nil}
  parsed,e:=x509.ParsePKCS8PrivateKey(block.Bytes);if e!=nil{return nil,e}
  key,ok:=parsed.(*rsa.PrivateKey);if !ok{return nil,errors.New("GitHub App key must be RSA")};return key,nil
}

func (b *GitHubAppBroker) appJWT()(string,error){
  key,err:=b.parseKey();if err!=nil{return "",err}
  now:=time.Now().UTC()
  enc:=base64.RawURLEncoding
  header:=enc.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
  payload:=enc.EncodeToString([]byte(fmt.Sprintf(`{"iat":%d,"exp":%d,"iss":"%s"}`,now.Add(-30*time.Second).Unix(),now.Add(8*time.Minute).Unix(),b.AppID)))
  unsigned:=header+"."+payload
  digest:=sha256.Sum256([]byte(unsigned))
  signature,err:=rsa.SignPKCS1v15(rand.Reader,key,crypto.SHA256,digest[:]);if err!=nil{return "",err}
  return unsigned+"."+enc.EncodeToString(signature),nil
}

// InstallationToken returns a short-lived token scoped to exactly one repository.
// The token must remain on the host and must never be passed to a workload or LLM.
func (b *GitHubAppBroker) InstallationToken(ctx context.Context, owner, repository string)(string,time.Time,error){
  if owner==""||repository==""||strings.ContainsAny(owner+repository,"/\\\n\r"){return "",time.Time{},errors.New("invalid GitHub repository identifier")}
  jwt,err:=b.appJWT();if err!=nil{return "",time.Time{},err}
  endpoint:=fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens",b.InstallationID)
  body:=map[string]interface{}{"repositories":[]string{repository},"permissions":map[string]string{"contents":"write","metadata":"read"}}
  raw,_:=json.Marshal(body)
  req,err:=http.NewRequestWithContext(ctx,http.MethodPost,endpoint,strings.NewReader(string(raw)));if err!=nil{return "",time.Time{},err}
  req.Header.Set("Authorization","Bearer "+jwt);req.Header.Set("Accept","application/vnd.github+json");req.Header.Set("X-GitHub-Api-Version","2022-11-28");req.Header.Set("Content-Type","application/json")
  client:=b.HTTPClient;if client==nil{client=&http.Client{Timeout:20*time.Second}}
  resp,err:=client.Do(req);if err!=nil{return "",time.Time{},err};defer resp.Body.Close()
  if resp.StatusCode!=http.StatusCreated{var bodyErr struct{Message string `json:"message"`};_=json.NewDecoder(resp.Body).Decode(&bodyErr);return "",time.Time{},fmt.Errorf("GitHub App token request failed (%d): %s",resp.StatusCode,bodyErr.Message)}
  var result installationTokenResponse;if err=json.NewDecoder(resp.Body).Decode(&result);err!=nil{return "",time.Time{},err};if result.Token==""{return "",time.Time{},errors.New("GitHub returned an empty installation token")}
  return result.Token,result.ExpiresAt,nil
}
