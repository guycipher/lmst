// Package lsmt tests
// Copyright (C) Alex Gaetano Padula
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
package lsmt

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	// Check if the directory exists
	if _, err := os.Stat("test_lsm_tree"); os.IsNotExist(err) {
		t.Fatal(err)
	}

}

func TestLMST_Put(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	// Insert 268 key-value pairs
	for i := 0; i < 268; i++ {
		log.Println(i)
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	// There should be 2 sstables
	// 0.sst and 1.sst
	if len(lsmt.sstables) != 2 {
		t.Fatalf("expected 2 sstables, got %d", len(lsmt.sstables))
	}

}

func TestLMST_Compact(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 3, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	// Insert 384 key-value pairs
	for i := 0; i < 384; i++ {
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	if len(lsmt.sstables) != 2 {
		t.Fatalf("expected 2 sstables, got %d", len(lsmt.sstables))
	}

	// Check for 0.sst
	if _, err := os.Stat("test_lsm_tree/0.sst"); os.IsNotExist(err) {
		t.Fatal(err)
	}

}

func TestLMST_Delete(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	// Insert 256 key-value pairs
	for i := 0; i < 256; i++ {
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	// Delete 128 key-value pairs
	for i := 0; i < 128; i++ {
		err = lsmt.Delete([]byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

}

func TestLMST_Get(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 15_000, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	// Insert 100,000 key-value pairs
	for i := 0; i < 100_000; i++ {
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	// Get the key
	value, err := lsmt.Get([]byte("99822"))
	if err != nil {
		t.Fatal(err)
	}

	if string(value) != "99822" {
		t.Fatalf("expected 99822, got %s", string(value))
	}
}

func TestLSMT_NGet(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	for i := 0; i < 10; i++ {
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	keys, _, err := lsmt.NGet([]byte("4"))
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 9 {
		t.Fatalf("expected 9 keys, got %d", len(keys))
	}

	expectKeys := [][]byte{
		[]byte("0"),
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
		[]byte("5"),
		[]byte("6"),
		[]byte("7"),
		[]byte("8"),
		[]byte("9"),
	}

	for _, key := range keys {
		for j, expectKey := range expectKeys {
			if string(key) == string(expectKey) {
				// remove the key from the expectKeys
				expectKeys = append(expectKeys[:j], expectKeys[j+1:]...)
				break
			}
			if j == len(expectKeys)-1 {
				t.Fatalf("expected key to be %s, got %s", string(expectKey), string(key))
			}
		}
	}

}

func TestLSMT_Range(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	for i := 0; i < 10; i++ {
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	keys, _, err := lsmt.Range([]byte("4"), []byte("7"))
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 4 {
		t.Fatalf("expected 4 keys, got %d", len(keys))
	}

	expectKeys := [][]byte{
		[]byte("4"),
		[]byte("5"),
		[]byte("6"),
		[]byte("7"),
	}

	for _, key := range keys {
		for j, expectKey := range expectKeys {
			if string(key) == string(expectKey) {
				// remove the key from the expectKeys
				expectKeys = append(expectKeys[:j], expectKeys[j+1:]...)
				break
			}
			if j == len(expectKeys)-1 {
				t.Fatalf("expected key to be %s, got %s", string(expectKey), string(key))
			}
		}
	}
}

func TestLSMT_NRange(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	for i := 0; i < 10; i++ {
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	keys, _, err := lsmt.NRange([]byte("4"), []byte("7"))
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 6 {
		t.Fatalf("expected 6 keys, got %d", len(keys))
	}

	expectKeys := [][]byte{
		[]byte("0"),
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
		[]byte("8"),
		[]byte("9"),
	}

	for _, key := range keys {
		for j, expectKey := range expectKeys {
			if string(key) == string(expectKey) {
				// remove the key from the expectKeys
				expectKeys = append(expectKeys[:j], expectKeys[j+1:]...)
				break
			}
			if j == len(expectKeys)-1 {
				t.Fatalf("expected key to be %s, got %s", string(expectKey), string(key))
			}
		}
	}

}

func TestLSMT_GreaterThan(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	for i := 0; i < 10; i++ {
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	keys, _, err := lsmt.GreaterThan([]byte("4"))
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 5 {
		t.Fatalf("expected 5 keys, got %d", len(keys))
	}

	expectKeys := [][]byte{
		[]byte("5"),
		[]byte("6"),
		[]byte("7"),
		[]byte("8"),
		[]byte("9"),
	}

	for _, key := range keys {
		for j, expectKey := range expectKeys {
			if string(key) == string(expectKey) {
				// remove the key from the expectKeys
				expectKeys = append(expectKeys[:j], expectKeys[j+1:]...)
				break
			}
			if j == len(expectKeys)-1 {
				t.Fatalf("expected key to be %s, got %s", string(expectKey), string(key))
			}
		}
	}
}

func TestLSMT_LessThan(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	for i := 0; i < 10; i++ {
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	keys, _, err := lsmt.LessThan([]byte("4"))
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 4 {
		t.Fatalf("expected 4 keys, got %d", len(keys))
	}

	expectKeys := [][]byte{
		[]byte("0"),
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
	}

	for _, key := range keys {
		for j, expectKey := range expectKeys {
			if string(key) == string(expectKey) {
				// remove the key from the expectKeys
				expectKeys = append(expectKeys[:j], expectKeys[j+1:]...)
				break
			}
			if j == len(expectKeys)-1 {
				t.Fatalf("expected key to be %s, got %s", string(expectKey), string(key))
			}
		}
	}
}

func TestLSMT_GreaterThanEqual(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	for i := 0; i < 10; i++ {
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	keys, _, err := lsmt.GreaterThanEqual([]byte("4"))
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 6 {
		t.Fatalf("expected 6 keys, got %d", len(keys))
	}

	expectKeys := [][]byte{
		[]byte("4"),
		[]byte("5"),
		[]byte("6"),
		[]byte("7"),
		[]byte("8"),
		[]byte("9"),
	}

	for _, key := range keys {
		for j, expectKey := range expectKeys {
			if string(key) == string(expectKey) {
				// remove the key from the expectKeys
				expectKeys = append(expectKeys[:j], expectKeys[j+1:]...)
				break
			}
			if j == len(expectKeys)-1 {
				t.Fatalf("expected key to be %s, got %s", string(expectKey), string(key))
			}
		}
	}
}

func TestLSMT_LessThanEqual(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	for i := 0; i < 10; i++ {
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	keys, _, err := lsmt.LessThanEqual([]byte("4"))
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 5 {
		t.Fatalf("expected 5 keys, got %d", len(keys))
	}

	expectKeys := [][]byte{
		[]byte("0"),
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
		[]byte("4"),
	}

	for _, key := range keys {
		for j, expectKey := range expectKeys {
			if string(key) == string(expectKey) {
				// remove the key from the expectKeys
				expectKeys = append(expectKeys[:j], expectKeys[j+1:]...)
				break
			}
			if j == len(expectKeys)-1 {
				t.Fatalf("expected key to be %s, got %s", string(expectKey), string(key))
			}
		}
	}
}

func TestLSMT_Put(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 1000, 100, 10)
	if err != nil {
		t.Fatal(err)
	}

	if lsmt == nil {
		t.Fatal("expected non-nil lmst")
	}

	defer lsmt.Close()

	// Insert 10000 key-value pairs
	for i := 0; i < 10000; i++ {
		err = lsmt.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			t.Fatal(err)
		}
	}

	// there should be 0-9 sstables

	if len(lsmt.sstables) != 9 {
		t.Fatalf("expected 10 sstables, got %d", len(lsmt.sstables))
	}

	// Get a key
	value, err := lsmt.Get([]byte("9982"))
	if err != nil {
		t.Fatal(err)
	}

	if string(value) != "9982" {
		t.Fatalf("expected 9982, got %s", string(value))
	}

}

func TestLSMT_Case(t *testing.T) {
	// Searching latest sstable..

	defer os.RemoveAll("test_lsm_tree")
	l, err := New("test_lsm_tree", 0755, 100, 15, 13)
	if err != nil {
		log.Fatal(err)
	}

	if l == nil {
		log.Fatal("expected non-nil lmst")
	}

	defer l.Close()

	// Insert 1000 key-value pairs
	for i := 0; i < 1000; i++ {
		err = l.Put([]byte(string(fmt.Sprintf("%d", i))), []byte(string(fmt.Sprintf("%d", i))))
		if err != nil {
			log.Fatal(err)
		}
	}

	// Get a key
	value, err := l.Get([]byte("832"))
	if err != nil {
		log.Fatal(err)
	}

	if string(value) != "832" {
		log.Fatalf("expected 832, got %s", string(value))
	}

}

func TestLSMT_Transaction(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	defer lsmt.Close()

	tx := lsmt.BeginTransaction()
	tx.AddPut([]byte("key1"), []byte("value1"))
	tx.AddPut([]byte("key2"), []byte("value2"))
	tx.AddDelete([]byte("key1"))

	err = lsmt.CommitTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}

	value, err := lsmt.Get([]byte("key1"))
	if err == nil || value != nil {
		t.Fatalf("expected key1 to be deleted, got %s", string(value))
	}

	value, err = lsmt.Get([]byte("key2"))
	if err != nil || string(value) != "value2" {
		t.Fatalf("expected value2, got %s", string(value))
	}
}

func TestLSMT_WalAndRecovery(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	tx := lsmt.BeginTransaction()
	tx.AddPut([]byte("key1"), []byte("value1"))
	tx.AddPut([]byte("key2"), []byte("value2"))
	tx.AddDelete([]byte("key1"))

	err = lsmt.CommitTransaction(tx)
	if err != nil {
		lsmt.Close()
		t.Fatal(err)
	}

	// Get key 2
	value, err := lsmt.Get([]byte("key2"))
	if err != nil || string(value) != "value2" {
		lsmt.Close()
		t.Fatalf("expected value2, got %s", string(value))
	}

	lsmt.Close()

	// delete the sstables
	dirFiles, err := os.ReadDir("test_lsm_tree")
	if err != nil {
		t.Fatal(err)
	}

	for _, dirFile := range dirFiles {
		if dirFile.IsDir() {
			continue
		}
		if strings.HasSuffix(dirFile.Name(), ".sst") {
			err = os.Remove("test_lsm_tree/" + dirFile.Name())
			if err != nil {
				t.Fatal(err)
			}
		}

	}

	lsmt, err = New("test_lsm_tree", 0755, 128, 2, 1)
	if err != nil {
		log.Println("here?")
		t.Fatal(err)
	}

	operations, err := lsmt.GetWal().Recover()
	if err != nil {
		t.Fatal(err)
		return
	}

	for _, operation := range operations {
		switch operation.Type {
		case OpPut:
			err = lsmt.Put(operation.Key, operation.Value)
			if err != nil {
				t.Fatal(err)
			}
		case OpDelete:
			err = lsmt.Delete(operation.Key)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// Get key 2
	value, err = lsmt.Get([]byte("key2"))
	if err != nil || string(value) != "value2" {
		lsmt.Close()
		t.Fatalf("expected value2, got %s", string(value))
	}

}

func TestLSMT_Concurrent(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 1000, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	defer lsmt.Close()

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := []byte(string(fmt.Sprintf("%d", i)))
			value := []byte(string(fmt.Sprintf("%d", i)))

			// Put operation
			err := lsmt.Put(key, value)
			if err != nil {
				t.Errorf("Put operation failed: %v", err)
			}

		}(i)
	}
	wg.Wait()

	for i := 0; i < 1000; i++ {
		value, err := lsmt.Get([]byte(fmt.Sprintf("%d", i)))
		if err != nil {
			t.Errorf("Get operation failed: %v", err)
		}

		if string(value) != fmt.Sprintf("%d", i) {
			t.Errorf("expected %d, got %s", i, string(value))
		}
	}
}

func TestLSMT_Concurrent2(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 22, 12)
	if err != nil {
		t.Fatal(err)
	}

	defer lsmt.Close()

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := []byte(string(fmt.Sprintf("%d", i)))
			value := []byte(string(fmt.Sprintf("%d", i)))

			// Put operation
			err := lsmt.Put(key, value)
			if err != nil {
				t.Errorf("Put operation failed: %v", err)
			}

		}(i)
	}
	wg.Wait()

	for i := 0; i < 1000; i++ {
		value, err := lsmt.Get([]byte(fmt.Sprintf("%d", i)))
		if err != nil {
			t.Errorf("Get operation failed: %v", err)
		}

		if string(value) != fmt.Sprintf("%d", i) {
			t.Errorf("expected %d, got %s", i, string(value))
		}
	}
}

func TestLSMT_Concurrent3(t *testing.T) {
	defer os.RemoveAll("test_lsm_tree")
	lsmt, err := New("test_lsm_tree", 0755, 128, 4, 2)
	if err != nil {
		t.Fatal(err)
	}

	defer lsmt.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := []byte(string(fmt.Sprintf("%d", i)))
			value := []byte(string(fmt.Sprintf("%d", i)))

			// Put operation
			err := lsmt.Put(key, value)
			if err != nil {
				t.Errorf("Put operation failed: %v", err)
			}

		}(i)
	}
	wg.Wait()

	for i := 0; i < 10; i++ {
		value, err := lsmt.Get([]byte(fmt.Sprintf("%d", i)))
		if err != nil {
			t.Errorf("Get operation failed: %v", err)
		}

		if string(value) != fmt.Sprintf("%d", i) {
			t.Errorf("expected %d, got %s", i, string(value))
		}
	}
}
