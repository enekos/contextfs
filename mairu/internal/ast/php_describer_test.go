package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPHPDescriber(t *testing.T) {
	d := PHPDescriber{}
	assert.Equal(t, "php", d.LanguageID())
	assert.Equal(t, []string{".php"}, d.Extensions())

	source := `<?php
namespace App\Services;

use App\Models\User;
use Illuminate\Support\Facades\Log;

class UserService {
    private $db;

    public function __construct($db) {
        $this->db = $db;
    }

    public function createUser(string $name, string $email) {
        $user = new User($name, $email);
        $this->save($user);
        Log::info("User created");
        return $user;
    }

    private function save($user) {
        // save to db
        $this->db->insert($user);
    }
}

function helperFunction() {
    echo "helper";
}
`
	graph := d.ExtractFileGraph("test.php", source)

	assert.Contains(t, graph.Imports, "App\\Models\\User")
	assert.Contains(t, graph.Imports, "Illuminate\\Support\\Facades\\Log")

	// Verify symbols
	symbolMap := map[string]LogicSymbol{}
	for _, s := range graph.Symbols {
		symbolMap[s.ID] = s
	}

	assert.Contains(t, symbolMap, "cls:App\\Services\\UserService")
	assert.True(t, symbolMap["cls:App\\Services\\UserService"].Exported)

	assert.Contains(t, symbolMap, "mtd:App\\Services\\UserService::__construct")
	assert.True(t, symbolMap["mtd:App\\Services\\UserService::__construct"].Exported)

	assert.Contains(t, symbolMap, "mtd:App\\Services\\UserService::createUser")
	assert.True(t, symbolMap["mtd:App\\Services\\UserService::createUser"].Exported)

	assert.Contains(t, symbolMap, "mtd:App\\Services\\UserService::save")
	assert.False(t, symbolMap["mtd:App\\Services\\UserService::save"].Exported)

	assert.Contains(t, symbolMap, "fn:App\\Services\\helperFunction")

	// Verify edges
	hasCreateToSave := false
	for _, e := range graph.Edges {
		if e.From == "mtd:App\\Services\\UserService::createUser" && e.To == "mtd:App\\Services\\UserService::save" {
			hasCreateToSave = true
		}
	}
	assert.True(t, hasCreateToSave, "Should have a call edge from createUser to save")
}
