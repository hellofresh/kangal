package backends

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"
)

var (
	// ErrInvalidCSVFormat when the number of columns is different than two
	ErrInvalidCSVFormat = errors.New("Invalid csv format, expecting: key, value")
)

// ReadEnvs reads data from csv file to save it as a map for creating a secret
func ReadEnvs(envVars string) (map[string]string, error) {
	m := make(map[string]string)
	reader := csv.NewReader(strings.NewReader(envVars))
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(line) != 2 {
			return nil, ErrInvalidCSVFormat
		}
		m[line[0]] = line[1]
	}
	return m, nil
}
