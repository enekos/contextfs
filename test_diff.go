package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	os.WriteFile("a.txt", []byte("hello\nworld\n"), 0644)
	os.WriteFile("b.txt", []byte("hello\nbrave\nworld\n"), 0644)
	cmd := exec.Command("diff", "-u", "a.txt", "b.txt")
	out, _ := cmd.CombinedOutput()
	fmt.Println(string(out))
}
