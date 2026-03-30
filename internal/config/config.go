package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDir  = ".qbridge"
	storeFile  = "store.enc"
	markerFile = ".initialized"
)

// DBType represents a supported database engine.
type DBType string

const (
	DBTypePostgres DBType = "postgres"
	DBTypeMySQL    DBType = "mysql"
	DBTypeSQLite   DBType = "sqlite"
)

// Operation represents an allowed SQL operation class.
type Operation string

const (
	OpSelect Operation = "SELECT"
	OpInsert Operation = "INSERT"
	OpUpdate Operation = "UPDATE"
	OpDelete Operation = "DELETE"
	OpDDL    Operation = "DDL"
)

// AllOperations contains all valid operation values.
var AllOperations = []Operation{OpSelect, OpInsert, OpUpdate, OpDelete, OpDDL}

// Database holds connection details for a registered database.
type Database struct {
	Name     string `json:"name"`
	Type     DBType `json:"type"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	DBName   string `json:"dbname,omitempty"`
	FilePath string `json:"filepath,omitempty"` // SQLite only
	SSLMode  string `json:"sslmode,omitempty"`  // Postgres only
}

// Profile defines which databases an agent can use and what operations are allowed.
type Profile struct {
	Name      string      `json:"name"`
	Databases []string    `json:"databases"` // references Database.Name
	Allow     []Operation `json:"allow"`
}

// Config is the root structure stored in the encrypted file.
type Config struct {
	Databases []Database `json:"databases"`
	Profiles  []Profile  `json:"profiles"`
}

// Store wraps the config with the master password for load/save operations.
type Store struct {
	path     string
	password string
	Config   Config
}

// QBridgeDir returns the path to the ~/.qbridge directory.
func QBridgeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDir), nil
}

// StorePath returns the full path to the encrypted store file.
func StorePath() (string, error) {
	dir, err := QBridgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, storeFile), nil
}

// IsInitialized returns true if QBridge has been initialised.
func IsInitialized() bool {
	dir, err := QBridgeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, markerFile))
	return err == nil
}

// Initialize creates the ~/.qbridge directory and writes an empty encrypted store.
func Initialize(password string) error {
	dir, err := QBridgeDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	s := &Store{password: password}
	if err := s.save(); err != nil {
		return err
	}

	marker := filepath.Join(dir, markerFile)
	return os.WriteFile(marker, []byte(""), 0600)
}

// Load decrypts and loads the store using the given master password.
func Load(password string) (*Store, error) {
	path, err := StorePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read store: %w", err)
	}

	plain, err := Decrypt(data, password)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(plain, &cfg); err != nil {
		return nil, fmt.Errorf("corrupt store: %w", err)
	}

	if cfg.Databases == nil {
		cfg.Databases = []Database{}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = []Profile{}
	}

	return &Store{path: path, password: password, Config: cfg}, nil
}

// Save encrypts and writes the store to disk.
func (s *Store) Save() error {
	return s.save()
}

func (s *Store) save() error {
	path, err := StorePath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(s.Config)
	if err != nil {
		return err
	}

	enc, err := Encrypt(data, s.password)
	if err != nil {
		return err
	}

	return os.WriteFile(path, enc, 0600)
}

// GetDB returns a database by name, or an error if not found.
func (s *Store) GetDB(name string) (*Database, error) {
	for i := range s.Config.Databases {
		if s.Config.Databases[i].Name == name {
			return &s.Config.Databases[i], nil
		}
	}
	return nil, fmt.Errorf("database %q not found", name)
}

// AddDB adds a new database entry. Returns error if name already exists.
func (s *Store) AddDB(db Database) error {
	for _, d := range s.Config.Databases {
		if d.Name == db.Name {
			return fmt.Errorf("database %q already exists", db.Name)
		}
	}
	s.Config.Databases = append(s.Config.Databases, db)
	return nil
}

// RemoveDB removes a database by name.
func (s *Store) RemoveDB(name string) error {
	for i, d := range s.Config.Databases {
		if d.Name == name {
			s.Config.Databases = append(s.Config.Databases[:i], s.Config.Databases[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("database %q not found", name)
}

// GetProfile returns a profile by name.
func (s *Store) GetProfile(name string) (*Profile, error) {
	for i := range s.Config.Profiles {
		if s.Config.Profiles[i].Name == name {
			return &s.Config.Profiles[i], nil
		}
	}
	return nil, fmt.Errorf("profile %q not found", name)
}

// AddProfile adds a new profile. Returns error if name already exists.
func (s *Store) AddProfile(p Profile) error {
	for _, existing := range s.Config.Profiles {
		if existing.Name == p.Name {
			return fmt.Errorf("profile %q already exists", p.Name)
		}
	}
	// Validate that all referenced databases exist.
	for _, dbName := range p.Databases {
		if _, err := s.GetDB(dbName); err != nil {
			return fmt.Errorf("database %q referenced in profile does not exist", dbName)
		}
	}
	s.Config.Profiles = append(s.Config.Profiles, p)
	return nil
}

// RemoveProfile removes a profile by name.
func (s *Store) RemoveProfile(name string) error {
	for i, p := range s.Config.Profiles {
		if p.Name == name {
			s.Config.Profiles = append(s.Config.Profiles[:i], s.Config.Profiles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("profile %q not found", name)
}

// UpdateProfile replaces an existing profile.
func (s *Store) UpdateProfile(p Profile) error {
	for _, dbName := range p.Databases {
		if _, err := s.GetDB(dbName); err != nil {
			return fmt.Errorf("database %q referenced in profile does not exist", dbName)
		}
	}
	for i, existing := range s.Config.Profiles {
		if existing.Name == p.Name {
			s.Config.Profiles[i] = p
			return nil
		}
	}
	return fmt.Errorf("profile %q not found", p.Name)
}

// ValidateOperation returns an error if op is not a recognised operation.
func ValidateOperation(op string) (Operation, error) {
	for _, valid := range AllOperations {
		if Operation(op) == valid {
			return valid, nil
		}
	}
	return "", errors.New("invalid operation: must be one of SELECT, INSERT, UPDATE, DELETE, DDL")
}
