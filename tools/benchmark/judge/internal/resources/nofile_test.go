package resources

import "testing"

func TestEstimateNoFileUsesBoundedFullObservationBudget(t *testing.T) {
	got, err := EstimateNoFile(10000)
	if err != nil || got != 20256 {
		t.Fatalf("estimate=%d err=%v", got, err)
	}
	if _, err := EstimateNoFile(0); err == nil {
		t.Fatal("zero max in-flight accepted")
	}
}

func TestCheckNoFileLimitFailsClosedWithoutChangingProcessLimit(t *testing.T) {
	if _, err := CheckNoFileLimit(100, 200); err == nil {
		t.Fatal("insufficient descriptor limit accepted")
	}
	if status, err := CheckNoFileLimit(200, 200); err != nil || status.SoftLimit != 200 {
		t.Fatalf("status=%+v err=%v", status, err)
	}
}
