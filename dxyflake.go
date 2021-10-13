// Package dxyflake implements dxyflake, duoxieyun distributed unique ID generator inspired by Twitter's Snowflake.
//
// +---------------------------------------------------------------------------------------+
// | 1 Bit Unused | 41 Bit Timestamp | 5 Bit NodeID | 5 Bit ServiceID | 12 Bit Sequence ID |
// +---------------------------------------------------------------------------------------+
//
// 41 bits for time in units of 10 msec (697 years)
//  5 bits for a machine id (32 nodes)
//  5 bits for a service id (32 services per node)
// 12 bits for a sequence number (0 ~ 4095)
package dxyflake

import (
	"errors"
	"sync"
	"time"
)

// These constants are the bit lengths of dxyflake ID parts.
const (
	BitLenTime      = 41 // bit length of time
	BitLenMachineID = 5  // bit length of machineID
	BitLenServiceID = 5  // bit length of serviceID
	BitLenSequence  = 12 // bit length of sequence number
)

// Settings configures dxyflake:
//
// StartTime is the time since which the dxyflake time is defined as the elapsed time.
// If StartTime is 0, the start time of the dxyflake is set to "2021-10-01 00:00:00 +0000 UTC".
// If StartTime is ahead of the current time, dxyflake is not created.
//
// MachineID returns the unique ID of the dxyflake instance.
// If MachineID returns an error, dxyflake is not created.
// If MachineID is nil, default MachineID(0) is used.
//
// ServiceID returns the unique ID of the dxyflake service per machine.
// If ServiceID returns an error, dxyflake is not created.
// If ServiceID is nil, default ServiceID(0) is used.
//
// CheckMachineID validates the uniqueness of the machine ID.
// If CheckMachineID returns false, dxyflake is not created.
// If CheckMachineID is nil, no validation is done.
//
// CheckServiceID validates the uniqueness of the service ID.
// If CheckServiceID returns false, dxyflake is not created.
// If CheckServiceID is nil, no validation is done.
type Settings struct {
	StartTime      time.Time
	MachineID      func() (uint16, error)
	ServiceID      func() (uint16, error)
	CheckMachineID func(uint16) bool
	CheckServiceID func(uint16) bool
}

// Init set default MachineID & ServiceID
func (s *Settings) Init(mID, sID uint16) {
	if s != nil {
		s.MachineID = func() (uint16, error) {
			return mID, nil
		}
		s.ServiceID = func() (uint16, error) {
			return sID, nil
		}
	}
}

// StartTimeSet set start time
func (s *Settings) StartTimeSet(t time.Time) {
	if s != nil {
		s.StartTime = t
	}
}

// dxyflake is a distributed unique ID generator.
type dxyflake struct {
	mutex       *sync.Mutex
	startTime   int64
	elapsedTime int64
	machineID   uint16
	serviceID   uint16
	sequence    uint16
}

// NewDxyflake returns a new dxyflake configured with the given Settings.
// NewDxyflake returns nil in the following cases:
// - Settings.StartTime is ahead of the current time.
// - Settings.MachineID returns an error.
// - Settings.ServiceID returns an error.
// - Settings.CheckMachineID returns false.
// - Settings.CheckServiceID returns false.
func NewDxyflake(st Settings) *dxyflake {
	df := new(dxyflake)
	df.mutex = new(sync.Mutex)
	df.sequence = uint16(1<<BitLenSequence - 1)

	if st.StartTime.After(time.Now()) {
		return nil
	}
	if st.StartTime.IsZero() {
		df.startTime = toDxyflakeTime(time.Date(2021, 10, 1, 0, 0, 0, 0, time.UTC))
	} else {
		df.startTime = toDxyflakeTime(st.StartTime)
	}

	var err error
	if st.MachineID == nil {
		df.machineID = 0
	} else {
		df.machineID, err = st.MachineID()
	}
	if st.ServiceID == nil {
		df.serviceID = 0
	} else {
		df.serviceID, err = st.ServiceID()
	}
	if err != nil ||
		(st.CheckMachineID != nil && !st.CheckMachineID(df.machineID)) ||
		(st.CheckServiceID != nil && !st.CheckServiceID(df.serviceID)) {
		return nil
	}

	return df
}

// NextID generates a next unique ID.
// After the dxyflake time overflows, NextID returns an error.
func (sf *dxyflake) NextID() (uint64, error) {
	const maskSequence = uint16(1<<BitLenSequence - 1)

	sf.mutex.Lock()
	defer sf.mutex.Unlock()

	current := currentElapsedTime(sf.startTime)
	if sf.elapsedTime < current {
		sf.elapsedTime = current
		sf.sequence = 0
	} else { // sf.elapsedTime >= current
		sf.sequence = (sf.sequence + 1) & maskSequence
		if sf.sequence == 0 { // overflow
			sf.elapsedTime++
			overtime := sf.elapsedTime - current
			time.Sleep(sleepTime((overtime)))
		}
	}

	return sf.toID()
}

const dxyflakeTimeUnit = 1e7 // nsec, i.e. 10 msec

func toDxyflakeTime(t time.Time) int64 {
	return t.UTC().UnixNano() / dxyflakeTimeUnit
}

func currentElapsedTime(startTime int64) int64 {
	return toDxyflakeTime(time.Now()) - startTime
}

func sleepTime(overtime int64) time.Duration {
	return time.Duration(overtime)*10*time.Millisecond -
		time.Duration(time.Now().UTC().UnixNano()%dxyflakeTimeUnit)*time.Nanosecond
}

func (sf *dxyflake) toID() (uint64, error) {
	if sf.elapsedTime >= 1<<BitLenTime {
		return 0, errors.New("over the time limit")
	}

	return uint64(sf.elapsedTime)<<(BitLenMachineID+BitLenServiceID+BitLenSequence) |
		uint64(sf.machineID)<<(BitLenServiceID+BitLenSequence) |
		uint64(sf.serviceID)<<BitLenSequence |
		uint64(sf.sequence), nil
}

// Decompose returns a set of dxyflake ID parts.
func Decompose(id uint64) map[string]uint64 {
	const maskMachineID = uint64((1<<BitLenMachineID - 1) << (BitLenServiceID + BitLenSequence))
	const maskServiceID = uint64((1<<BitLenServiceID - 1) << BitLenSequence)
	const maskSequence = uint64(1<<BitLenSequence - 1)

	msb := id >> 63
	time := id >> (BitLenMachineID + BitLenServiceID + BitLenSequence)
	machineID := (id & maskMachineID) >> (BitLenServiceID + BitLenSequence)
	serviceID := (id & maskServiceID) >> BitLenSequence
	sequence := (id & maskSequence)
	return map[string]uint64{
		"id":         id,
		"msb":        msb,
		"time":       time,
		"machine-id": machineID,
		"service-id": serviceID,
		"sequence":   sequence,
	}
}
