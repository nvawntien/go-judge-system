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

func TestEstimateNoFileAdmissionOnlyUsesOneConnectionSlotAndReportsRecommendation(t *testing.T) {
	got, err := EstimateNoFileFor(100000, false)
	if err != nil || got != 100256 {
		t.Fatalf("estimate=%d err=%v", got, err)
	}
	if got, err := EstimateNoFileFor(100000, true); err != nil || got != 200256 {
		t.Fatalf("full estimate=%d err=%v", got, err)
	}
}

func TestEstimateNoFileRealisticUsesSeparateLongLivedSSEQualificationModel(t *testing.T) {
	got, err := EstimateNoFileForObservation(100000, ObservationRealistic)
	if err != nil || got != 200256 {
		t.Fatalf("estimate=%d err=%v", got, err)
	}
	if _, err := EstimateNoFileForObservation(1, Observation("unknown")); err == nil {
		t.Fatal("unknown observation model accepted")
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
