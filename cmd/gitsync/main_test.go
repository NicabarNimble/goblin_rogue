package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCommand(t *testing.T) {
	cmd := newRootCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "gitsync", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestSubcommands(t *testing.T) {
	cmd := newRootCmd()
	subcommands := cmd.Commands()

	// Verify all expected subcommands exist
	commandNames := make(map[string]bool)
	for _, subcmd := range subcommands {
		commandNames[subcmd.Name()] = true
	}

	expectedCommands := []string{"init", "run", "status", "logs", "configure"}
	for _, expected := range expectedCommands {
		assert.True(t, commandNames[expected], "Expected command %s not found", expected)
	}
}

func TestInitCommand(t *testing.T) {
	cmd := newInitCmd()
	assert.NotNil(t, cmd)

	// Test required flags by attempting to execute without them
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, buf.String(), "required flag(s) \"source\", \"target\" not set")
}

func TestRunCommand(t *testing.T) {
	cmd := newRunCmd()
	assert.NotNil(t, cmd)

	// Test required flag by attempting to execute without it
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, buf.String(), "required flag(s) \"repo\" not set")
}

func TestStatusCommand(t *testing.T) {
	cmd := newStatusCmd()
	assert.NotNil(t, cmd)

	// Test required flag by attempting to execute without it
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, buf.String(), "required flag(s) \"repo\" not set")
}

func TestLogsCommand(t *testing.T) {
	cmd := newLogsCmd()
	assert.NotNil(t, cmd)

	// Test required flags by attempting to execute without them
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, buf.String(), "required flag(s) \"repo\", \"run-id\" not set")
}

func TestConfigureCommand(t *testing.T) {
	cmd := newConfigureCmd()
	assert.NotNil(t, cmd)
}
