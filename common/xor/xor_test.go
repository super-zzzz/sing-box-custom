package xor

import (
	"bytes"
	"io"
	"net"
	"testing"
)

func TestBytesLength(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	Bytes(data, 0x0F, 2)
	if expected := []byte{0x0E, 0x0D, 0x03, 0x04}; !bytes.Equal(data, expected) {
		t.Fatalf("unexpected limited xor result: %x", data)
	}
}

func TestStreamConnLengthAcrossChunks(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	conn := NewStreamConn(client, 0x0F, 3)
	done := make(chan error, 1)
	go func() {
		_, err := conn.Write([]byte{0x01, 0x02})
		if err == nil {
			_, err = conn.Write([]byte{0x03, 0x04})
		}
		done <- err
	}()
	encoded := make([]byte, 4)
	_, err := io.ReadFull(server, encoded)
	if err != nil {
		t.Fatal(err)
	}
	if expected := []byte{0x0E, 0x0D, 0x0C, 0x04}; !bytes.Equal(encoded, expected) {
		t.Fatalf("unexpected stream xor result: %x", encoded)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestDatagramConnLengthPerWrite(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	conn := NewDatagramConn(client, 0x0F, 2)
	done := make(chan error, 1)
	go func() {
		_, err := conn.Write([]byte{0x01, 0x02, 0x03})
		if err == nil {
			_, err = conn.Write([]byte{0x04, 0x05, 0x06})
		}
		done <- err
	}()
	encoded := make([]byte, 6)
	_, err := io.ReadFull(server, encoded)
	if err != nil {
		t.Fatal(err)
	}
	if expected := []byte{0x0E, 0x0D, 0x03, 0x0B, 0x0A, 0x06}; !bytes.Equal(encoded, expected) {
		t.Fatalf("unexpected datagram xor result: %x", encoded)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
