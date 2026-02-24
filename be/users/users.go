// Package users provides an in-memory user registry mapping static API keys
// to user identities. In production this would be backed by a database or
// secrets manager; here it's intentionally static for simulation purposes.
package users

// User holds identity info for a registered user.
// NOTE: passwords are plaintext here for simulation only — never do this in production.
type User struct {
	ID       string // human-readable name used for accounting
	Key      string // Bearer API key
	Password string // login password (simulation only). In prod, this would be a hash.
	IsAdmin  bool   // admin flag (simulation only)
}

// registry is the static user database keyed by API key.
// TODO(Taman / critical / prod): Replace this with a jwt solution like Firebase Auth.
var registry = map[string]User{
	"alice":   {ID: "alice", Key: "sk-alice-001", Password: "alice123", IsAdmin: false},
	"bob":     {ID: "bob", Key: "sk-bob-001", Password: "bob123", IsAdmin: false},
	"charlie": {ID: "charlie", Key: "sk-charlie-001", Password: "charlie123", IsAdmin: false},
	"admin":   {ID: "admin", Key: "sk-admin-001", Password: "admin123", IsAdmin: true},
}

// byKey is a secondary index for API key → User lookups.
var byKey = func() map[string]User {
	m := make(map[string]User, len(registry))
	for _, u := range registry {
		m[u.Key] = u
	}
	return m
}()

// Lookup resolves an API key to a User.
// Returns the zero User and false if the key is unknown.
func Lookup(key string) (User, bool) {
	u, ok := byKey[key]
	return u, ok
}

// Login validates username + password and returns the User on success.
func Login(id, password string) (User, bool) {
	u, ok := registry[id] // registry is keyed by user ID
	if !ok || u.Password != password {
		return User{}, false
	}
	return u, true
}

// All returns a copy of the full user registry (for admin inspection).
func All() []User {
	out := make([]User, 0, len(registry))
	for _, u := range registry {
		out = append(out, u)
	}
	return out
}
