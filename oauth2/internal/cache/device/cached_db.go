package device

import (
	"github.com/lucap9056/auth-middleware/database"
	"golang.org/x/sync/singleflight"
)

type CachedDB struct {
	*database.Database
	cache       SecretCache
	secretLoads singleflight.Group
}

func NewCachedDB(db *database.Database, cache SecretCache) *CachedDB {
	return &CachedDB{Database: db, cache: cache}
}

func (c *CachedDB) GetDeviceSecret(deviceID string) (string, error) {
	if secret, ok := c.cache.GetSecret(deviceID); ok {
		return secret, nil
	}

	v, err, _ := c.secretLoads.Do(deviceID, func() (any, error) {
		secret, err := c.Database.GetDeviceSecret(deviceID)
		if err != nil {
			return "", err
		}
		c.cache.SetSecret(deviceID, secret, false)
		return secret, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (c *CachedDB) UpdateDeviceSecret(deviceID, secret string) error {
	if err := c.Database.UpdateDeviceSecret(deviceID, secret); err != nil {
		return err
	}
	c.cache.SetSecret(deviceID, secret, true)
	return nil
}

func (c *CachedDB) SaveDeviceSecret(userID, deviceName, secret string) (string, error) {
	deviceID, err := c.Database.SaveDeviceSecret(userID, deviceName, secret)
	if err != nil {
		return "", err
	}
	c.cache.SetSecret(deviceID, secret, true)
	return deviceID, nil
}

func (c *CachedDB) DeleteDevice(userID, deviceID string) error {
	if err := c.Database.DeleteDevice(userID, deviceID); err != nil {
		return err
	}
	c.cache.DeleteSecret(deviceID)
	return nil
}

func (c *CachedDB) DeleteAllDevices(userID string) error {
	ids, err := c.Database.DeleteAllDevicesReturningIDs(userID)
	if err != nil {
		return err
	}

	for _, deviceID := range ids {
		c.cache.DeleteSecret(deviceID)
	}
	return nil
}
