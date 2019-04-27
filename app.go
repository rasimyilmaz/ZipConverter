package main

import (
	"archive/zip"
	"io"
    "os"
    "log"
    "time"
	"path/filepath"
	"encoding/json"
	"io/ioutil"
	"golang.org/x/sys/windows/svc/eventlog"
	"fmt"
	"github.com/carlescere/scheduler"
)
var (
    yesterday string 
    yesterdayFilename string 
    yesterdayZipFilename string
	currentPath string
	currentSetting Setting
	settingFilename string
)
//Setting is configuration parameters
type Setting struct {
	location string `json: "location"`
	username string `json: "username"`
	password string `json: "username"`
}
//CheckFileExists returns Exists , NotExists , DontKnow
func CheckFileExists(filename string) (string){
    if _, err := os.Stat(filename); err == nil {
        return "Exists"      
      } else if os.IsNotExist(err) {
        return "NotExists"
      } else {
        return "DontKnow"
      }
}
func initialize(){
	const name = "ZipConverter"
	elog, err := eventlog.Open(name)
	if (err!=nil){
		log.Println("Event logger could not open.")
	}
    dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		elog.Warning(1,err.Error())
	} else {
    	currentPath = dir 
	}
	settingFilename=filepath.Join(currentPath, "setting.json")
	logFilename := filepath.Join(dir, name+".log")
	f, err := os.OpenFile(logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		elog.Info(1, fmt.Sprintf("Error opening file: %v", err))
	} else {
		log.SetOutput(f)
	}
	defer f.Close()
	elog.Warning(0,name +" is started.")
}
func getSetting(){
	file, err := ioutil.ReadFile(settingFilename)
	err = json.Unmarshal([]byte(file), &currentSetting)
	if err != nil {
		elog.Warning(2, err.Error())
	}
}
func finalize(){
	err:=elog.Close()
	if (err!=nil){
		log.Println(err.Error)
		elog.Warning(0,"Program is end.")
	}
}
func cycle() {
	t := time.Now()
	elog.Warning(1,fmt.Sprintln("Program is executing ", t.Local()))
	getSetting()
    yesterday=time.Now().AddDate(0,0,-1).Format("2006-01-02")
    yesterdayFilename= filepath.Join(currentSetting.location, yesterday + ".txt")
    yesterdayZipFilename= filepath.Join(currentSetting.location,yesterday+".zip")
	// List of Files to Zip
	files := []string{yesterdayFilename}
	output :=  yesterdayZipFilename
    if (CheckFileExists(yesterdayFilename) == "NotExists"){
        log.Println(yesterdayFilename + " dosyası yok.")
    } else {
        if (CheckFileExists(yesterdayZipFilename) == "Exists"){
            log.Println(yesterdayZipFilename + " dosyası zaten mevcut.")
        } else {
            if err := ZipFiles(output, files); err != nil {
                log.Println(err.Error())
            }else {
                log.Println("Zip dosyası hazır:", output)
                os.Remove(yesterdayFilename)
            }
        }
    }
}

// ZipFiles compresses one or many files into a single zip archive file.
// Param 1: filename is the output zip file's name.
// Param 2: files is a list of files to add to the zip.
func ZipFiles(filename string, files []string) error {
	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {
		if err = AddFileToZip(zipWriter, file); err != nil {
			return err
		}
	}
	return nil
}
//AddFileToZip save writer to file
func AddFileToZip(zipWriter *zip.Writer, filename string) error {

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
	//scheduler.Every().Day().At("0:01").Run(cycle)
	scheduler.Every().Minutes().Run(cycle)
	<-closesignal
	finalize()
}