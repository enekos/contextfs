package eval

type Case struct {
	Query    string
	Expected []string
	Got      []RetrievalResult
}

type Metrics struct {
	MRR    float64
	Recall float64
}

func EvaluateCases(cases []Case, k int) Metrics {
	if len(cases) == 0 {
		return Metrics{}
	}
	var mrr, recall float64
	for _, c := range cases {
		mrr += MeanReciprocalRank(c.Expected, c.Got)
		recall += RecallAtK(c.Expected, c.Got, k)
	}
	return Metrics{MRR: mrr / float64(len(cases)), Recall: recall / float64(len(cases))}
}
