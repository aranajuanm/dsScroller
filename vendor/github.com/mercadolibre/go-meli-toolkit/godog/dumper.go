/**
* @author mlabarinas
 */

package godog

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

const (
	TAB       = "\t"
	NEW_LINE  = "\n"
	SEPARATOR = ":"
)

type byModTime []os.FileInfo

func (f byModTime) Len() int {
	return len(f)
}

func (f byModTime) Less(i, j int) bool {
	return f[i].ModTime().Unix() > f[j].ModTime().Unix()
}

func (f byModTime) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func start() {
	for {
		if error := dump(); error != nil {
			log.Println("Error to create dump, error: " + error.Error())
		}

		if sleep, _ := strconv.Atoi(Config["dump_default_interval"]); sleep > 0 {
			time.Sleep(time.Duration(sleep) * time.Second)
		}
	}
}

func dump() error {
	if !validAmountOfMetrics() {
		cleanMetrics()

		return errors.New("Invalid amount of metrics")
	}

	files := listDumpFiles()

	if maxFiles, _ := strconv.Atoi(Config["max_files"]); maxFiles <= len(files) {
		deleteOldestFile(files)
	}

	metrics := cloneMetrics()

	if metrics.Count() == 0 {
		return nil
	}

	var buffer bytes.Buffer

	for metric := range metrics.Iter() {
		dumpMetric(&buffer, metric.Val.(*Metric))
	}

	file, error := os.Create(getDumpFileName())

	if error != nil {
		return error
	}

	defer file.Close()

	file.WriteString(buffer.String())

	return nil
}

func validAmountOfMetrics() bool {
	if maxCombinatory, _ := strconv.Atoi(Config["max_metrics_combinatory"]); maxCombinatory < getMetricsCombinatoryCount() {
		return false
	}

	return true
}

func listDumpFiles() []os.FileInfo {
	if files, error := ioutil.ReadDir(Config["dump_base_path"]); error != nil {
		return make([]os.FileInfo, 0, 0)

	} else {
		return files
	}
}

func deleteOldestFile(files []os.FileInfo) error {
	if len(files) <= 0 {
		return nil
	}

	sort.Sort(byModTime(files))

	return os.Remove(fmt.Sprintf("%s%s%s", Config["dump_base_path"], string(filepath.Separator), files[0].Name()))
}

func getDumpFileName() string {
	return fmt.Sprintf("%s%s%d.dat", Config["dump_base_path"], string(filepath.Separator), time.Now().UnixNano()/int64(time.Millisecond))
}

func dumpMetric(buffer *bytes.Buffer, metric *Metric) {
	buffer.WriteString(metric.GetName())
	buffer.WriteString(TAB)
	buffer.WriteString(metric.GetClass())
	buffer.WriteString(TAB)
	buffer.WriteString(strconv.FormatFloat(metric.GetSum(), 'f', 5, 64))

	if metric.GetClass() == "C" || metric.GetClass() == "F" {
		buffer.WriteString(SEPARATOR)
		buffer.WriteString(strconv.FormatInt(metric.GetAmount(), 10))
	}

	if metric.GetClass() == "F" {
		buffer.WriteString(SEPARATOR)
		buffer.WriteString(strconv.FormatFloat(metric.GetMin(), 'f', 5, 64))
		buffer.WriteString(SEPARATOR)
		buffer.WriteString(strconv.FormatFloat(metric.GetMax(), 'f', 5, 64))
	}

	tags := metric.GetTags()

	for _, tag := range tags {
		buffer.WriteString(TAB)
		buffer.WriteString(tag)
	}

	buffer.WriteString(NEW_LINE)
}

func init() {
	if _, err := os.Stat(Config["dump_base_path"]); os.IsNotExist(err) {
		os.MkdirAll(Config["dump_base_path"], 0755)
	}
}
