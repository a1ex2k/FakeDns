package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func runPasswd(path string) error {
	br := bufio.NewReader(os.Stdin)

	fmt.Print("Set username: ")
	user, err := br.ReadString('\n')
	if err != nil {
		return err
	}
	user = strings.TrimSpace(user)

	if user == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if strings.ContainsAny(user, " \t\r\n:") {
		return fmt.Errorf("username must not contain spaces or ':'")
	}

	fmt.Print("Set password: ")
	p1, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}
	fmt.Println()

	fmt.Print("Repeat password: ")
	p2, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}
	fmt.Println()

	if string(p1) != string(p2) {
		return fmt.Errorf("passwords do not match")
	}
	if len(p1) < 6 {
		return fmt.Errorf("password too short (min 6)")
	}

	hash, err := bcrypt.GenerateFromPassword(p1, bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	line := []byte(user + ":" + string(hash) + "\n")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, line, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
