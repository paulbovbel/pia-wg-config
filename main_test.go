package main

import (
	"flag"
	"os"
	"testing"

	cli "github.com/urfave/cli/v2"
)

func TestCredentials(t *testing.T) {
	for _, tc := range []struct {
		name               string
		env                map[string]string
		args               []string
		username, password string
	}{
		{name: "environment", env: map[string]string{"PIA_USERNAME": "user", "PIA_PASSWORD": " space $and\"quote "}, username: "user", password: " space $and\"quote "},
		{name: "legacy arguments", args: []string{"user", "password"}, username: "user", password: "password"},
		{name: "environment wins", env: map[string]string{"PIA_USERNAME": "env-user", "PIA_PASSWORD": "env-password"}, args: []string{"arg-user", "arg-password"}, username: "env-user", password: "env-password"},
		{name: "incomplete environment does not mix sources", env: map[string]string{"PIA_USERNAME": "env-user"}, args: []string{"arg-user", "arg-password"}, username: "env-user"},
		{name: "empty environment does not fall back", env: map[string]string{"PIA_PASSWORD": ""}, args: []string{"arg-user", "arg-password"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{"PIA_USERNAME", "PIA_PASSWORD"} {
				t.Setenv(name, "")
				if err := os.Unsetenv(name); err != nil {
					t.Fatal(err)
				}
			}
			for name, value := range tc.env {
				t.Setenv(name, value)
			}
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			if err := flags.Parse(tc.args); err != nil {
				t.Fatal(err)
			}
			username, password := credentials(cli.NewContext(nil, flags, nil))
			if username != tc.username || password != tc.password {
				t.Fatal("unexpected credentials")
			}
		})
	}
}
