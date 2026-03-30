package connection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mygui/backend/internal/connection"
	"mygui/backend/internal/security"
	"mygui/backend/internal/storage"
	"mygui/backend/types"
)

// Manager manages MongoDB connection profiles and active connections.
type Manager struct {
	clients   map[string]*mongo.Client
	tunnels   map[string]*connection.SSHTunnel
	storage   *storage.ConfigStorage
	encryptor *security.Encryptor
	mu        sync.RWMutex
}

// NewManager creates a new MongoDB connection Manager.
func NewManager(storage *storage.ConfigStorage, encryptor *security.Encryptor) *Manager {
	return &Manager{
		clients:   make(map[string]*mongo.Client),
		tunnels:   make(map[string]*connection.SSHTunnel),
		storage:   storage,
		encryptor: encryptor,
	}
}

// CreateProfile creates a new MongoDB connection profile.
// It generates a UUID, encrypts passwords, and persists the profile.
func (m *Manager) CreateProfile(profile types.MongoConnectionProfile) error {
	// Generate ID
	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}

	// Set defaults
	if profile.Timeout == 0 {
		profile.Timeout = 10
	}
	if profile.AuthDB == "" {
		profile.AuthDB = "admin"
	}

	// Encrypt password
	if profile.Password != "" {
		encrypted, err := m.encryptor.Encrypt(profile.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		profile.Password = encrypted
	}

	// Encrypt SSH password
	if profile.SSHPassword != "" {
		encrypted, err := m.encryptor.Encrypt(profile.SSHPassword)
		if err != nil {
			return fmt.Errorf("failed to encrypt SSH password: %w", err)
		}
		profile.SSHPassword = encrypted
	}

	if err := m.storage.SaveMongoProfile(profile); err != nil {
		return fmt.Errorf("failed to save mongo profile: %w", err)
	}

	return nil
}

// UpdateProfile updates an existing MongoDB connection profile.
// It verifies the profile exists, encrypts changed passwords, and persists.
func (m *Manager) UpdateProfile(id string, profile types.MongoConnectionProfile) error {
	existing, err := m.storage.GetMongoProfile(id)
	if err != nil {
		return fmt.Errorf("profile not found: %w", err)
	}

	// Preserve ID and creation time
	profile.ID = id
	profile.CreatedAt = existing.CreatedAt

	// Set defaults
	if profile.Timeout == 0 {
		profile.Timeout = 10
	}
	if profile.AuthDB == "" {
		profile.AuthDB = "admin"
	}

	// Encrypt password if changed
	if profile.Password != "" && profile.Password != existing.Password {
		encrypted, err := m.encryptor.Encrypt(profile.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		profile.Password = encrypted
	} else if profile.Password == "" {
		// Keep existing encrypted password when not provided
		profile.Password = existing.Password
	}

	// Encrypt SSH password if changed
	if profile.SSHPassword != "" && profile.SSHPassword != existing.SSHPassword {
		encrypted, err := m.encryptor.Encrypt(profile.SSHPassword)
		if err != nil {
			return fmt.Errorf("failed to encrypt SSH password: %w", err)
		}
		profile.SSHPassword = encrypted
	} else if profile.SSHPassword == "" {
		profile.SSHPassword = existing.SSHPassword
	}

	if err := m.storage.SaveMongoProfile(profile); err != nil {
		return fmt.Errorf("failed to update mongo profile: %w", err)
	}

	return nil
}

// DeleteProfile disconnects any active connection and deletes the profile.
func (m *Manager) DeleteProfile(id string) error {
	// Disconnect active connection if any
	m.mu.Lock()
	if client, exists := m.clients[id]; exists {
		_ = client.Disconnect(context.TODO())
		delete(m.clients, id)
	}
	if tunnel, exists := m.tunnels[id]; exists {
		_ = tunnel.Close()
		delete(m.tunnels, id)
	}
	m.mu.Unlock()

	if err := m.storage.DeleteMongoProfile(id); err != nil {
		return fmt.Errorf("failed to delete mongo profile: %w", err)
	}

	return nil
}

// ListProfiles returns all MongoDB connection profiles with decrypted passwords.
func (m *Manager) ListProfiles() ([]types.MongoConnectionProfile, error) {
	profiles, err := m.storage.ListMongoProfiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list mongo profiles: %w", err)
	}

	for i := range profiles {
		if profiles[i].Password != "" {
			decrypted, err := m.encryptor.Decrypt(profiles[i].Password)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt password for profile %s: %w", profiles[i].ID, err)
			}
			profiles[i].Password = decrypted
		}

		if profiles[i].SSHPassword != "" {
			decrypted, err := m.encryptor.Decrypt(profiles[i].SSHPassword)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt SSH password for profile %s: %w", profiles[i].ID, err)
			}
			profiles[i].SSHPassword = decrypted
		}
	}

	return profiles, nil
}

// buildConnectionURI constructs a MongoDB connection URI from a profile.
// If SSH tunnel is active, localAddr replaces the original host:port.
func buildConnectionURI(profile types.MongoConnectionProfile, localAddr string) string {
	if profile.UseURI {
		return profile.URI
	}

	host := profile.Host
	port := profile.Port
	if localAddr != "" {
		host = "127.0.0.1"
		// parse port from localAddr
		var p int
		fmt.Sscanf(localAddr, "127.0.0.1:%d", &p)
		if p > 0 {
			port = p
		}
	}

	authDB := profile.AuthDB
	if authDB == "" {
		authDB = "admin"
	}

	if profile.Username != "" {
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
			profile.Username, profile.Password, host, port, authDB)
	}
	return fmt.Sprintf("mongodb://%s:%d/%s", host, port, authDB)
}

// TestConnection tests a MongoDB connection profile without persisting it.
// It supports SSH tunnels and both direct and URI connection modes.
// Returns server version info on success.
func (m *Manager) TestConnection(profile types.MongoConnectionProfile) (*types.MongoTestResult, error) {
	timeout := profile.Timeout
	if timeout <= 0 {
		timeout = 10
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	var tunnel *connection.SSHTunnel
	localAddr := ""

	if profile.SSHEnabled {
		sshPort := profile.SSHPort
		if sshPort == 0 {
			sshPort = 22
		}
		var err error
		tunnel, err = connection.NewSSHTunnel(connection.SSHTunnelConfig{
			SSHHost:     profile.SSHHost,
			SSHPort:     sshPort,
			SSHUsername: profile.SSHUsername,
			SSHPassword: profile.SSHPassword,
			SSHKeyPath:  profile.SSHKeyPath,
			RemoteHost:  profile.Host,
			RemotePort:  profile.Port,
			Timeout:     timeout,
		})
		if err != nil {
			return &types.MongoTestResult{
				Success: false,
				Message: "SSH tunnel failed",
				Error:   err.Error(),
			}, nil
		}
		defer tunnel.Close()
		localAddr = tunnel.GetLocalAddr()
	}

	uri := buildConnectionURI(profile, localAddr)

	clientOpts := options.Client().ApplyURI(uri).
		SetConnectTimeout(time.Duration(timeout) * time.Second).
		SetServerSelectionTimeout(time.Duration(timeout) * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return &types.MongoTestResult{
			Success: false,
			Message: "Connection failed",
			Error:   err.Error(),
		}, nil
	}
	defer client.Disconnect(context.Background())

	// Ping to verify connectivity
	if err := client.Ping(ctx, nil); err != nil {
		return &types.MongoTestResult{
			Success: false,
			Message: "Ping failed",
			Error:   err.Error(),
		}, nil
	}

	// Get server version via buildInfo
	var buildInfo bson.M
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&buildInfo); err != nil {
		return &types.MongoTestResult{
			Success: false,
			Message: "Failed to get server info",
			Error:   err.Error(),
		}, nil
	}

	version, _ := buildInfo["version"].(string)

	return &types.MongoTestResult{
		Success:       true,
		Message:       "Connection successful",
		ServerVersion: version,
	}, nil
}

// Connect establishes a persistent MongoDB connection for the given profile ID.
// It reads the profile from storage, decrypts passwords, sets up an SSH tunnel
// if needed, and stores the active client.
func (m *Manager) Connect(profileID string) (*mongo.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return existing client if already connected
	if client, exists := m.clients[profileID]; exists {
		return client, nil
	}

	profile, err := m.storage.GetMongoProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("profile not found: %w", err)
	}

	// Decrypt password
	if profile.Password != "" {
		decrypted, err := m.encryptor.Decrypt(profile.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password: %w", err)
		}
		profile.Password = decrypted
	}

	// Decrypt SSH password
	if profile.SSHPassword != "" {
		decrypted, err := m.encryptor.Decrypt(profile.SSHPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt SSH password: %w", err)
		}
		profile.SSHPassword = decrypted
	}

	timeout := profile.Timeout
	if timeout <= 0 {
		timeout = 10
	}

	localAddr := ""

	if profile.SSHEnabled {
		sshPort := profile.SSHPort
		if sshPort == 0 {
			sshPort = 22
		}
		tunnel, err := connection.NewSSHTunnel(connection.SSHTunnelConfig{
			SSHHost:     profile.SSHHost,
			SSHPort:     sshPort,
			SSHUsername: profile.SSHUsername,
			SSHPassword: profile.SSHPassword,
			SSHKeyPath:  profile.SSHKeyPath,
			RemoteHost:  profile.Host,
			RemotePort:  profile.Port,
			Timeout:     timeout,
		})
		if err != nil {
			return nil, fmt.Errorf("SSH tunnel failed: %w", err)
		}
		m.tunnels[profileID] = tunnel
		localAddr = tunnel.GetLocalAddr()
	}

	uri := buildConnectionURI(*profile, localAddr)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri).
		SetConnectTimeout(time.Duration(timeout) * time.Second).
		SetServerSelectionTimeout(time.Duration(timeout) * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		if tunnel, ok := m.tunnels[profileID]; ok {
			tunnel.Close()
			delete(m.tunnels, profileID)
		}
		return nil, fmt.Errorf("mongo connect failed: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(context.Background())
		if tunnel, ok := m.tunnels[profileID]; ok {
			tunnel.Close()
			delete(m.tunnels, profileID)
		}
		return nil, fmt.Errorf("mongo ping failed: %w", err)
	}

	m.clients[profileID] = client
	return client, nil
}

// Disconnect closes the MongoDB client and SSH tunnel for the given profile ID.
func (m *Manager) Disconnect(profileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[profileID]
	if !exists {
		return nil
	}

	if err := client.Disconnect(context.Background()); err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}
	delete(m.clients, profileID)

	if tunnel, ok := m.tunnels[profileID]; ok {
		tunnel.Close()
		delete(m.tunnels, profileID)
	}

	return nil
}

// GetClient returns the active MongoDB client for the given profile ID.
// Returns an error if no active connection exists.
func (m *Manager) GetClient(profileID string) (*mongo.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[profileID]
	if !exists {
		return nil, fmt.Errorf("no active connection for profile %s", profileID)
	}
	return client, nil
}

// DisconnectAll closes all active MongoDB clients and SSH tunnels.
func (m *Manager) DisconnectAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for id, client := range m.clients {
		if err := client.Disconnect(context.Background()); err != nil {
			lastErr = err
		}
		delete(m.clients, id)
	}

	for id, tunnel := range m.tunnels {
		if err := tunnel.Close(); err != nil {
			lastErr = err
		}
		delete(m.tunnels, id)
	}

	return lastErr
}
