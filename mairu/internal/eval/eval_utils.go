package eval

type RetrievalResult struct {
	ID    string
	Score float64
}

func MeanReciprocalRank(expected []string, got []RetrievalResult) float64 {
	for i, r := range got {
		for _, e := range expected {
			if r.ID == e {
				return 1.0 / float64(i+1)
			}
		}
	}
	return 0
}

func RecallAtK(expected []string, got []RetrievalResult, k int) float64 {
	if len(expected) == 0 || k <= 0 {
		return 0
	}
	hits := 0
	limit := k
	if limit > len(got) {
		limit = len(got)
	}
	for i := 0; i < limit; i++ {
		for _, e := range expected {
			if got[i].ID == e {
				hits++
				break
			}
		}
	}
	return float64(hits) / float64(len(expected))
}
