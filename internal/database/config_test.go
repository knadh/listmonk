package database

import (
	"testing"
	"time"
)

func TestConfigValidatePostgreSQL(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid PostgreSQL config",
			config: Config{
				Type:     PostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
			},
			expectErr: false,
		},
		{
			name: "missing host",
			config: Config{
				Type:     PostgreSQL,
				Database: "testdb",
				Username: "testuser",
			},
			expectErr: true,
			errMsg:    "PostgreSQL host is required",
		},
		{
			name: "missing database",
			config: Config{
				Type:     PostgreSQL,
				Host:     "localhost",
				Username: "testuser",
			},
			expectErr: true,
			errMsg:    "PostgreSQL database name is required",
		},
		{
			name: "missing username",
			config: Config{
				Type:     PostgreSQL,
				Host:     "localhost",
				Database: "testdb",
			},
			expectErr: true,
			errMsg:    "PostgreSQL username is required",
		},
		{
			name: "zero port sets default",
			config: Config{
				Type:     PostgreSQL,
				Host:     "localhost",
				Port:     0,
				Database: "testdb",
				Username: "testuser",
			},
			expectErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if err.Error() != tt.errMsg {
					t.Errorf("Expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				
				// Check defaults were set
				if tt.config.Port == 0 && tt.config.Port != 5432 {
					t.Error("Expected default port 5432 to be set")
				}
			}
		})
	}
}

func TestConfigValidateMongoDB(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid MongoDB config with URI",
			config: Config{
				Type: MongoDB,
				MongoDB: MongoDBConfig{
					URI: "mongodb://localhost:27017/testdb",
				},
			},
			expectErr: false,
		},
		{
			name: "valid MongoDB config without URI",
			config: Config{
				Type:     MongoDB,
				Host:     "localhost",
				Port:     27017,
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
			},
			expectErr: false,
		},
		{
			name: "missing host and URI",
			config: Config{
				Type:     MongoDB,
				Database: "testdb",
			},
			expectErr: true,
			errMsg:    "MongoDB host or URI is required",
		},
		{
			name: "missing database without URI",
			config: Config{
				Type: MongoDB,
				Host: "localhost",
			},
			expectErr: true,
			errMsg:    "MongoDB database name is required",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if err.Error() != tt.errMsg {
					t.Errorf("Expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigSetDefaults(t *testing.T) {
	config := Config{}
	config.SetDefaults()
	
	if config.MaxOpenConnections != 25 {
		t.Errorf("Expected MaxOpenConnections 25, got %d", config.MaxOpenConnections)
	}
	
	if config.MaxIdleConnections != 25 {
		t.Errorf("Expected MaxIdleConnections 25, got %d", config.MaxIdleConnections)
	}
	
	if config.ConnectionMaxLifetime != 5*time.Minute {
		t.Errorf("Expected ConnectionMaxLifetime 5m, got %v", config.ConnectionMaxLifetime)
	}
	
	if config.ConnectionTimeout != 30*time.Second {
		t.Errorf("Expected ConnectionTimeout 30s, got %v", config.ConnectionTimeout)
	}
}

func TestGetPostgreSQLDSN(t *testing.T) {
	config := Config{
		Type:     PostgreSQL,
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
		PostgreSQL: PostgreSQLConfig{
			SSLMode:         "require",
			ApplicationName: "listmonk",
			ConnectTimeout:  30 * time.Second,
		},
	}
	
	dsn := config.GetPostgreSQLDSN()
	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=require application_name=listmonk connect_timeout=30"
	
	if dsn != expected {
		t.Errorf("Expected DSN %q, got %q", expected, dsn)
	}
}

func TestGetMongoDBURI(t *testing.T) {
	config := Config{
		Type: MongoDB,
		MongoDB: MongoDBConfig{
			URI: "mongodb://localhost:27017/testdb",
		},
	}
	
	uri := config.GetMongoDBURI()
	expected := "mongodb://localhost:27017/testdb"
	
	if uri != expected {
		t.Errorf("Expected URI %q, got %q", expected, uri)
	}
}

func TestValidateUnsupportedType(t *testing.T) {
	config := Config{
		Type: DatabaseType("unsupported"),
	}
	
	err := config.Validate()
	if err == nil {
		t.Error("Expected error for unsupported database type")
	}
	
	expectedMsg := "unsupported database type: unsupported"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestValidateEmptyType(t *testing.T) {
	config := Config{}
	
	err := config.Validate()
	if err == nil {
		t.Error("Expected error for empty database type")
	}
	
	expectedMsg := "database type is required"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}