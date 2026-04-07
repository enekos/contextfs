package main

import (
	"log"
	"mairu/internal/cmd"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func main() {
	// Try loading from current directory
	_ = godotenv.Load()

	// Optionally try loading from executable directory
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		_ = godotenv.Load(filepath.Join(dir, "..", ".env"))
		_ = godotenv.Load(filepath.Join(dir, "..", "..", ".env"))
	}

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
