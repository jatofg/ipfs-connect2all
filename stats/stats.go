package stats

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

type StatsFile struct {
	dataPtr *[][]float64
	dataMutex *sync.Mutex
}

var statsFilesMutex = &sync.Mutex{}
var statsFiles = make(map[string]StatsFile)

// get pointer for adding data elements, not synced yet
func NewFile(filename string) StatsFile {
	statsFilesMutex.Lock()
	defer statsFilesMutex.Unlock()
	var ret StatsFile
	ret.dataPtr = &[][]float64{}
	ret.dataMutex = &sync.Mutex{}
	statsFiles[filename] = ret
	return ret
}

func (statsFile *StatsFile) AddValues(values []float64) {
	*statsFile.dataPtr = append(*statsFile.dataPtr, values)
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

func (statsFile *StatsFile) AddValuesTS(values []float64) {
	statsFile.dataMutex.Lock()
	statsFile.AddValues(values)
	statsFile.dataMutex.Unlock()
}

func (statsFile *StatsFile) AddFloatsTS(values []float64) {
	statsFile.AddValuesTS(values)
}

func (statsFile *StatsFile) AddIntsTS(values []int) {
	statsFile.dataMutex.Lock()
	statsFile.AddInts(values...)
	statsFile.dataMutex.Unlock()
}

// write collected data to files
func WriteData() {
	statsFilesMutex.Lock()
	for dataFile, dataValues := range statsFiles {
		f, err := os.OpenFile(dataFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			var fileContent strings.Builder
			for i, dataRow := range *dataValues.dataPtr {
				if i > 0 {
					fileContent.WriteByte('\n')
				}
				for j, dataVal := range dataRow {
					if j > 0 {
						fileContent.WriteByte('\t')
					}
					fileContent.WriteString(fmt.Sprintf("%f", dataVal))
				}
			}
			_, _ = f.WriteString(fileContent.String())
			_ = f.Close()
		}
	}
	statsFilesMutex.Unlock()
}