package eval

func SeedEvalCases() []Case {
	return []Case{
		{
			Query:    "authentication",
			Expected: []string{"mem_1"},
			Got:      []RetrievalResult{{ID: "mem_1", Score: 0.9}},
		},
	}
}
