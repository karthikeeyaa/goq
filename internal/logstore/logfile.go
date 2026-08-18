package logstore

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"time"
)

func (lf *LogFile) Append(payload any) (LogMessage, error) {
	lf.mu.Lock()
	defer lf.mu.Unlock()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return LogMessage{}, err
	}

	payloadLength := len(payloadBytes)
	if payloadLength == 0 {
		return LogMessage{}, fmt.Errorf("payload cannot be empty")
	}
	if lf.Topic.MaxMessageBytes.Valid && payloadLength > int(lf.Topic.MaxMessageBytes.Int64) {
		return LogMessage{}, fmt.Errorf("payload size exceeds max message bytes limit")
	}

	startOffset, err := lf.File.Seek(0, io.SeekEnd)
	if err != nil {
		return LogMessage{}, err
	}

	timestampMS := time.Now().UnixNano() / int64(time.Millisecond)
	checksum := crc32.ChecksumIEEE(payloadBytes)

	logData := make([]byte, 4+4+8+payloadLength)
	binary.LittleEndian.PutUint32(logData[0:4], checksum)
	binary.LittleEndian.PutUint32(logData[4:8], uint32(payloadLength))
	binary.LittleEndian.PutUint64(logData[8:16], uint64(timestampMS))
	copy(logData[16:], payloadBytes)

	if _, err = lf.File.Write(logData); err != nil {
		return LogMessage{}, err
	}

	indexFileInfo, err := lf.IndexFile.Stat()
	if err != nil {
		return LogMessage{}, err
	}
	logicalOffset := int(indexFileInfo.Size() / 12)

	indexData := make([]byte, 12)
	binary.LittleEndian.PutUint64(indexData[0:8], uint64(startOffset))
	binary.LittleEndian.PutUint32(indexData[8:12], uint32(payloadLength))

	if _, err = lf.IndexFile.Write(indexData); err != nil {
		return LogMessage{}, err
	}

	if _, err = lf.File.Seek(0, io.SeekEnd); err != nil {
		return LogMessage{}, err
	}
	if _, err = lf.IndexFile.Seek(0, io.SeekEnd); err != nil {
		return LogMessage{}, err
	}

	return LogMessage{
		Offset:      logicalOffset,
		TimestampMS: timestampMS,
		Payload:     payloadBytes,
	}, nil
}

func (lf *LogFile) Read(offset int, limit int) ([]LogMessage, int, error) {
	lf.mu.Lock()
	defer lf.mu.Unlock()

	indexInfo, err := lf.IndexFile.Stat()
	if err != nil {
		return nil, 0, err
	}
	highWatermark := int(indexInfo.Size() / 12)
	if offset >= highWatermark {
		return nil, highWatermark, nil
	}

	if limit > highWatermark-offset {
		limit = highWatermark - offset
	}

	messages := make([]LogMessage, 0, limit)
	for i := offset; i < offset+limit; i++ {
		indexEntry := make([]byte, 12)
		if _, err = lf.IndexFile.Seek(int64(i*12), io.SeekStart); err != nil {
			return nil, highWatermark, err
		}
		if _, err = io.ReadFull(lf.IndexFile, indexEntry); err != nil {
			return nil, highWatermark, err
		}

		absolutePosition := binary.LittleEndian.Uint64(indexEntry[0:8])
		payloadLength := int(binary.LittleEndian.Uint32(indexEntry[8:12]))

		if _, err = lf.File.Seek(int64(absolutePosition), io.SeekStart); err != nil {
			return nil, highWatermark, err
		}
		header := make([]byte, 16)
		if _, err = io.ReadFull(lf.File, header); err != nil {
			return nil, highWatermark, err
		}
		payload := make([]byte, payloadLength)
		if _, err = io.ReadFull(lf.File, payload); err != nil {
			return nil, highWatermark, err
		}

		timestampMS := int64(binary.LittleEndian.Uint64(header[8:16]))
		messages = append(messages, LogMessage{
			Offset:      i,
			TimestampMS: timestampMS,
			Payload:     json.RawMessage(payload),
		})
	}

	if _, err = lf.File.Seek(0, io.SeekEnd); err != nil {
		return nil, highWatermark, err
	}
	if _, err = lf.IndexFile.Seek(0, io.SeekEnd); err != nil {
		return nil, highWatermark, err
	}

	return messages, highWatermark, nil
}

func (lf *LogFile) Close() error {
	lf.mu.Lock()
	defer lf.mu.Unlock()

	var closeErr error
	if lf.File != nil {
		if err := lf.File.Close(); err != nil {
			closeErr = err
		}
		lf.File = nil
	}
	if lf.IndexFile != nil {
		if err := lf.IndexFile.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
		lf.IndexFile = nil
	}
	return closeErr
}
