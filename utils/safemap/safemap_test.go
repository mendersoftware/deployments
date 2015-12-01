package safemap

import (
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

func TestSafeMapSetGetString(t *testing.T) {

	sm := NewStringMap()
	sm.Set("mykey", "string")
	val, found := sm.Get("mykey")
	if !found {
		t.FailNow()
	}

	if val != "string" {
		t.FailNow()
	}
}

func TestSafeMapSetGetStruct(t *testing.T) {

	data := struct {
		a string
		b int
	}{
		"lala",
		1,
	}

	sm := NewStringMap()
	sm.Set("mykey", data)
	val, found := sm.Get("mykey")
	if !found {
		t.FailNow()
	}

	if !reflect.DeepEqual(data, val) {
		t.FailNow()
	}
}

func TestSafeMapGetNotFound(t *testing.T) {

	sm := NewStringMap()
	_, found := sm.Get("not_my_key")
	if found {
		t.FailNow()
	}
}

func TestSafeMapHas(t *testing.T) {

	sm := NewStringMap()
	sm.Set("mykey", "string")

	if !sm.Has("mykey") {
		t.FailNow()
	}
}

func TestSafeMapRemove(t *testing.T) {

	sm := NewStringMap()
	sm.Set("mykey", "string")

	sm.Remove("mykey")

	if sm.Has("mykey") {
		t.FailNow()
	}
}

func TestSafeMapCount(t *testing.T) {

	sm := NewStringMap()

	if sm.Count() != 0 {
		t.FailNow()
	}

	sm.Set("mykey", "string")

	if sm.Count() != 1 {
		t.FailNow()
	}
}

func TestSafeMapKeys(t *testing.T) {

	sm := NewStringMap()

	if len(sm.Keys()) != 0 {
		t.FailNow()
	}

	sm.Set("mykey_1", "string")
	sm.Set("mykey_2", "string")

	keys := sm.Keys()

	if len(keys) != 2 {
		t.FailNow()
	}
}

func BenchmarkStringMapSet1(b *testing.B) {

	for n := 0; n < b.N; n++ {

		b.StopTimer()
		m := NewStringMap()
		b.StartTimer()

		m.Set("key", "my_dummy_value")
	}
}

func BenchmarkStringMapSet10(b *testing.B) {

	for n := 0; n < b.N; n++ {

		b.StopTimer()
		m := NewStringMap()
		b.StartTimer()

		for i := 0; i < 10; i++ {
			m.Set("key_"+strconv.Itoa(i), "my_dummy_value")
		}
	}
}

func BenchmarkStringMapGetFrom100(b *testing.B) {

	b.StopTimer()
	size := 100
	m := NewStringMap()

	for i := 0; i < size; i++ {
		m.Set("key_"+strconv.Itoa(i), "my_dummy_value")
	}
	b.StartTimer()

	for n := 0; n < b.N; n++ {
		b.StopTimer()
		rand_key := "key_" + strconv.Itoa(rand.Intn(size))
		b.StartTimer()
		if _, found := m.Get(rand_key); !found {
			b.FailNow()
		}
	}
}

func BenchmarkStringMapGetFrom1000(b *testing.B) {

	b.StopTimer()
	size := 1000
	m := NewStringMap()

	for i := 0; i < size; i++ {
		m.Set("key_"+strconv.Itoa(i), "my_dummy_value")
	}
	b.StartTimer()

	for n := 0; n < b.N; n++ {
		b.StopTimer()
		rand_key := "key_" + strconv.Itoa(rand.Intn(size))
		b.StartTimer()
		if _, found := m.Get(rand_key); !found {
			b.FailNow()
		}
	}
}

func BenchmarkStringMapGetFrom1000Concurent100CPU1(b *testing.B) {

	if runtime.NumCPU() < 1 {
		b.SkipNow()
	}

	runtime.GOMAXPROCS(1)

	size := 1000
	routines := 100
	m := NewStringMap()

	for i := 0; i < size; i++ {
		m.Set("key_"+strconv.Itoa(i), "my_dummy_value")
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {

		var wg sync.WaitGroup

		for i := 0; i < routines; i++ {
			wg.Add(1)
			go func() {
				b.StopTimer()
				defer wg.Done()
				rand_key := "key_" + strconv.Itoa(rand.Intn(size))
				b.StartTimer()

				if _, found := m.Get(rand_key); !found {
					b.FailNow()
				}
			}()
		}
		wg.Wait()
	}
}

func BenchmarkStringMapGetFrom1000Concurent100CPU2(b *testing.B) {

	if runtime.NumCPU() < 2 {
		b.SkipNow()
	}

	runtime.GOMAXPROCS(2)

	size := 1000
	routines := 100
	m := NewStringMap()

	for i := 0; i < size; i++ {
		m.Set("key_"+strconv.Itoa(i), "my_dummy_value")
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {

		var wg sync.WaitGroup

		for i := 0; i < routines; i++ {
			wg.Add(1)
			go func() {
				b.StopTimer()
				defer wg.Done()
				rand_key := "key_" + strconv.Itoa(rand.Intn(size))
				b.StartTimer()

				if _, found := m.Get(rand_key); !found {
					b.FailNow()
				}
			}()
		}
		wg.Wait()
	}
}

func BenchmarkStringMapGetFrom1000Concurent100CPU4(b *testing.B) {

	if runtime.NumCPU() < 4 {
		b.SkipNow()
	}

	runtime.GOMAXPROCS(4)

	size := 1000
	routines := 100
	m := NewStringMap()

	for i := 0; i < size; i++ {
		m.Set("key_"+strconv.Itoa(i), "my_dummy_value")
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {

		var wg sync.WaitGroup

		for i := 0; i < routines; i++ {
			wg.Add(1)
			go func() {
				b.StopTimer()
				defer wg.Done()
				rand_key := "key_" + strconv.Itoa(rand.Intn(size))
				b.StartTimer()

				if _, found := m.Get(rand_key); !found {
					b.FailNow()
				}
			}()
		}
		wg.Wait()
	}
}
