
package main
import (
  "encoding/json"
  "fmt"
  "os"
  "github.com/dnviti/arsenale/backend/internal/desktopbroker"
)
func main() {
  token, err := desktopbroker.DecryptToken(os.Args[1], os.Args[2])
  if err != nil { panic(err) }
  out, _ := json.MarshalIndent(token, "", "  ")
  fmt.Println(string(out))
}
