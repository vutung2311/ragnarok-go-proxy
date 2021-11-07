package main

import (
	"bytes"
	"encoding/binary"
)

type Message struct {
	ID   int16
	Body []byte
}

func (m *Message) Read(in []byte, size int) error {
	err := binary.Read(bytes.NewReader(in), binary.LittleEndian, &m.ID)
	if err != nil {
		return err
	}
	m.Body = make([]byte, size)
	copy(m.Body, in)
	return nil
}

func (m *Message) Get(offset int, pointer interface{}) error {
	return binary.Read(bytes.NewReader(m.Body[offset:]), binary.LittleEndian, pointer)
}

func (m *Message) SetBytes(offset int, data []byte) {
	for i := len(data) - 1; i >= 0; i-- {
		m.Body[offset+i] = data[i]
	}
}

func (m *Message) SetUint32(offset int, data uint32) {
	binary.LittleEndian.PutUint32(m.Body[offset:], data)
}

func (m *Message) SetUint16(offset int, data uint16) {
	binary.LittleEndian.PutUint16(m.Body[offset:], data)
}
