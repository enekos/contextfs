import os
import re

with open("mairu/internal/contextsrv/repository.go", "r") as f:
    content = f.read()

def extract_funcs(content, names):
    pattern = r"(func \(r \*PostgresRepository\) (?:{0})\(.*?\)(?:.|\n)*?)(?=\nfunc \(r \*PostgresRepository\)|\Z)".format("|".join(names))
    matches = re.findall(pattern, content)
    return "\n\n".join(matches)

# Memory
mem_funcs = ["CreateMemory", "ListMemories", "UpdateMemory", "DeleteMemory"]
mem_code = "package contextsrv\n\nimport (\n\t\"context\"\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"time\"\n)\n\n" + extract_funcs(content, mem_funcs)

# Skill
skill_funcs = ["CreateSkill", "ListSkills", "UpdateSkill", "DeleteSkill"]
skill_code = "package contextsrv\n\nimport (\n\t\"context\"\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"time\"\n)\n\n" + extract_funcs(content, skill_funcs)

# Context
ctx_funcs = ["CreateContextNode", "ListContextNodes", "UpdateContextNode", "DeleteContextNode"]
ctx_code = "package contextsrv\n\nimport (\n\t\"context\"\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"time\"\n)\n\n" + extract_funcs(content, ctx_funcs)

# Search & Outbox & Moderation Queue
search_funcs = ["SearchText", "ListModerationQueue", "ReviewModeration", "EnqueueOutbox", "insertModerationEventTx", "insertAuditTx", "reviewState", "jsonString"]
search_code = "package contextsrv\n\nimport (\n\t\"context\"\n\t\"database/sql\"\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"strings\"\n\t\"time\"\n)\n\n" + extract_funcs(content, search_funcs)

# Core repo
core_funcs = ["NewPostgresRepository", "Close", "Migrate"]
# we'll just parse manually by keeping the top until CreateMemory
idx = content.find("func (r *PostgresRepository) CreateMemory")
core_code = content[:idx]

with open("mairu/internal/contextsrv/repository_memory.go", "w") as f:
    f.write(mem_code)
with open("mairu/internal/contextsrv/repository_skill.go", "w") as f:
    f.write(skill_code)
with open("mairu/internal/contextsrv/repository_context.go", "w") as f:
    f.write(ctx_code)
with open("mairu/internal/contextsrv/repository_search.go", "w") as f:
    f.write(search_code)
with open("mairu/internal/contextsrv/repository.go", "w") as f:
    f.write(core_code)
