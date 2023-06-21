package lib

import (
	"bufio"
	"encoding/binary"
)

func ReadAttach(reader *bufio.Reader) (chan []byte, chan error) {
	output := make(chan []byte)
	erCh := make(chan error, 1)
	go func() {
	logReader:
		for {
			outputType := make([]byte, 1)
			_, err := reader.Read(outputType)
			if err != nil {
				erCh <- err
				close(output)
				break logReader
			}
			_, err = reader.Discard(3)
			if err != nil {
				erCh <- err
				close(output)
				break logReader
			}
			var length uint32
			err = binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				erCh <- err
				close(output)
				break logReader
			}
			outputData := make([]byte, length)
			n, err := reader.Read(outputData)
			if err != nil {
				erCh <- err
				close(output)
				break logReader
			}
			output <- outputData[:n]
		}
	}()
	return output, erCh
}
