// Copyright (c) 2018-2025, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
)

var (
	objectIDCounter = readRandomUint32()
	machineID       = readMachineID()
	processID       = os.Getpid()
)

func readRandomUint32() uint32 {
	var b [4]byte

	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(fmt.Errorf("cannot read random object id: %v", err))
	}

	return ((uint32(b[0]) << 0) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16) | (uint32(b[3]) << 24))
}

func readMachineID() []byte {
	var sum [3]byte

	id := sum[:]

	hostname, err1 := os.Hostname()
	if err1 != nil {
		_, err2 := io.ReadFull(rand.Reader, id)
		if err2 != nil {
			panic(fmt.Errorf("cannot get hostname: %v; %v", err1, err2))
		}

		return id
	}

	hw := md5.New()
	if _, err := hw.Write([]byte(hostname)); err != nil {
		panic(err)
	}

	copy(id, hw.Sum(nil))

	return id
}

func newObjectID() string {
	var b [12]byte
	// Timestamp, 4 bytes, big endian
	binary.BigEndian.PutUint32(b[:], uint32(time.Now().Unix()))
	// Machine, first 3 bytes of md5(hostname)
	b[4] = machineID[0]
	b[5] = machineID[1]
	b[6] = machineID[2]
	// Pid, 2 bytes, specs don't specify endianness, but we use big endian.
	b[7] = byte(processID >> 8)
	b[8] = byte(processID)
	// Increment, 3 bytes, big endian
	i := atomic.AddUint32(&objectIDCounter, 1)
	b[9] = byte(i >> 16)
	b[10] = byte(i >> 8)
	b[11] = byte(i)

	return hex.EncodeToString(b[:])
}
