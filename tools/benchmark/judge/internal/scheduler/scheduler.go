// Package scheduler provides deterministic, open-loop arrival plans and local
// user eligibility accounting. It never waits for Judge completion.
package scheduler

import (
	"errors"
	"math"
	"math/big"
	"math/rand/v2"
	"sort"
	"time"
)

type UserState string

const (
	Eligible    UserState = "ELIGIBLE"
	Leased      UserState = "LEASED_FOR_POST"
	Cooldown    UserState = "COOLDOWN_UNTIL"
	Quarantined UserState = "QUARANTINED"
	Disabled    UserState = "DISABLED"
)

type User struct {
	Alias        string
	State        UserState
	NextEligible time.Time
	Tie          uint64
}

type Pool struct {
	users map[string]*User
}

func NewPool(aliases []string, seed uint64) (*Pool, error) {
	if len(aliases) == 0 {
		return nil, errors.New("at least one user is required")
	}
	pool := &Pool{users: make(map[string]*User, len(aliases))}
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	for _, alias := range aliases {
		if alias == "" {
			return nil, errors.New("empty benchmark user alias")
		}
		if _, exists := pool.users[alias]; exists {
			return nil, errors.New("duplicate benchmark user alias")
		}
		pool.users[alias] = &User{Alias: alias, State: Eligible, Tie: rng.Uint64()}
	}
	return pool, nil
}

// Lease picks a deterministic currently eligible user. A miss is returned
// immediately; callers must record it instead of delaying/bunching an arrival.
func (p *Pool) Lease(now time.Time) (*User, bool) {
	candidates := make([]*User, 0, len(p.users))
	for _, user := range p.users {
		if (user.State == Eligible || user.State == Cooldown || user.State == Quarantined) && !now.Before(user.NextEligible) {
			candidates = append(candidates, user)
		}
	}
	if len(candidates) == 0 {
		return nil, false
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].NextEligible.Equal(candidates[j].NextEligible) {
			if candidates[i].Tie == candidates[j].Tie {
				return candidates[i].Alias < candidates[j].Alias
			}
			return candidates[i].Tie < candidates[j].Tie
		}
		return candidates[i].NextEligible.Before(candidates[j].NextEligible)
	})
	user := candidates[0]
	user.State = Leased
	return cloneUser(user), true
}

// LeaseAlias reserves a preselected burst user. It is intentionally separate
// from Lease so a barrier burst can preserve its deterministic user plan.
func (p *Pool) LeaseAlias(alias string, now time.Time) (*User, bool) {
	user, ok := p.users[alias]
	if !ok || user.State == Disabled || user.State == Leased || now.Before(user.NextEligible) {
		return nil, false
	}
	user.State = Leased
	return cloneUser(user), true
}

func (p *Pool) Accepted(alias string, observed time.Time, cooldown, guard time.Duration) error {
	return p.release(alias, Cooldown, observed.Add(cooldown+guard))
}

func (p *Pool) Quarantine(alias string, observed time.Time, cooldown, guard time.Duration) error {
	return p.release(alias, Quarantined, observed.Add(cooldown+guard))
}

func (p *Pool) RateLimited(alias string, observed time.Time, cooldown, guard, retryAfter time.Duration) error {
	delay := cooldown
	if retryAfter > delay {
		delay = retryAfter
	}
	return p.release(alias, Quarantined, observed.Add(delay+guard))
}

func (p *Pool) Disable(alias string) error {
	user, ok := p.users[alias]
	if !ok {
		return errors.New("unknown benchmark user")
	}
	user.State = Disabled
	return nil
}

func (p *Pool) release(alias string, state UserState, next time.Time) error {
	user, ok := p.users[alias]
	if !ok {
		return errors.New("unknown benchmark user")
	}
	if user.State != Leased {
		return errors.New("benchmark user is not leased")
	}
	user.State = state
	user.NextEligible = next
	return nil
}

func (p *Pool) CountEligible(at time.Time) int {
	count := 0
	for _, user := range p.users {
		if user.State != Disabled && user.State != Leased && !at.Before(user.NextEligible) {
			count++
		}
	}
	return count
}

func cloneUser(value *User) *User {
	copy := *value
	return &copy
}

// FixedOffsets derives every arrival from the original load origin. Values are
// exact rational intervals rounded down to a nanosecond, not accumulated adds.
func FixedOffsets(rate *big.Rat, duration time.Duration) ([]time.Duration, error) {
	if rate == nil || rate.Sign() <= 0 || duration <= 0 {
		return nil, errors.New("positive rate and duration are required")
	}
	count, err := ArrivalCount(rate, duration)
	if err != nil {
		return nil, err
	}
	// This helper is intentionally for tests and small plans. The runner streams
	// sustained arrivals instead of materializing an unbounded slice.
	if count > 1_000_000 {
		return nil, errors.New("too many offsets to materialize; stream arrivals instead")
	}
	offsets := make([]time.Duration, 0, count)
	for index := 0; index < count; index++ {
		offset := OffsetFor(rate, index)
		offsets = append(offsets, offset)
	}
	return offsets, nil
}

// ExactOffsets derives exactly total open-loop arrival offsets from the
// original origin. It deliberately does not convert a total into a duration.
func ExactOffsets(rate *big.Rat, total int) ([]time.Duration, error) {
	if rate == nil || rate.Sign() <= 0 || total <= 0 {
		return nil, errors.New("positive rate and total submissions are required")
	}
	if total > 1_000_000 {
		return nil, errors.New("too many offsets to materialize; stream arrivals instead")
	}
	offsets := make([]time.Duration, total)
	for index := range offsets {
		offsets[index] = OffsetFor(rate, index)
	}
	return offsets, nil
}

// ExactArrivalCount validates and returns an exact requested measured volume.
// The runner streams this count without materializing a plan.
func ExactArrivalCount(total int) (int, error) {
	if total <= 0 {
		return 0, errors.New("total submissions must be positive")
	}
	return total, nil
}

// ArrivalCount returns the number of half-open arrival offsets in [0,duration).
// For t_i = i/rate it is ceil(rate * duration), calculated exactly as a rational
// number rather than by repeated duration addition or floating-point arithmetic.
func ArrivalCount(rate *big.Rat, duration time.Duration) (int, error) {
	if rate == nil || rate.Sign() <= 0 || duration <= 0 {
		return 0, errors.New("positive rate and duration are required")
	}
	count := new(big.Rat).Mul(rate, big.NewRat(int64(duration), int64(time.Second)))
	value := ceilBigRat(count)
	if !value.IsInt64() || value.Int64() > int64(^uint(0)>>1) {
		return 0, errors.New("arrival count exceeds local integer capacity")
	}
	return int(value.Int64()), nil
}

// OffsetFor derives each arrival from its original origin. It deliberately
// avoids repeatedly adding a rounded duration.
func OffsetFor(rate *big.Rat, index int) time.Duration {
	seconds := new(big.Rat).Quo(big.NewRat(int64(index), 1), rate)
	ns := new(big.Rat).Mul(seconds, big.NewRat(int64(time.Second), 1))
	value := new(big.Int).Quo(ns.Num(), ns.Denom())
	if !value.IsInt64() || value.Int64() > math.MaxInt64 {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(value.Int64())
}

func ceilBigRat(value *big.Rat) *big.Int {
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(value.Num(), value.Denom(), remainder)
	if remainder.Sign() != 0 && value.Sign() > 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	return quotient
}

func RequiredUsers(rate *big.Rat, cooldown, guard, submitLatencyBudget time.Duration, headroomPercent float64) (int, error) {
	if rate == nil || rate.Sign() <= 0 || cooldown <= 0 || guard < 0 || submitLatencyBudget < 0 || headroomPercent < 0 {
		return 0, errors.New("invalid user-pool parameters")
	}
	cycle := cooldown + guard + submitLatencyBudget
	load := new(big.Rat).Mul(rate, big.NewRat(int64(cycle), int64(time.Second)))
	base := ceilRat(load)
	spare := int(math.Ceil(float64(base) * headroomPercent / 100))
	if spare < 1 {
		spare = 1
	}
	return base + spare, nil
}

func ceilRat(value *big.Rat) int {
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(value.Num(), value.Denom(), remainder)
	if remainder.Sign() != 0 && value.Sign() > 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	return int(quotient.Int64())
}

type BurstArrival struct {
	Alias  string
	Offset time.Duration
}

func BurstPlan(aliases []string, size int, seed uint64, jitter time.Duration) ([]BurstArrival, error) {
	if size <= 0 || size > len(aliases) || jitter < 0 {
		return nil, errors.New("invalid burst plan")
	}
	values := append([]string(nil), aliases...)
	sort.Strings(values)
	rng := rand.New(rand.NewPCG(seed, seed^0xd1b54a32d192ed03))
	rng.Shuffle(len(values), func(i, j int) { values[i], values[j] = values[j], values[i] })
	arrivals := make([]BurstArrival, size)
	for i := range arrivals {
		offset := time.Duration(0)
		if jitter > 0 {
			offset = time.Duration(rng.Int64N(int64(jitter) + 1))
		}
		arrivals[i] = BurstArrival{Alias: values[i], Offset: offset}
	}
	return arrivals, nil
}

func BurstSpread(starts []time.Time) time.Duration {
	if len(starts) < 2 {
		return 0
	}
	min, max := starts[0], starts[0]
	for _, value := range starts[1:] {
		if value.Before(min) {
			min = value
		}
		if value.After(max) {
			max = value
		}
	}
	return max.Sub(min)
}
