package stats

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

type StatsFile struct {
	filename string
	dataPtr *[][]float64
	dataMutex *sync.Mutex
	callback func([]float64)
}

var statsFilesMutex = &sync.Mutex{}
var statsFiles = make([]*StatsFile, 0, 1)

// get pointer for adding data elements, not synced yet
func NewFile(filename string) *StatsFile {
	statsFilesMutex.Lock()
	defer statsFilesMutex.Unlock()

	ret := &StatsFile{}
	ret.filename = filename
	ret.dataPtr = &[][]float64{}
	ret.dataMutex = &sync.Mutex{}
	ret.callback = nil

	statsFiles = append(statsFiles, ret)
	return ret
}

func NewFileWithCallback(filename string, callback func([]float64)) *StatsFile {
	ret := NewFile(filename)
	ret.callback = callback
	return ret
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
	f, err := os.OpenFile(statsFile.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
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
	_, err = f.WriteString(fileContent.String())
	if err != nil {
		return err
	}
	return f.Close()
}

// write collected data to files
func FlushAll() []error {
	statsFilesMutex.Lock()
	errs := make([]error, 0)
	for _, statsFile := range statsFiles {
		err := statsFile.Flush()
		if err != nil {
			errs = append(errs, err)
		}
	}
	statsFilesMutex.Unlock()
	return errs
}