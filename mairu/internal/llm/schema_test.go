package llm

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestGenerateSchema(t *testing.T) {
	type Person struct {
		Name string `json:"name" desc:"The person's name"`
		Age  int    `json:"age" desc:"Age in years"`
	}
	s := GenerateSchema(Person{})
	b, _ := json.MarshalIndent(s, "", "  ")
	fmt.Println(string(b))
}
