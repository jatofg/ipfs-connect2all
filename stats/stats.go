package stats

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

type StatsFile struct {
	file *os.File
	dataPtr *[][]float64
	dataMutex *sync.Mutex
	callback func([]float64)
	invalid bool
}

var statsFilesMutex = &sync.Mutex{}
var statsFiles = make([]*StatsFile, 0, 1)

// get pointer for adding data elements, not synced yet
func NewFile(filename string) (*StatsFile, error) {
	statsFilesMutex.Lock()
	defer statsFilesMutex.Unlock()
	ret := &StatsFile{}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	ret.file = f
	ret.dataPtr = &[][]float64{}
	ret.dataMutex = &sync.Mutex{}
	ret.callback = nil
	ret.invalid = false

	statsFiles = append(statsFiles, ret)
	return ret, nil
}

func NewFileWithCallback(filename string, callback func([]float64)) (*StatsFile, error) {
	ret, err := NewFile(filename)
	if err != nil {
		return nil, err
	}

	ret.callback = callback
	return ret, nil
}

func (statsFile *StatsFile) AddValues(values []float64) {
	statsFile.dataMutex.Lock()
	*statsFile.dataPtr = append(*statsFile.dataPtr, values)
	statsFile.dataMutex.Unlock()

	if statsFile.callback != nil {
		statsFile.callback(values)
	}
}

func (statsFile *StatsFile) AddFloats(values ...float64) {
	statsFile.AddValues(values)
}

func (statsFile *StatsFile) AddInts(intValues ...int) {
	values := make([]float64, len(intValues))
	for i, v := range intValues {
		values[i] = float64(v)
	}
	statsFile.AddValues(values)
}

func (statsFile *StatsFile) Flush() error {
	statsFile.dataMutex.Lock()
	defer statsFile.dataMutex.Unlock()

	var fileContent strings.Builder
	for _, dataRow := range *statsFile.dataPtr {
		for j, dataVal := range dataRow {
			if j > 0 {
				fileContent.WriteByte('\t')
			}
			fileContent.WriteString(fmt.Sprintf("%f", dataVal))
		}
		fileContent.WriteByte('\n')
	}
	statsFile.dataPtr = &[][]float64{}
	_, err := statsFile.file.WriteString(fileContent.String())
	if err != nil {
		return err
	}
	err = statsFile.file.Sync()
	if err != nil {
		return err
	}
	return nil
}

func (statsFile *StatsFile) Close() error {
	statsFile.invalid = true
	return statsFile.file.Close()
}

func (statsFile *StatsFile) FlushAndClose() (error, error) {
	return statsFile.Flush(), statsFile.Close()
}

// write collected data to files
func FlushAll() []error {
	statsFilesMutex.Lock()
	errs := make([]error, 0)
	for _, statsFile := range statsFiles {
		if statsFile.invalid {
			continue
		}
		err := statsFile.Flush()
		if err != nil {
			errs = append(errs, err)
		}
	}
	statsFilesMutex.Unlock()
	return errs
}

func CloseAll() []error {
	statsFilesMutex.Lock()
	errs := make([]error, 0)
	for _, statsFile := range statsFiles {
		if statsFile.invalid {
			continue
		}
		err := statsFile.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	statsFiles = make([]*StatsFile, 0, 1)
	statsFilesMutex.Unlock()
	return errs
}

func FlushAndCloseAll() []error {
	errs := FlushAll()
	return append(errs, CloseAll()...)
}
