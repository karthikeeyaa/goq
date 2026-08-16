package logstore

func (lf *LogFile) Append(payload any) (int64, error) {
	return 0, nil
}

func (lf *LogFile) Read(offset int64, limit int) ([]byte, error) {
	return nil, nil
}

func (lf *LogFile) Close() error {
	lf.mu.Lock()
	defer lf.mu.Unlock()

	if lf.File != nil {
		return lf.File.Close()
	}

	return nil
}
