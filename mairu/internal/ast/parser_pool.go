package ast

import (
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
)

var parserPool = sync.Pool{
	New: func() interface{} {
		return sitter.NewParser()
	},
}

// GetParser returns a new or recycled tree-sitter parser.
// The caller MUST call parser.SetLanguage() before using it.
func GetParser() *sitter.Parser {
	return parserPool.Get().(*sitter.Parser)
}

// PutParser returns a parser to the pool.
// Do NOT call Close() on the parser before putting it back.
func PutParser(p *sitter.Parser) {
	if p != nil {
		parserPool.Put(p)
	}
}

var parserPoolState struct {
	sync.Mutex
	initialized bool
}

func InitParserPool() {
	parserPoolState.Lock()
	defer parserPoolState.Unlock()
	parserPoolState.initialized = true
}

func DeleteParserPool() {
	parserPoolState.Lock()
	defer parserPoolState.Unlock()
	parserPoolState.initialized = false
}

func ParserPoolInitialized() bool {
	parserPoolState.Lock()
	defer parserPoolState.Unlock()
	return parserPoolState.initialized
}
