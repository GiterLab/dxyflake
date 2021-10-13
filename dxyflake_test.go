package dxyflake

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

var df *dxyflake

var startTime int64
var machineID uint64
var serviceID uint64

func init() {
	var st Settings
	st.StartTime = time.Now()
	st.MachineID = func() (uint16, error) {
		return 1, nil
	}
	st.ServiceID = func() (uint16, error) {
		return 2, nil
	}

	df = NewDxyflake(st)
	if df == nil {
		panic("dxyflake not created")
	}

	startTime = toDxyflakeTime(st.StartTime)
	machineID = uint64(1)
	serviceID = uint64(2)
}

func nextID(t *testing.T) uint64 {
	id, err := df.NextID()
	if err != nil {
		t.Fatal("id not generated")
	}
	return id
}

func TestDxyflakeOnce(t *testing.T) {
	sleepTime := uint64(50)
	time.Sleep(time.Duration(sleepTime) * 10 * time.Millisecond)

	id := nextID(t)
	parts := Decompose(id)

	actualMSB := parts["msb"]
	if actualMSB != 0 {
		t.Errorf("unexpected msb: %d", actualMSB)
	}

	actualTime := parts["time"]
	if actualTime < sleepTime || actualTime > sleepTime+1 {
		t.Errorf("unexpected time: %d", actualTime)
	}

	actualMachineID := parts["machine-id"]
	if actualMachineID != machineID {
		t.Errorf("unexpected machine id: %d", actualMachineID)
	}

	actualServiceID := parts["service-id"]
	if actualServiceID != serviceID {
		t.Errorf("unexpected service id: %d", actualServiceID)
	}

	actualSequence := parts["sequence"]
	if actualSequence != 0 {
		t.Errorf("unexpected sequence: %d", actualSequence)
	}

	fmt.Println("dxyflake id:", id)
	fmt.Println("decompose:", parts)
	fmt.Println("hex of id:", fmt.Sprintf("%02X", id))
}

func currentTime() int64 {
	return toDxyflakeTime(time.Now())
}

func TestDxyflakeFor10Sec(t *testing.T) {
	var numID uint32
	var lastID uint64
	var maxSequence uint64

	initial := currentTime()
	current := initial
	for current-initial < 1000 {
		id := nextID(t)
		parts := Decompose(id)
		numID++

		if id <= lastID {
			t.Fatal("duplicated id")
		}
		lastID = id

		current = currentTime()

		actualMSB := parts["msb"]
		if actualMSB != 0 {
			t.Errorf("unexpected msb: %d", actualMSB)
		}

		actualTime := int64(parts["time"])
		overtime := startTime + actualTime - current
		if overtime > 0 {
			t.Errorf("unexpected overtime: %d", overtime)
		}

		actualMachineID := parts["machine-id"]
		if actualMachineID != machineID {
			t.Errorf("unexpected machine id: %d", actualMachineID)
		}

		actualServiceID := parts["service-id"]
		if actualServiceID != serviceID {
			t.Errorf("unexpected service id: %d", actualServiceID)
		}

		actualSequence := parts["sequence"]
		if maxSequence < actualSequence {
			maxSequence = actualSequence
		}
	}

	if maxSequence != 1<<BitLenSequence-1 {
		t.Errorf("unexpected max sequence: %d", maxSequence)
	}
	fmt.Println("max sequence:", maxSequence)
	fmt.Println("number of id:", numID)
}

func TestDxyflakeInParallel(t *testing.T) {
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	fmt.Println("number of cpu:", numCPU)

	consumer := make(chan uint64)

	const numID = 10000
	generate := func() {
		for i := 0; i < numID; i++ {
			consumer <- nextID(t)
		}
	}

	const numGenerator = 10
	for i := 0; i < numGenerator; i++ {
		go generate()
	}

	set := make(map[uint64]struct{})
	for i := 0; i < numID*numGenerator; i++ {
		id := <-consumer
		if _, ok := set[id]; ok {
			t.Fatal("duplicated id")
		}
		set[id] = struct{}{}
	}
	fmt.Println("number of id:", len(set))
}

func TestNildxyflake(t *testing.T) {
	var startInFuture Settings
	startInFuture.StartTime = time.Now().Add(time.Duration(1) * time.Minute)
	if NewDxyflake(startInFuture) != nil {
		t.Errorf("dxyflake starting in the future")
	}

	var noMachineID Settings
	noMachineID.MachineID = func() (uint16, error) {
		return 0, fmt.Errorf("no machine id")
	}
	if NewDxyflake(noMachineID) != nil {
		t.Errorf("dxyflake with no machine id")
	}

	var invalidMachineID Settings
	invalidMachineID.CheckMachineID = func(uint16) bool {
		return false
	}
	if NewDxyflake(invalidMachineID) != nil {
		t.Errorf("dxyflake with invalid machine id")
	}

	var noServiceID Settings
	noServiceID.ServiceID = func() (uint16, error) {
		return 0, fmt.Errorf("no service id")
	}
	if NewDxyflake(noServiceID) != nil {
		t.Errorf("dxyflake with no service id")
	}

	var invalidServiceID Settings
	invalidServiceID.CheckServiceID = func(uint16) bool {
		return false
	}
	if NewDxyflake(invalidServiceID) != nil {
		t.Errorf("dxyflake with invalid service id")
	}
}

func pseudoSleep(period time.Duration) {
	df.startTime -= int64(period)
}

func TestNextIDError(t *testing.T) {
	year := time.Duration(365*24) * time.Hour / dxyflakeTimeUnit
	pseudoSleep(time.Duration(697) * year)
	fmt.Println(df.startTime, 1<<BitLenTime)
	nextID(t)

	pseudoSleep(time.Duration(1) * year)
	_, err := df.NextID()
	fmt.Println(err, df.elapsedTime)
	if err == nil {
		t.Errorf("time is not over")
	}
}
