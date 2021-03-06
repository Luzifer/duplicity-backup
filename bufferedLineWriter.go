package main

import "strings"

type messageChanWriter struct {
	msgChan chan string

	buffer []byte
}

func newMessageChanWriter(outputChannel chan string) *messageChanWriter {
	return &messageChanWriter{
		msgChan: outputChannel,
		buffer:  []byte{},
	}
}

func (m *messageChanWriter) Write(p []byte) (int, error) {
	var (
		n   = len(p)
		err error
	)

	m.buffer = append(m.buffer, p...)
	if strings.Contains(string(m.buffer), "\n") {
		lines := strings.Split(string(m.buffer), "\n")
		for _, l := range lines[:len(lines)-1] {
			m.msgChan <- l
		}
		m.buffer = []byte(lines[len(lines)-1])
	}

	return n, err
}
