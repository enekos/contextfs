import os

def add_import(file_path):
    with open(file_path, "r") as f:
        lines = f.readlines()
    
    out = []
    in_imports = False
    added = False
    for line in lines:
        if line.startswith("import ("):
            in_imports = True
            out.append(line)
        elif in_imports and line.strip() == ")":
            if not added:
                out.append('\t"mairu/internal/llm"\n')
                added = True
            out.append(line)
            in_imports = False
        else:
            out.append(line)
    
    with open(file_path, "w") as f:
        f.writelines(out)

add_import("mairu/internal/contextsrv/server_contract_test.go")
add_import("mairu/internal/contextsrv/parity_contract_test.go")
