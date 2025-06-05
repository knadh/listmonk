package database

import (
	"fmt"
	"time"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgresql"
	MongoDB    DatabaseType = "mongodb"
)

// Config represents database configuration
type Config struct {
	Type DatabaseType `mapstructure:"type" json:"type"`
	
	// Common connection settings
	Host     string `mapstructure:"host" json:"host"`
	Port     int    `mapstructure:"port" json:"port"`
	Database string `mapstructure:"database" json:"database"`
	Username string `mapstructure:"username" json:"username"`
	Password string `mapstructure:"password" json:"password"`
	
	// PostgreSQL specific settings
	PostgreSQL PostgreSQLConfig `mapstructure:"postgresql" json:"postgresql"`
	
	// MongoDB specific settings
	MongoDB MongoDBConfig `mapstructure:"mongodb" json:"mongodb"`
	
	// Connection pool settings
	MaxOpenConnections    int           `mapstructure:"max_open_connections" json:"max_open_connections"`
	MaxIdleConnections    int           `mapstructure:"max_idle_connections" json:"max_idle_connections"`
	ConnectionMaxLifetime time.Duration `mapstructure:"connection_max_lifetime" json:"connection_max_lifetime"`
	ConnectionTimeout     time.Duration `mapstructure:"connection_timeout" json:"connection_timeout"`
	
	// Additional options for database-specific configurations
	Options map[string]interface{} `mapstructure:"options" json:"options"`
}

// PostgreSQLConfig contains PostgreSQL-specific configuration
type PostgreSQLConfig struct {
	SSLMode           string        `mapstructure:"ssl_mode" json:"ssl_mode"`
	SSLCert           string        `mapstructure:"ssl_cert" json:"ssl_cert"`
	SSLKey            string        `mapstructure:"ssl_key" json:"ssl_key"`
	SSLRootCert       string        `mapstructure:"ssl_root_cert" json:"ssl_root_cert"`
	ConnectTimeout    time.Duration `mapstructure:"connect_timeout" json:"connect_timeout"`
	StatementTimeout  time.Duration `mapstructure:"statement_timeout" json:"statement_timeout"`
	ApplicationName   string        `mapstructure:"application_name" json:"application_name"`
	SearchPath        string        `mapstructure:"search_path" json:"search_path"`
	Timezone          string        `mapstructure:"timezone" json:"timezone"`
}

// MongoDBConfig contains MongoDB-specific configuration
type MongoDBConfig struct {
	URI                 string        `mapstructure:"uri" json:"uri"`
	AuthDatabase        string        `mapstructure:"auth_database" json:"auth_database"`
	ReplicaSet          string        `mapstructure:"replica_set" json:"replica_set"`
	ReadPreference      string        `mapstructure:"read_preference" json:"read_preference"`
	WriteConcern        string        `mapstructure:"write_concern" json:"write_concern"`
	ReadConcern         string        `mapstructure:"read_concern" json:"read_concern"`
	ServerSelectionTimeout time.Duration `mapstructure:"server_selection_timeout" json:"server_selection_timeout"`
	ConnectTimeout      time.Duration `mapstructure:"connect_timeout" json:"connect_timeout"`
	SocketTimeout       time.Duration `mapstructure:"socket_timeout" json:"socket_timeout"`
	TLS                 TLSConfig     `mapstructure:"tls" json:"tls"`
}

// TLSConfig contains TLS configuration for MongoDB
type TLSConfig struct {
	Enabled            bool   `mapstructure:"enabled" json:"enabled"`
	CertificateFile    string `mapstructure:"certificate_file" json:"certificate_file"`
	PrivateKeyFile     string `mapstructure:"private_key_file" json:"private_key_file"`
	CAFile             string `mapstructure:"ca_file" json:"ca_file"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify"`
}

// Validate validates the database configuration
func (c *Config) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("database type is required")
	}
	
	if c.Type != PostgreSQL && c.Type != MongoDB {
		return fmt.Errorf("unsupported database type: %s", c.Type)
	}
	
	switch c.Type {
	case PostgreSQL:
		return c.validatePostgreSQL()
	case MongoDB:
		return c.validateMongoDB()
	}
	
	return nil
}

// validatePostgreSQL validates PostgreSQL-specific configuration
func (c *Config) validatePostgreSQL() error {
	if c.Host == "" {
		return fmt.Errorf("PostgreSQL host is required")
	}
	
	if c.Port <= 0 {
		c.Port = 5432 // Default PostgreSQL port
	}
	
	if c.Database == "" {
		return fmt.Errorf("PostgreSQL database name is required")
	}
	
	if c.Username == "" {
		return fmt.Errorf("PostgreSQL username is required")
	}
	
	// Set defaults for PostgreSQL
	if c.PostgreSQL.SSLMode == "" {
		c.PostgreSQL.SSLMode = "disable"
	}
	
	if c.PostgreSQL.ConnectTimeout == 0 {
		c.PostgreSQL.ConnectTimeout = 30 * time.Second
	}
	
	if c.PostgreSQL.ApplicationName == "" {
		c.PostgreSQL.ApplicationName = "listmonk"
	}
	
	return nil
}

// validateMongoDB validates MongoDB-specific configuration
func (c *Config) validateMongoDB() error {
	if c.MongoDB.URI == "" {
		// Build URI from individual components if not provided
		if c.Host == "" {
			return fmt.Errorf("MongoDB host or URI is required")
		}
		
		if c.Port <= 0 {
			c.Port = 27017 // Default MongoDB port
		}
		
		if c.Database == "" {
			return fmt.Errorf("MongoDB database name is required")
		}
		
		// Build URI
		if c.Username != "" && c.Password != "" {
			c.MongoDB.URI = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s", 
				c.Username, c.Password, c.Host, c.Port, c.Database)
		} else {
			c.MongoDB.URI = fmt.Sprintf("mongodb://%s:%d/%s", 
				c.Host, c.Port, c.Database)
		}
	}
	
	// Set defaults for MongoDB
	if c.MongoDB.ReadPreference == "" {
		c.MongoDB.ReadPreference = "primary"
	}
	
	if c.MongoDB.WriteConcern == "" {
		c.MongoDB.WriteConcern = "majority"
	}
	
	if c.MongoDB.ServerSelectionTimeout == 0 {
		c.MongoDB.ServerSelectionTimeout = 30 * time.Second
	}
	
	if c.MongoDB.ConnectTimeout == 0 {
		c.MongoDB.ConnectTimeout = 30 * time.Second
	}
	
	if c.MongoDB.SocketTimeout == 0 {
		c.MongoDB.SocketTimeout = 30 * time.Second
	}
	
	return nil
}

// SetDefaults sets default values for the configuration
func (c *Config) SetDefaults() {
	if c.MaxOpenConnections == 0 {
		c.MaxOpenConnections = 25
	}
	
	if c.MaxIdleConnections == 0 {
		c.MaxIdleConnections = 25
	}
	
	if c.ConnectionMaxLifetime == 0 {
		c.ConnectionMaxLifetime = 5 * time.Minute
	}
	
	if c.ConnectionTimeout == 0 {
		c.ConnectionTimeout = 30 * time.Second
	}
}

// GetPostgreSQLDSN builds PostgreSQL connection string
func (c *Config) GetPostgreSQLDSN() string {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.Database, c.PostgreSQL.SSLMode)
	
	if c.PostgreSQL.ApplicationName != "" {
		dsn += fmt.Sprintf(" application_name=%s", c.PostgreSQL.ApplicationName)
	}
	
	if c.PostgreSQL.ConnectTimeout > 0 {
		dsn += fmt.Sprintf(" connect_timeout=%d", int(c.PostgreSQL.ConnectTimeout.Seconds()))
	}
	
	if c.PostgreSQL.StatementTimeout > 0 {
		dsn += fmt.Sprintf(" statement_timeout=%d", int(c.PostgreSQL.StatementTimeout.Milliseconds()))
	}
	
	if c.PostgreSQL.SearchPath != "" {
		dsn += fmt.Sprintf(" search_path=%s", c.PostgreSQL.SearchPath)
	}
	
	if c.PostgreSQL.Timezone != "" {
		dsn += fmt.Sprintf(" timezone=%s", c.PostgreSQL.Timezone)
	}
	
	if c.PostgreSQL.SSLCert != "" {
		dsn += fmt.Sprintf(" sslcert=%s", c.PostgreSQL.SSLCert)
	}
	
	if c.PostgreSQL.SSLKey != "" {
		dsn += fmt.Sprintf(" sslkey=%s", c.PostgreSQL.SSLKey)
	}
	
	if c.PostgreSQL.SSLRootCert != "" {
		dsn += fmt.Sprintf(" sslrootcert=%s", c.PostgreSQL.SSLRootCert)
	}
	
	return dsn
}

// GetMongoDBURI returns the MongoDB connection URI
func (c *Config) GetMongoDBURI() string {
	return c.MongoDB.URI
}