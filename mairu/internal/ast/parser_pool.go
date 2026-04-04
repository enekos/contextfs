package ast

import (
	"sync"
)

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
