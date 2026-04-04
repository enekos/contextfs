import os
import re

def patch_stub(file_path):
    with open(file_path, 'r') as f:
        content = f.read()
    
    if "func (s *stubService)" in content and "func (s *stubService) Ingest(" not in content:
        content += "\nfunc (s *stubService) Ingest(text, baseURI string) ([]mairu.ProposedContextNode, error) { return nil, nil }\n"
        content = content.replace("[]mairu.ProposedContextNode", "[]llm.ProposedContextNode")
    
    if "func (s *parityStubService)" in content and "func (s *parityStubService) Ingest(" not in content:
        content += "\nfunc (s *parityStubService) Ingest(text, baseURI string) ([]llm.ProposedContextNode, error) { return nil, nil }\n"

    with open(file_path, 'w') as f:
        f.write(content)

patch_stub("mairu/internal/contextsrv/server_contract_test.go")
patch_stub("mairu/internal/contextsrv/parity_contract_test.go")

with open("mairu/internal/contextsrv/vibe_test.go", "r") as f:
    vibe = f.read()
if "func (l *vibeLLM) GenerateContent(" not in vibe:
    vibe += "\nfunc (l *vibeLLM) GenerateContent(ctx context.Context, model, prompt string) (string, error) { return \"\", nil }\n"
with open("mairu/internal/contextsrv/vibe_test.go", "w") as f:
    f.write(vibe)

