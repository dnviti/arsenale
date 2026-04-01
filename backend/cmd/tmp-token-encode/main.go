
package main
import (
  "encoding/json"
  "fmt"
  "os"
  "github.com/dnviti/arsenale/backend/internal/desktopbroker"
)
func main() {
  var token desktopbroker.ConnectionToken
  if err := json.NewDecoder(os.Stdin).Decode(&token); err != nil { panic(err) }
  out, err := desktopbroker.EncryptToken(os.Args[1], token)
  if err != nil { panic(err) }
  fmt.Println(out)
}
