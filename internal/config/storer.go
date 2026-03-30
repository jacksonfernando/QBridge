package config

// Storer is the interface the cmd layer depends on.
// Using an interface allows commands to be unit-tested with a mock store.
type Storer interface {
	GetDB(name string) (*Database, error)
	AddDB(db Database) error
	RemoveDB(name string) error

	GetProfile(name string) (*Profile, error)
	AddProfile(p Profile) error
	RemoveProfile(name string) error
	UpdateProfile(p Profile) error

	GetDatabases() []Database
	GetProfiles() []Profile

	Save() error
}

// GetDatabases returns all registered databases.
func (s *Store) GetDatabases() []Database {
	return s.Config.Databases
}

// GetProfiles returns all defined profiles.
func (s *Store) GetProfiles() []Profile {
	return s.Config.Profiles
}
