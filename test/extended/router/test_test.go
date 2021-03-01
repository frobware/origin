package router

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestFoo(t *testing.T) {
	pwd, _ := os.Getwd()
	fmt.Println(pwd)
	data, err := makeCompressedTarArchive([]string{"./http2/cluster/server/server.go"})
	fmt.Println(err)
	lines := split(base64.StdEncoding.EncodeToString(data), 76)
	base64 := strings.Join(lines, "\n")
	fmt.Println(base64)
	fmt.Println(strings.ReplaceAll(base64, "\n", "\\n"))
}
