package main

import (
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"
)

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func s2b(s string) (b []byte) {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Len = sh.Len
	bh.Cap = sh.Len
	return b
}

func clearScreen() {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Windows uses "cmd /c cls" to clear the screen
		cmd = exec.Command("cmd", "/c", "cls")
	case "linux", "darwin":
		// Linux and macOS (darwin) use "clear"
		cmd = exec.Command("clear")
	default:
		// Fallback for other systems, though may not be bulletproof
		return
	}
	cmd.Stdout = os.Stdout // Set the command output to the standard output
	cmd.Run()              // Execute the command
}

func terminalWidth() int {
	fd := int(os.Stdout.Fd())
	if term.IsTerminal(fd) {
		width, _, _ := term.GetSize(fd)
		return width
	}
	return 0
}

func displayHelp() {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Command/Flag", "Description"})
	for _, v := range defaultMenu {
		if v.Description != "" {
			table.Append([]string{v.Text, v.Description})
		}
	}
	table.Render()
}
