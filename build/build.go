package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/DNSControl/dnscontrol/v4/pkg/version"
)

var goos = flag.String("os", "", "OS to build (linux, windows, or darwin) Defaults to all.")

func main() {
	flag.Parse()
	flags := fmt.Sprintf(`-s -w -X "github.com/DNSControl/dnscontrol/v4/pkg/version.version=%s"`, version.Version())
	pkg := "github.com/DNSControl/dnscontrol/v4"

	build := func(out, goos string) {
		log.Printf("Building %s", out)
		cmd := exec.Command("go", "build", "-o", out, "-ldflags", flags, pkg)
		os.Setenv("GOOS", goos)
		os.Setenv("GO111MODULE", "on")
		os.Setenv("CGO_ENABLED", "0")
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		// For now just build for amd64 everywhere
		if os.Getenv("GOARCH") == "" {
			os.Setenv("GOARCH", "amd64")
		}

		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, env := range []struct {
		binary, goos string
	}{
		{"dnscontrol-Linux", "linux"},
		{"dnscontrol.exe", "windows"},
		{"dnscontrol-Darwin", "darwin"},
	} {
		if *goos == "" || *goos == env.goos {
			build(env.binary, env.goos)
		}
	}
}
