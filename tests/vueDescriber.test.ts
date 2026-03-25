import { describe, expect, it, beforeAll, afterAll } from "vitest";
import { VueDescriber } from "../src/ast/vueDescriber";

describe("VueDescriber", () => {
  const describer = new VueDescriber();

  beforeAll(async () => {
    await describer.initParsers();
  });

  afterAll(() => {
    describer.deleteParsers();
  });

  it("extracts script and template symbols from Vue SFC", () => {
    const source = `
<script setup lang="ts">
import { ref } from 'vue'
import ChildPanel from './ChildPanel.vue'

const isOpen = ref(false)

function toggle() {
  isOpen.value = !isOpen.value
}
</script>

<template>
  <div v-if="isOpen">
    <ChildPanel />
  </div>
</template>
`;

    const result = describer.extractFileGraph("/tmp/test/App.vue", source);

    const symbolIds = result.symbols.map((s) => s.id);
    expect(symbolIds).toContain("fn:toggle");
    expect(symbolIds).toContain("var:isOpen");
    expect(symbolIds).toContain("tpl:App");

    // v-if should produce a tpl-branch
    const branchSymbol = result.symbols.find((s) => s.kind === "tpl-branch");
    expect(branchSymbol).toBeDefined();

    const edgeKeys = result.edges.map((e) => `${e.kind}:${e.from}->${e.to}`);
    // render edges from template
    expect(edgeKeys.some((k) => k.startsWith("render:"))).toBe(true);
  });

  it("extracts v-for as tpl-loop", () => {
    const source = `
<script setup lang="ts">
import ItemCard from './ItemCard.vue'
</script>

<template>
  <ItemCard v-for="item in items" :key="item.id" />
</template>
`;

    const result = describer.extractFileGraph("/tmp/test/List.vue", source);

    const symbolIds = result.symbols.map((s) => s.id);
    const loopSymbol = result.symbols.find((s) => s.kind === "tpl-loop");
    expect(loopSymbol).toBeDefined();
    expect(loopSymbol!.name).toContain("for_");

    const edgeKeys = result.edges.map((e) => `${e.kind}:${e.from}->${e.to}`);
    expect(edgeKeys).toContain("render:tpl-loop:List.for_items->type:ItemCard");
  });

  it("extracts v-if/v-else-if/v-else chain", () => {
    const source = `
<template>
  <div v-if="status === 'loading'">Loading...</div>
  <div v-else-if="status === 'error'">Error!</div>
  <div v-else>Done</div>
</template>
`;

    const result = describer.extractFileGraph("/tmp/test/Status.vue", source);

    const branchSymbols = result.symbols.filter((s) => s.kind === "tpl-branch");
    expect(branchSymbols.length).toBe(3);

    const branchNames = branchSymbols.map((s) => s.name);
    expect(branchNames.some((n) => n.startsWith("if_"))).toBe(true);
    expect(branchNames.some((n) => n.startsWith("elseif_"))).toBe(true);
    expect(branchNames).toContain("else");
  });

  it("extracts slot definitions", () => {
    const source = `
<template>
  <div>
    <slot></slot>
  </div>
</template>
`;

    const result = describer.extractFileGraph("/tmp/test/Wrapper.vue", source);

    const slotSymbol = result.symbols.find((s) => s.kind === "tpl-slot");
    expect(slotSymbol).toBeDefined();
    expect(slotSymbol!.name).toBe("default");

    const edgeKeys = result.edges.map((e) => `${e.kind}:${e.from}->${e.to}`);
    expect(edgeKeys).toContain("slot:tpl:Wrapper->tpl-slot:Wrapper.default");
  });

  it("handles template-only SFC without script", () => {
    const source = `
<template>
  <div>
    <span>Hello</span>
  </div>
</template>
`;

    const result = describer.extractFileGraph("/tmp/test/Simple.vue", source);

    const symbolIds = result.symbols.map((s) => s.id);
    expect(symbolIds).toContain("tpl:Simple");

    // Should not crash
    expect(result.symbols.length).toBeGreaterThan(0);

    // Template root should exist
    const tplRoot = result.symbols.find((s) => s.kind === "tpl");
    expect(tplRoot).toBeDefined();
  });

  it("handles malformed template gracefully", () => {
    const source = `
<script setup lang="ts">
const x = 1
</script>

<template>
  <div>
    <span>unclosed
  </div>
</template>
`;

    // Should not throw
    expect(() => {
      const result = describer.extractFileGraph("/tmp/test/Bad.vue", source);
      // Should still return a valid result (at least script extraction)
      expect(result).toBeDefined();
      expect(result.symbols).toBeDefined();
      expect(result.edges).toBeDefined();
    }).not.toThrow();
  });

  it("extracts v-show through Vue pipeline", () => {
    const source = `
<template>
  <div v-show="isVisible">Content</div>
</template>
`;

    const result = describer.extractFileGraph("/tmp/test/Toggle.vue", source);

    const branchSymbol = result.symbols.find((s) => s.kind === "tpl-branch");
    expect(branchSymbol).toBeDefined();
    expect(branchSymbol!.name).toContain("show_");
  });
});
