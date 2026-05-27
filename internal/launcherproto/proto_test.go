// SPDX-License-Identifier: Apache-2.0

package launcherproto_test

import (
	"bytes"
	"testing"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

func TestRequestRoundTrip(t *testing.T) {
	req := launcherproto.Request{
		Argv: []string{"exec", "foo.mdl"},
		Cwd:  "/project",
		Env:  map[string]string{"MX_DEBUG": "1"},
	}
	var buf bytes.Buffer
	if err := launcherproto.WriteMsg(&buf, req); err != nil {
		t.Fatal(err)
	}
	var got launcherproto.Request
	if err := launcherproto.ReadMsg(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if got.Cwd != "/project" || len(got.Argv) != 2 || got.Env["MX_DEBUG"] != "1" {
		t.Fatalf("round-trip failed: %+v", got)
	}
}

func TestFrameExitRoundTrip(t *testing.T) {
	code := 42
	frame := launcherproto.Frame{Exit: &code}
	var buf bytes.Buffer
	if err := launcherproto.WriteMsg(&buf, frame); err != nil {
		t.Fatal(err)
	}
	var got launcherproto.Frame
	if err := launcherproto.ReadMsg(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if got.Exit == nil || *got.Exit != 42 {
		t.Fatalf("exit code not preserved: %+v", got)
	}
}

func TestFrameStdoutRoundTrip(t *testing.T) {
	frame := launcherproto.Frame{Stream: "stdout", Data: []byte("hello\n")}
	var buf bytes.Buffer
	if err := launcherproto.WriteMsg(&buf, frame); err != nil {
		t.Fatal(err)
	}
	var got launcherproto.Frame
	if err := launcherproto.ReadMsg(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if got.Stream != "stdout" || string(got.Data) != "hello\n" {
		t.Fatalf("stdout frame not preserved: %+v", got)
	}
}

func TestHealthCheckRoundTrip(t *testing.T) {
	frame := launcherproto.Frame{OK: true, Version: "v0.14.0"}
	var buf bytes.Buffer
	if err := launcherproto.WriteMsg(&buf, frame); err != nil {
		t.Fatal(err)
	}
	var got launcherproto.Frame
	if err := launcherproto.ReadMsg(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.Version != "v0.14.0" {
		t.Fatalf("health check frame not preserved: %+v", got)
	}
}
