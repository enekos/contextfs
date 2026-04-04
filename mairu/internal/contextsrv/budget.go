package contextsrv

type Budget struct {
	MemoryPerProject int
	SkillPerProject  int
	NodePerProject   int
}

func ExceedsBudget(currentMemories, currentSkills, currentNodes int, b Budget) bool {
	if b.MemoryPerProject > 0 && currentMemories > b.MemoryPerProject {
		return true
	}
	if b.SkillPerProject > 0 && currentSkills > b.SkillPerProject {
		return true
	}
	if b.NodePerProject > 0 && currentNodes > b.NodePerProject {
		return true
	}
	return false
}

func FlushVibeOps(ops []VibeMutationOp, max int) []VibeMutationOp {
	if max <= 0 || len(ops) <= max {
		return ops
	}
	return ops[:max]
}
