package storage

import (
	"mygui/backend/types"
	"sync"

	"gopkg.in/yaml.v3"
)

// ConnectionsStorage is a legacy storage implementation (superseded by internal/storage).
// Kept for reference only; not used by the application.
type ConnectionsStorage struct {
	stroage *LocalStorage
	mutex   sync.Mutex
}

func NewConnectionsStorage() *ConnectionsStorage {
	return &ConnectionsStorage{
		stroage: NewLocalStorage("connections.yaml"),
	}
}

func (c *ConnectionsStorage) DefaultConnections() []types.ConnectionProfile {
	return []types.ConnectionProfile{}
}

// GetConnections retrieves all connection profiles.
func (c *ConnectionsStorage) GetConnections() (ret []types.ConnectionProfile) {
	conf, err := c.stroage.Load()
	ret = c.DefaultConnections()
	if err != nil {
		return
	}

	if err = yaml.Unmarshal(conf, &ret); err != nil {
		ret = c.DefaultConnections()
		return
	}

	if len(ret) <= 0 {
		ret = c.DefaultConnections()
	}
	return
}

// GetConnectionByName retrieves a connection profile by name.
func (c *ConnectionsStorage) GetConnectionByName(name string) types.ConnectionProfile {
	connections := c.GetConnections()
	for _, connection := range connections {
		if connection.Name == name {
			return connection
		}
	}
	return types.ConnectionProfile{}
}

// SaveConnections saves all connection profiles.
func (c *ConnectionsStorage) SaveConnections(connections []types.ConnectionProfile) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	conf, err := yaml.Marshal(connections)
	if err != nil {
		return err
	}
	return c.stroage.Save(conf)
}

// AddConnections adds a new connection profile.
func (c *ConnectionsStorage) AddConnections(conn types.ConnectionProfile) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	conf, err := yaml.Marshal(conn)
	if err != nil {
		return err
	}
	return c.stroage.Save(conf)
}
