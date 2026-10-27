package pia

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestVerboseKeyGenerationDoesNotLogPrivateKey(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	generator := NewPIAWgGenerator(&PIAClientMock{}, PIAWgGeneratorConfig{Verbose: true})
	private, _, err := generator.generateKeys()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), private) {
		t.Fatal("private key appeared in logs")
	}
}

func TestSelectedServerName(t *testing.T) {
	client := &PIAClient{selectedServerName: "test-server"}
	if client.SelectedServerName() != "test-server" {
		t.Fatal("wrong server name")
	}
}
