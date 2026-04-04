package eval

import "testing"

func TestEvalUtils(t *testing.T) {
	got := []RetrievalResult{{ID: "x"}, {ID: "y"}}
	if MeanReciprocalRank([]string{"y"}, got) != 0.5 {
		t.Fatal("unexpected mrr")
	}
	if RecallAtK([]string{"x", "z"}, got, 1) != 0.5 {
		t.Fatal("unexpected recall")
	}
}
