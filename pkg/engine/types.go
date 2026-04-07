package engine

// Engine identifies a database engine type.
type Engine string

const (
	EngineMariaDB    Engine = "mariadb"
	EnginePostgreSQL Engine = "postgresql"
	EngineSQLite     Engine = "sqlite"
)

// ConnectionConfig holds the configuration for a single database connection.
type ConnectionConfig struct {
	Name        string  `yaml:"name" json:"name"`
	Engine      Engine  `yaml:"engine" json:"engine"`
	Host        *string `yaml:"host,omitempty" json:"host,omitempty"`
	Port        *int    `yaml:"port,omitempty" json:"port,omitempty"`
	User        *string `yaml:"user,omitempty" json:"user,omitempty"`
	Password    *string `yaml:"password,omitempty" json:"password,omitempty"`
	PasswordEnv *string `yaml:"password_env,omitempty" json:"password_env,omitempty"`
	Database    *string `yaml:"database,omitempty" json:"database,omitempty"`
	Path        *string `yaml:"path,omitempty" json:"path,omitempty"` // SQLite file path
}

// DatabaseTarget is the (connection, database) product type that serves as the
// fundamental data unit for most operations.
type DatabaseTarget struct {
	Connection string
	Database   string
}

// ForeignKey describes a foreign key relationship between two tables.
type ForeignKey struct {
	Table           string // table that has the foreign key
	ReferencedTable string // table that is referenced
}
