package ast

import "testing"

func TestWalkTemplate(t *testing.T) {
	s := WalkTemplate(`<template><slot/><div v-if="ok" v-for="x in xs"></div></template>`)
	if s.Slots != 1 || s.Branches != 1 || s.Loops != 1 {
		t.Fatalf("unexpected stats: %#v", s)
	}
}
