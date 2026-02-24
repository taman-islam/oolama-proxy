package limiter

import (
	"fmt"
	"lb/users"
	"sync"
	"sync/atomic"

	"golang.org/x/time/rate"
)

// tokenQuotaGrace is the number of extra tokens a user may consume beyond
// their configured quota before requests start being rejected. This tolerates
// the inherent async delay between a request completing and its tokens being
// recorded — keeping the common case off the hot path.
// TODO(Taman / critical): Make configurable.
const tokenQuotaGrace = 5

const (
	FREE_TIER_RPS            = 1000
	FREE_TIER_TOKENS         = 100000
	FREE_TIER_TOKENS_PER_REQ = 4000

	INF_RPS           = -1
	INF_TOKENS        = -1
	INF_TOKEN_PER_REQ = -1
)

// userLimit holds rate + quota state for one user.
type userLimit struct {
	limiter         *rate.Limiter
	maxTokens       int64        // INF_TOKENS = unlimited
	maxTokensPerReq int64        // INF_TOKEN_PER_REQ = unlimited; caps max_tokens per request
	usedTokens      atomic.Int64 // total tokens consumed
}

// Limiter manages per-user RPS and token quota limits.
type Limiter struct {
	mu    sync.Mutex
	users map[string]*userLimit
}

func New() *Limiter {
	return &Limiter{users: make(map[string]*userLimit)}
}

func (l *Limiter) getOrCreate(user string) *userLimit {
	l.mu.Lock()
	defer l.mu.Unlock()
	if u, ok := l.users[user]; ok {
		return u
	}
	// New users start on the free tier.
	u := &userLimit{
		limiter:         rate.NewLimiter(rate.Limit(FREE_TIER_RPS), FREE_TIER_RPS),
		maxTokens:       FREE_TIER_TOKENS,
		maxTokensPerReq: FREE_TIER_TOKENS_PER_REQ,
	}
	l.users[user] = u
	return u
}

// SetLimits updates RPS, total token quota, and per-request token cap for a user.
// Use INF_RPS / INF_TOKENS / INF_TOKEN_PER_REQ (-1) to remove a limit.
// Use 0 for any field to leave it unchanged.
// Takes effect immediately for all subsequent requests.
func (l *Limiter) SetLimits(user string, rps int, maxTokens, maxTokensPerReq int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	u, ok := l.users[user]
	if !ok {
		u = &userLimit{}
		l.users[user] = u
	}
	switch rps {
	case INF_RPS:
		u.limiter = rate.NewLimiter(rate.Inf, 0)
	default:
		u.limiter = rate.NewLimiter(rate.Limit(rps), rps)
	}
	if maxTokens != 0 {
		u.maxTokens = maxTokens // INF_TOKENS (-1) stored as-is = unlimited
	}
	if maxTokensPerReq != 0 {
		u.maxTokensPerReq = maxTokensPerReq // INF_TOKEN_PER_REQ (-1) stored as-is = unlimited
	}
	// Reset consumed token counter when limits are updated.
	u.usedTokens.Store(0)
}

// MaxTokensPerRequest returns the per-request token cap for a user (INF_TOKEN_PER_REQ = unlimited).
func (l *Limiter) MaxTokensPerRequest(user string) int64 {
	u := l.getOrCreate(user)
	return u.maxTokensPerReq
}

// CheckRPS returns an error (429) if the user has exceeded their RPS limit.
func (l *Limiter) CheckRPS(user string) error {
	u := l.getOrCreate(user)
	if !u.limiter.Allow() {
		return fmt.Errorf("rate limit exceeded")
	}
	return nil
}

// CheckQuota returns an error (403) if the user has exceeded their token quota.
// A grace of tokenQuotaGrace tokens is allowed beyond the configured limit to
// account for async accounting — the common (under-quota) case never blocks.
func (l *Limiter) CheckQuota(user string) error {
	u := l.getOrCreate(user)
	if u.maxTokens == INF_TOKENS {
		return nil // unlimited
	}
	if u.usedTokens.Load() >= u.maxTokens+tokenQuotaGrace {
		return fmt.Errorf("token quota exceeded")
	}
	return nil
}

// ConsumeTokens atomically records token usage after inference.
func (l *Limiter) ConsumeTokens(user string, n int) {
	u := l.getOrCreate(user)
	u.usedTokens.Add(int64(n))
}

// LimitInfo holds limit config for one user (used by admin UI).
type LimitInfo struct {
	MaxTokens       int64
	MaxTokensPerReq int64
	UsedTokens      int64
	RPS             float64
}

func (l *Limiter) GetAllLimits() map[string]LimitInfo {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make(map[string]LimitInfo)

	// Seed all registered users, using their current limiter state if known
	// or free-tier defaults if they haven't made a request yet.
	for _, u := range users.All() {
		if lu, ok := l.users[u.ID]; ok {
			out[u.ID] = LimitInfo{
				MaxTokens:       lu.maxTokens,
				MaxTokensPerReq: lu.maxTokensPerReq,
				UsedTokens:      lu.usedTokens.Load(),
				RPS:             float64(lu.limiter.Limit()),
			}
		} else {
			out[u.ID] = LimitInfo{
				MaxTokens:       FREE_TIER_TOKENS,
				MaxTokensPerReq: FREE_TIER_TOKENS_PER_REQ,
				UsedTokens:      0,
				RPS:             FREE_TIER_RPS,
			}
		}
	}
	return out
}
