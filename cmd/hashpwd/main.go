package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: hashpwd <password>")
		fmt.Println("Example: hashpwd mySecretPassword123")
		os.Exit(1)
	}

	password := os.Args[1]
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Error generating hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Bcrypt hash for '%s':\n%s\n", password, string(hash))
	fmt.Println("\nAdd this to your config.yaml:")
	fmt.Println("admin:")
	fmt.Println("  enabled: true")
	fmt.Println("  username: admin")
	fmt.Printf("  password_hash: \"%s\"\n", string(hash))
}
