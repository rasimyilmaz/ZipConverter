package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var (
	yesterday            string
	yesterdayZip         string
	yesterdayFilename    string
	yesterdayZipFilename string
	currentPath          string
	currentSetting       Setting
	settingFilename      string
)

//Setting is configuration parameters
type Setting struct {
	Location       string `json: "Location"`
	ZamaneUsername string `json: "ZamaneUsername"`
	ZamanePassword string `json: "ZamanePassword"`
	ZamaneFilename string `json: "ZamaneFilename"`
}

//CheckFileExists returns Exists , NotExists , DontKnow
func CheckFileExists(filename string) string {
	if _, err := os.Stat(filename); err == nil {
		return "Exists"
	} else if os.IsNotExist(err) {
		return "NotExists"
	} else {
		return "DontKnow"
	}
}
func initialize() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		elog.Warning(1, err.Error())
	} else {
		currentPath = dir
	}
	settingFilename = filepath.Join(currentPath, "setting.json")
	logFilename := filepath.Join(currentPath, svcName+".log")
	f, err := os.OpenFile(logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		elog.Info(1, fmt.Sprintf("Error opening file: %v", err))
	} else {
		log.Printf(svcName + " is started.")
		log.SetOutput(f)
	}
	defer f.Close()
	elog.Info(1, svcName+" is started.")
}
func getSetting() {
	file, err := ioutil.ReadFile(settingFilename)
	err = json.Unmarshal([]byte(file), &currentSetting)
	if err != nil {
		elog.Warning(2, err.Error())
	} else {
		elog.Info(1, fmt.Sprintf("Location : %s , ZamaneUsername : %s , ZamanePassword : %s", currentSetting.Location, currentSetting.ZamaneUsername, currentSetting.ZamanePassword))
	}
}
func finalize() {
	log.Println("Program is ended.")
}
func cycle() {
	var restSecondToNewDay, todaySpentSecond int
	var t time.Time
	var d time.Duration
	for {
		t = time.Now()
		todaySpentSecond = t.Hour()*3600 + t.Minute()*60 + t.Second()
		restSecondToNewDay = (24*60*60 - todaySpentSecond) + 10
		d = time.Duration(restSecondToNewDay) * time.Second
		//time.Sleep(d)
		elog.Info(1, fmt.Sprintln(d, " seconds for executing."))
		time.Sleep(time.Duration(10) * time.Second)
		elog.Info(1, fmt.Sprintln("Program is executing ", t.Local()))
		getSetting()
		yesterday = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		yesterdayFilename = filepath.Join(currentSetting.Location, yesterday+".txt")
		yesterdayZip = yesterday + ".zip"
		yesterdayZipFilename = filepath.Join(currentSetting.Location, yesterdayZip)
		output := yesterdayZipFilename
		if CheckFileExists(yesterdayFilename) == "NotExists" {
			elog.Info(1, fmt.Sprintf("%s dosyası yok.", yesterdayFilename))
		} else {
			if CheckFileExists(yesterdayZipFilename) == "Exists" {
				elog.Info(1, fmt.Sprintf("%s dosyası zaten mevcut.", yesterdayZipFilename))
			} else {
				if err := ZipFile(output, currentSetting.Location, yesterdayZip); err != nil {
					elog.Info(1, fmt.Sprintln(err.Error()))
				} else {
					elog.Info(1, fmt.Sprintf("Zip dosyası hazır: %s", output))
					os.Remove(yesterdayFilename)
					makeTimeStamp()
				}
			}
		}
	}
}

// ZipFiles compresses one or many files into a single zip archive file.
// Param 1: filename is the output zip file's name.
// Param 2: files is a list of files to add to the zip.
func ZipFile(zipfilename string, path string, filename string) error {
	newZipFile, err := os.Create(zipfilename)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	// Add files to zip

	if err = AddFileToZip(zipWriter, path, filename); err != nil {
		return err
	}

	return nil
}

//AddFileToZip save writer to file
func AddFileToZip(zipWriter *zip.Writer, path string, filename string) error {

	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Using FileInfoHeader() above only uses the basename of the file. If we want
	// to preserve the folder structure we can overwrite this with the full path.
	header.Name = filename

	// Change to deflate to gain better compression
	// see http://golang.org/pkg/archive/zip/#pkg-constants
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}
func serve(closesignal chan int) {
	initialize()
	go cycle()
	<-closesignal
	finalize()
}
func makeTimeStamp() {
	_, err := copyZamane()
	if err != nil {
		elog.Warning(2, "Error occurred while copying "+currentSetting.ZamaneFilename+"file."+err.Error())
	}
	cmd := exec.Command("java", "-jar", currentSetting.ZamaneFilename, "-Z", yesterdayZip, "http://zd.kamusm.gov.tr", "80", currentSetting.ZamaneUsername, currentSetting.ZamanePassword, "sha-256")
	cmd.Dir = currentSetting.Location
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		elog.Warning(2, "Error occurred while executing java based jar file."+err.Error())
	} else {
		elog.Info(2, "Zamane command response as :"+out.String())
	}
}
func copyZamane() (int64, error) {
	dst := filepath.Join(currentSetting.Location, currentSetting.ZamaneFilename)
	src := filepath.Join(currentPath, currentSetting.ZamaneFilename)
	if CheckFileExists(dst) == "NotExists" {
		return copy(src, dst)
	}
	return 0, nil
}
func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
