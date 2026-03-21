package database

type UserDevice struct {
	UserID     string `json:"user_id"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Secret     string `json:"secret,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

func (d *Database) SaveDeviceSecret(userID, deviceName, secret string) (string, error) {
	query := `
	INSERT INTO user_devices (device_name, user_id, secret, updated_at)
	VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
	RETURNING device_id;
	`
	var deviceID string
	err := d.db.QueryRow(query, deviceName, userID, secret).Scan(&deviceID)
	if err != nil {
		return "", err
	}
	return deviceID, nil
}

func (d *Database) UpdateDeviceSecret(deviceID, secret string) error {
	query := `
	UPDATE user_devices 
	SET secret = $1, updated_at = CURRENT_TIMESTAMP
	WHERE device_id = $2;
	`
	_, err := d.db.Exec(query, secret, deviceID)
	return err
}

func (d *Database) GetDeviceSecret(deviceID string) (string, error) {
	var secret string
	err := d.db.QueryRow("SELECT secret FROM user_devices WHERE device_id = $1", deviceID).Scan(&secret)
	if err != nil {
		return "", err
	}
	return secret, nil
}

func (d *Database) DeleteDevice(userID, deviceID string) error {
	query := `DELETE FROM user_devices WHERE user_id = $1 AND device_id = $2`
	_, err := d.db.Exec(query, userID, deviceID)
	return err
}
