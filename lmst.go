// Package lmst provides a single-level embedded log-structured merge-tree (LSM-tree)
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
package lmst

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"lmst/avl"
	"os"
	"strings"
	"sync"
)

const SSTABLE_EXTENSION = ".sst"

// LMST is the main struct for the log-structured merge-tree.
type LMST struct {
	memtable           *avl.AVLTree
	memtableLock       *sync.RWMutex
	sstables           []*SSTable
	sstablesLock       *sync.RWMutex
	directory          string
	memtableFlushSize  int
	compactionInterval int
}

// SSTable is a struct representing a sorted string table.
type SSTable struct {
	file   *os.File
	minKey []byte
	maxKey []byte
	lock   *sync.RWMutex
}

// New creates a new LSM-tree or opens an existing one.
func New(directory string, directoryPerm os.FileMode, memtableFlushSize, compactionInterval int) (*LMST, error) {
	if directory == "" {
		return nil, errors.New("directory cannot be empty")
	}

	// Check if the directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		// Create the directory if it doesn't exist
		err = os.Mkdir(directory, directoryPerm)
		if err != nil {
			return nil, err
		}

		return &LMST{
			memtable:           avl.NewAVLTree(),
			memtableLock:       &sync.RWMutex{},
			sstables:           make([]*SSTable, 0),
			sstablesLock:       &sync.RWMutex{},
			directory:          directory,
			memtableFlushSize:  memtableFlushSize,
			compactionInterval: compactionInterval,
		}, nil
	} else {
		// If the directory exists, check if it is a directory
		if s, err := os.Stat(directory); err != nil || !s.IsDir() {
			return nil, errors.New("directory is not a directory")
		}

		// We create the directory and populate it with the SSTables
		files, err := os.ReadDir(directory)
		if err != nil {
			return nil, err
		}

		sstables := make([]*SSTable, 0)

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if !strings.HasSuffix(file.Name(), SSTABLE_EXTENSION) {
				continue
			}

			// Open the SSTable file
			sstableFile, err := os.OpenFile(directory+string(os.PathSeparator)+file.Name(), os.O_RDWR, 0644)
			if err != nil {
				return nil, err
			}

			// Create a new SSTable
			sstable := &SSTable{
				file:   sstableFile,
				minKey: nil,
				maxKey: nil,
				lock:   &sync.RWMutex{},
			}

			// Add the SSTable to the list of SSTables
			sstables = append(sstables, sstable)

			return &LMST{
				memtable:           avl.NewAVLTree(),
				memtableLock:       &sync.RWMutex{},
				sstables:           sstables,
				sstablesLock:       &sync.RWMutex{},
				directory:          directory,
				memtableFlushSize:  memtableFlushSize,
				compactionInterval: compactionInterval,
			}, nil
		}

	}

	return nil, errors.New("directory is not a directory")

}

func (l *LMST) Put(key, value []byte) error {
	// We will first put the key-value pair in the memtable.
	// If the memtable size exceeds the flush size, we will flush the memtable to disk.

	// Put the key-value pair in the memtable.
	l.memtable.Insert(key, value)

	// If the memtable size exceeds the flush size, flush the memtable to disk.
	if l.memtable.GetSize() > l.memtableFlushSize {
		if err := l.flushMemtable(); err != nil {
			return err
		}
	}

	return nil
}

func (l *LMST) flushMemtable() error {
	// We will create a new SSTable from the memtable and add it to the list of SSTables.
	// We will then clear the memtable.

	// Create a new SSTable from the memtable.
	sstable, err := newSSTable(l.directory, l.memtable)
	if err != nil {
		return err
	}

	// Add the SSTable to the list of SSTables.
	l.sstables = append(l.sstables, sstable)

	// Clear the memtable.
	l.memtable = avl.NewAVLTree()

	return nil
}

type KeyValue struct {
	Key   []byte
	Value []byte
}

func newSSTable(directory string, memtable *avl.AVLTree) (*SSTable, error) {

	// Create a sorted map from the memtable which will be used to create the SSTable.

	sstableSlice := make([]*KeyValue, 0)

	memtable.InOrderTraversal(func(node *avl.Node) {
		sstableSlice = append(sstableSlice, &KeyValue{Key: node.Key, Value: node.Value})
	})

	// Based on amount of sstables we name the file
	fileName := fmt.Sprintf("%s%s%d%s", directory, string(os.PathSeparator), len(sstableSlice), SSTABLE_EXTENSION)

	// Create a new SSTable file.
	ssltableFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	enc := gob.NewEncoder(ssltableFile)

	enc.Encode(sstableSlice)

	return &SSTable{
		file:   ssltableFile,
		minKey: sstableSlice[0].Key,
		maxKey: sstableSlice[len(sstableSlice)-1].Key,
	}, nil
}

// getSSTableKVs reads the key-value pairs from the SSTable file.
func getSSTableKVs(file *os.File) ([]*KeyValue, error) {
	// Decode the SSTable file.
	dec := gob.NewDecoder(file)

	var kvs []*KeyValue
	err := dec.Decode(&kvs)
	if err != nil {
		return nil, err
	}

	return kvs, nil
}

func (l *LMST) Get(key []byte) ([]byte, error) {
	// We will first check the memtable for the key.
	// If the key is not found in the memtable, we will search the SSTables.

	// Check the memtable for the key.
	if node := l.memtable.Search(key); node != nil {
		return node.Value, nil
	}

	// Search the SSTables for the key.
	for i := len(l.sstables) - 1; i >= 0; i-- {
		sstable := l.sstables[i]

		// If the key is not within the range of this SSTable, skip it.
		if bytes.Compare(key, sstable.minKey) < 0 || bytes.Compare(key, sstable.maxKey) > 0 {
			continue
		}

		// Read the key-value pairs from the SSTable file.
		kvs, err := getSSTableKVs(sstable.file)
		if err != nil {
			return nil, err
		}

		// Perform a binary search on the SSTable.
		index := binarySearch(kvs, key)
		if index != -1 {
			return kvs[index].Value, nil
		}
	}

	return nil, errors.New("key not found")
}

func (l *LMST) Delete(key []byte) error {
	// We will write a tombstone value to the memtable for the key.

	// Write a tombstone value to the memtable for the key.
	l.memtable.Insert(key, []byte(""))

	return nil
}

// binarySearch performs a binary search on the key-value pairs to find the key.
func binarySearch(kvs []*KeyValue, key []byte) int {
	low, high := 0, len(kvs)-1

	for low <= high {
		mid := low + (high-low)/2

		if bytes.Compare(kvs[mid].Key, key) == 0 {
			return mid
		} else if bytes.Compare(kvs[mid].Key, key) < 0 {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	return -1
}

func (l *LMST) Compact() error {
	// Create a new empty memtable.
	newMemtable := avl.NewAVLTree()

	// Iterate over all existing SSTables.
	for _, sstable := range l.sstables {
		// Read all key-value pairs from the SSTable.
		kvs, err := getSSTableKVs(sstable.file)
		if err != nil {
			return err
		}

		// For each key-value pair, check if the value is a tombstone.
		for _, kv := range kvs {
			if len(kv.Value) != 0 {
				// If the value is not a tombstone, add it to the new memtable.
				newMemtable.Insert(kv.Key, kv.Value)
			}
		}
	}

	// Flush the new memtable to disk, creating a new SSTable.
	newSSTable, err := newSSTable(l.directory, newMemtable)
	if err != nil {
		return err
	}

	// Replace the list of old SSTables with the new SSTable.
	l.sstables = []*SSTable{newSSTable}

	return nil
}
