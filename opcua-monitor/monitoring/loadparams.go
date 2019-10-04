package monitoring

import (
	"bufio"
	"os"
)

// LoadParametersFromFile reads the file lines and loads
// NodeIDs of the parameters we need to monitor on the server.
//
// Remark: it's impossible to browse parameters on the OPC UA
// server with gopcua/opcua library, so I decided to read them from
// a file.
func LoadParametersFromFile(filePath string) ([]string, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)

	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := make([]string, 0)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, nil
}
