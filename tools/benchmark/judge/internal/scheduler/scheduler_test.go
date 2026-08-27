package scheduler

import (
	"math/big"
	"testing"
	"time"
)

func TestFixedOffsetsAreOriginBased(t *testing.T) {
	rate, _ := new(big.Rat).SetString("0.3")
	offsets, err := FixedOffsets(rate, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(offsets) != 3 || offsets[0] != 0 || offsets[1] != 3333333333*time.Nanosecond || offsets[2] != 6666666666*time.Nanosecond {
		t.Fatalf("offsets=%v", offsets)
	}
}

func TestFixedOffsetsFractionalRatesUseOriginalOrigin(t *testing.T) {
	for _, raw := range []string{"0.1", "0.3", "1.5", "7.25"} {
		rate, _ := new(big.Rat).SetString(raw)
		offsets, err := FixedOffsets(rate, 30*time.Second)
		if err != nil || len(offsets) == 0 || offsets[0] != 0 {
			t.Fatalf("rate=%s offsets=%v err=%v", raw, offsets, err)
		}
		for index, offset := range offsets {
			if want := OffsetFor(rate, index); offset != want {
				t.Fatalf("rate=%s index=%d offset=%s want=%s", raw, index, offset, want)
			}
		}
	}
}

func TestArrivalCountUsesHalfOpenLoadWindow(t *testing.T) {
	cases := []struct {
		rate string
		want int
	}{
		{rate: "0.1", want: 3},
		{rate: "0.3", want: 9},
		{rate: "1.5", want: 45},
		{rate: "7.25", want: 218},
	}
	for _, test := range cases {
		rate, _ := new(big.Rat).SetString(test.rate)
		got, err := ArrivalCount(rate, 30*time.Second)
		if err != nil || got != test.want {
			t.Fatalf("rate=%s count=%d err=%v want=%d", test.rate, got, err, test.want)
		}
		if final := OffsetFor(rate, got-1); final >= 30*time.Second {
			t.Fatalf("rate=%s final offset %s escaped [0,duration)", test.rate, final)
		}
		if next := OffsetFor(rate, got); next < 30*time.Second {
			t.Fatalf("rate=%s next offset %s remained in [0,duration)", test.rate, next)
		}
	}
}

func TestExactOffsetsScheduleOnlyRequestedTotal(t *testing.T) {
	rate, _ := new(big.Rat).SetString("1.5")
	for _, total := range []int{1, 2, 10} {
		offsets, err := ExactOffsets(rate, total)
		if err != nil || len(offsets) != total {
			t.Fatalf("total=%d offsets=%v err=%v", total, offsets, err)
		}
		for index, offset := range offsets {
			if want := OffsetFor(rate, index); offset != want {
				t.Fatalf("total=%d index=%d offset=%s want=%s", total, index, offset, want)
			}
		}
		if len(offsets) > 1 && offsets[len(offsets)-1] <= offsets[len(offsets)-2] {
			t.Fatalf("total=%d did not preserve rate spacing: %v", total, offsets)
		}
	}
}

func TestExactArrivalCountRejectsZeroOrNegative(t *testing.T) {
	for _, total := range []int{0, -1} {
		if _, err := ExactArrivalCount(total); err == nil {
			t.Fatalf("total=%d accepted", total)
		}
	}
}

func TestPoolDoesNotDelayUnavailableArrival(t *testing.T) {
	pool, err := NewPool([]string{"a"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	user, ok := pool.Lease(now)
	if !ok || user.Alias != "a" {
		t.Fatal("first lease failed")
	}
	if err := pool.Accepted("a", now, time.Second, 0); err != nil {
		t.Fatal(err)
	}
	if _, ok := pool.Lease(now.Add(500 * time.Millisecond)); ok {
		t.Fatal("cooldown user was reused")
	}
	if _, ok := pool.Lease(now.Add(time.Second)); !ok {
		t.Fatal("expired cooldown user unavailable")
	}
}

func TestDisabledUserIsNeverReused(t *testing.T) {
	pool, err := NewPool([]string{"a"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Disable("a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := pool.Lease(time.Now().Add(time.Hour)); ok {
		t.Fatal("disabled authentication-failure user was reused")
	}
}

func TestRequiredUsersIncludesHeadroom(t *testing.T) {
	rate, _ := new(big.Rat).SetString("0.3")
	got, err := RequiredUsers(rate, 3*time.Second, 100*time.Millisecond, 500*time.Millisecond, 25)
	if err != nil {
		t.Fatal(err)
	}
	if got != 3 {
		t.Fatalf("required users=%d, want 3", got)
	}
}

func TestBurstPlanIsDeterministicAndDistinct(t *testing.T) {
	aliases := []string{"a", "b", "c"}
	first, err := BurstPlan(aliases, 3, 9, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := BurstPlan(aliases, 3, 9, time.Millisecond)
	seen := map[string]bool{}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("plan not deterministic: %#v != %#v", first, second)
		}
		if seen[first[i].Alias] {
			t.Fatal("duplicate burst user")
		}
		seen[first[i].Alias] = true
	}
}

func TestCommonOriginBurstPlanCreatesExactlyOneArrivalPerSelectedUser(t *testing.T) {
	aliases := []string{"bench-001", "bench-002", "bench-003", "bench-004"}
	plan, err := BurstPlan(aliases, len(aliases), 19, 0)
	if err != nil || len(plan) != len(aliases) {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	seen := map[string]bool{}
	for _, arrival := range plan {
		if arrival.Offset != 0 || seen[arrival.Alias] {
			t.Fatalf("arrival=%+v seen=%v", arrival, seen)
		}
		seen[arrival.Alias] = true
	}
	if _, err := BurstPlan(aliases, len(aliases)+1, 19, 0); err == nil {
		t.Fatal("burst plan allowed N+1 selected users")
	}
}
