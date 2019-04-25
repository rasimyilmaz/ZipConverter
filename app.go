package main

import (
	"archive/zip"
	"io"
    "os"
    "log"
    "time"
    "path/filepath"
)
var (
    yesterday string 
    yesterdayFilename string 
    yesterdayZipFilename string
    currentPath string 
)
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
func main() {
    dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Println(err)
	} else {
    	currentPath = dir 
    }
    yesterday=time.Now().AddDate(0,0,-1).Format("2006-01-02")
    yesterdayFilename= yesterday + ".txt"
    yesterdayZipFilename= yesterday+".zip"
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
