// Go
//
//  Get mp3 from list of youtube links.
//  Trying to obtain best quality, and then lowering, if better not existing.
//
//   Options:
//				m only mp3
//				v only video
//				no option || b Keep both (video and mp3) (DEFAULT)
//
//
//
//  OS: linux, macosx
//  !keep python3 /usr/local/bin/youtube-dl updated
//
//  usage: create file list.txt and put youtube links inside, one link per line, example:
//  list.txt:
//  	https://www.youtube.com/watch?v=u3m2kQ-tOEk
//  	https://www.youtube.com/watch?v=kkJtHXfRH74&list=WL&index=24&t=0s m
//  	https://www.youtube.com/watch?v=qGyPuey-1Jw v
//  	https://www.youtube.com/watch?v=qGyPuey-1Jw v newFolder
//  	https://www.youtube.com/watch?v=qGyPuey-1Jw v FolderKeyDefinedInSettingsFile
//
//  @requrired python3 /usr/local/bin/youtube-dl, ffmpeg
//  @author    Pavel Filipcik
//  @year      2017-2021

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	"youtube/codes/core"
)

var listFileNamePath = "list.txt"
var videoHTMLpagePath = "videolisthtml.txt"
var videoHTMLpagePathLoaded = "videolisthtml.txt_d"

var listHTMLpagePath = "playlisthtml.txt"
var listHTMLpagePathLoaded = "playlisthtml.txt_d"

const (
	root           = "root"
	youtubeURL     = "https://youtube.com"
	youtubeURLFull = "https://www.youtube.com/"
	listHTMLFolder = "listHTMLFolder"
	typeMusic      = "typeMusic"
	typeVideo      = "typeVideo"
	typeMusicShort = "m"
	typeMusicLong  = "mp3"
	typeVideoShort = "v"
	typeVideoLong  = "video"
	typeBoth       = "both"
	typeBothShort  = "b"
	listURLPart    = "/watch?v="
)

var folders map[string]string

type ytVideoOptions struct {
	musicIndex string
	videoIndex string
}

type video struct {
	hasError       bool
	err            error
	errMsg         string
	link           string
	createMp3      bool
	keepVideo      bool
	videoFilePath  string
	mp3FilePath    string
	parsingLine    string
	videoName      string
	videoExtension string
	moveDir        string
	ytVideoOptions
}

func (v *video) initVideo() {
	v.hasError = false
	v.errMsg = ""
	v.err = nil
	v.link = ""
	v.createMp3 = false
	v.keepVideo = false
	v.videoFilePath = ""
	v.mp3FilePath = ""
	v.moveDir = ""
}

func (v *video) printMe() {
	fmt.Printf("%+v\n", v)
}

func (v *video) setError(msg string, err error) {
	v.hasError = true

	if err == nil {
		v.err = errors.New(msg)
	} else {
		v.err = err
	}
}

var videoList = []video{}

func (v *video) getFullName() string {
	return v.videoName + "-" + v.videoIndex + "." + v.videoExtension
}

func (v *video) getFullMp3Name() string {
	return v.videoName + ".mp3"
}

func (v *video) getMp3() {

	if v.createMp3 == true {
		var fullName = v.getFullName()
		fmt.Println("get mp3 + " + fullName)

		if !core.FileExists(fullName) {
			v.setError("Create of mp3 file failed: missing video file to be converted to mp3.", nil)
		}

		if !core.FileExists(v.getFullMp3Name()) {
			var quality string

			if v.musicIndex == "22" {
				//.mp4
				quality = "192k"

			} else if v.musicIndex == "251" {
				//.webm
				quality = "160k"
			} else {
				v.setError("Create of mp3 file failed: missing part for quality of result mp3, definition of index", nil)
				return
			}

			errCreateMp3 := createMp3(quality, fullName, v.videoName)

			if errCreateMp3 != nil {
				v.setError("Create of mp3 file failed: ffmpeg convert.", errCreateMp3)
			}
		}
	}
}

func createMp3(quality string, fullName string, videoName string) error {

	cmd := exec.Command("ffmpeg", "-i", fullName, "-vn", "-acodec", "mp3", "-ab", quality, "-ar", "44100", "-ac", "2", "-map", "a", videoName+".mp3")
	out, errCO := cmd.CombinedOutput()

	if errCO != nil {
		return errCO
	}

	if !core.FileExists(videoName + ".mp3") {
		outp := string(out)
		// should not happen, maybe full disk?
		errMsg := fmt.Sprintf("Mp3 file was not created: %s, fullName: %s, vidoname: %s \n output: %s \n", quality, fullName, videoName, outp)

		return errors.New(errMsg)
	}

	return nil
}

func (v *video) downloadVideoIndexesFiles() {
	videoFullName := v.getFullName()

	fmt.Println("Download of", videoFullName, v.link)

	if v.ytVideoOptions.videoIndex != "" {
		cmd := exec.Command("python3", "/usr/local/bin/youtube-dl", "-f", v.ytVideoOptions.videoIndex, "-o", videoFullName, v.link)
		err := cmd.Run()

		if err != nil {
			//fmt.Println("python3 /usr/local/bin/youtube-dl", "-f", v.ytVideoOptions.videoIndex, "-o", videoFullName, v.link)

			errString := fmt.Sprintln("python3 /usr/local/bin/youtube-dl", "-f", v.ytVideoOptions.videoIndex, "-o", videoFullName, v.link)

			v.setError("Command python3 /usr/local/bin/youtube-dl failed with: "+errString, err)
		}

		// fmt.Println(string(out))
	}

	if v.createMp3 && v.ytVideoOptions.videoIndex != v.ytVideoOptions.musicIndex {
		cmd := exec.Command("python3", "/usr/local/bin/youtube-dl", "-f", v.ytVideoOptions.musicIndex, "-o", videoFullName, v.link)
		err := cmd.Run()

		if err != nil {
			errString := fmt.Sprint("python3 /usr/local/bin/youtube-dl", "-f", v.ytVideoOptions.musicIndex, "-o", videoFullName, v.link)

			//fmt.Println("python3 /usr/local/bin/youtube-dl", "-f", v.ytVideoOptions.musicIndex, "-o", videoFullName, v.link)
			v.setError("Command python3 /usr/local/bin/youtube-dl failed with: "+errString, err)
		}
	}
}

func (v *video) removeVideo() {
	if v.hasError == false && v.keepVideo == false {
		fullName := v.getFullName()
		fmt.Println("Delete video: ", fullName)
		removeErr := os.Remove(fullName)
		if removeErr != nil {
			v.setError("Delete video file failed.", removeErr)
		}
	}
}

func (v *video) getBestQualityVideo(output string) (hasAudioSource bool, err error) {

	stringsReader := strings.NewReader(output)
	scanner := bufio.NewScanner(stringsReader)

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 1 && strings.Contains(line, "(best)") {

			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "22") {
				v.ytVideoOptions.musicIndex = "22" // contains also best audio
				v.ytVideoOptions.videoIndex = "22"
				v.videoExtension = getExtensionFromYtIndexLine(line)
				return true, nil
			}

			if v.keepVideo == true {
				splitedLine := strings.Split(line, " ")
				videoIndex := strings.TrimSpace(splitedLine[0])

				//fmt.Println("video best option: ", videoIndex, v.link)
				v.ytVideoOptions.videoIndex = videoIndex
				v.videoExtension = getExtensionFromYtIndexLine(line)
				return false, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		v.setError("scanner getBestQualityVideo failed", err)
		return false, err
	}

	return false, nil
}

func (v *video) getBestQualityAudio(output string) (err error) {
	// options 251, 140
	stringsReader := strings.NewReader(output)
	scanner := bufio.NewScanner(stringsReader)

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 1 && strings.Contains(line, "audio only") {

			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "251") {
				v.ytVideoOptions.musicIndex = "251"

				v.videoExtension = getExtensionFromYtIndexLine(line)
				return nil
			}

			// TODO selection of formats
			splitedLine := strings.Split(line, " ")
			videoIndex := strings.TrimSpace(splitedLine[0])

			v.ytVideoOptions.musicIndex = videoIndex
			v.videoExtension = getExtensionFromYtIndexLine(line)
			// fmt.Println("worst audio: ", videoIndex, v.link)

			// do not return, get all options, and overwrite with the best
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func getExtensionFromYtIndexLine(line string) (extension string) {
	splitedLine := strings.Fields(line)
	extension = strings.TrimSpace(splitedLine[1])

	return extension
}

func loadVideoNames(v video) video {
	if v.hasError == false {
		cmd := exec.Command("python3", "/usr/local/bin/youtube-dl", "-e", v.link)
		out, errCO := cmd.CombinedOutput()

		if errCO != nil {
			core.LogError(errCO, "cmd.Run() failed with")
		}

		bufferOutput := string(out)

		bufferOutput = core.CleanCharactersFromString(bufferOutput)
		v.videoName = bufferOutput
	}
	return v
}

func loadVideoOptions(v video) video {
	if v.hasError == false {
		cmd := exec.Command("python3", "/usr/local/bin/youtube-dl", "-F", v.link)
		out, errCO := cmd.CombinedOutput()

		if errCO != nil {
			v.setError("get video index failed	", errCO)
		}

		bufferOutput := string(out)
		hasAudioSource, errVideo := v.getBestQualityVideo(bufferOutput)

		if errVideo != nil {
			v.setError("bestquality failed with", errVideo)
		}

		if v.createMp3 && hasAudioSource == false {
			errAudio := v.getBestQualityAudio(bufferOutput)

			if errAudio != nil {
				v.setError("audio get best quality failed with", errAudio)
			}
		}
	}

	return v
}

type incCounter struct {
	mux sync.Mutex
	c   int
}

var processVideosCounter incCounter = incCounter{}

func (counter *incCounter) increment() {
	counter.mux.Lock()
	defer counter.mux.Unlock()

	counter.c++
}

func (counter *incCounter) getValueAsString() string {
	counter.mux.Lock()
	defer counter.mux.Unlock()

	return strconv.Itoa(counter.c)
}

func processVideoList(wg *sync.WaitGroup, videoChannel chan video, errorChannel chan video) {
	defer wg.Done()

	for v := range videoChannel {
		processVideosCounter.increment()
		fmt.Printf("  Processing video %s/%d: %s | %s\n", processVideosCounter.getValueAsString(), allVideosCount, v.videoName, v.link)
		if v.hasError == false {
			v.downloadVideoIndexesFiles()

			if v.hasError == false {

				v.getMp3()
				v.removeVideo()
				v = moveFile(v)
			}

			if v.hasError == true {
				errorChannel <- v
			}
		}
	}

}

func moveFile(v video) video {
	moveToDir := ""
	nameOfFile := ""

	if v.keepVideo {
		moveToDir = folders[typeVideo]

		nameOfFile = v.getFullName()

		if v.moveDir != "" {
			moveToDir = v.moveDir
		}
	}

	if core.FileExists(nameOfFile) {
		errMoveVideo := moveWrapper(nameOfFile, moveToDir+"/"+nameOfFile)

		if errMoveVideo != nil {
			v.setError("Moving video file failed for file:", errMoveVideo)
		}
	}

	if v.createMp3 {
		moveToDir = folders[typeMusic]

		if v.moveDir != "" {
			moveToDir = v.moveDir
		}

		nameOfFile = v.getFullMp3Name()

		errMoveMp3 := moveWrapper(nameOfFile, moveToDir+"/"+nameOfFile)

		if errMoveMp3 != nil {
			v.setError("Moving mp3 file failed for file:", errMoveMp3)
		}
	}

	return v
}

// fix the cross-device error for external disks
func moveWrapper(srcFolder, dstFolder string) error {
	fmt.Println("move file : ", srcFolder, dstFolder)
	if srcFolder == "" || dstFolder == "" {
		return errors.New("move file failed, empty source or destination")
	}
	if srcFolder == dstFolder {
		return nil
	}

	errMoveMp3 := os.Rename(srcFolder, dstFolder)
	if errMoveMp3 != nil && strings.Contains(errMoveMp3.Error(), "cross-device") {

		cpCmd := exec.Command("cp", "-rf", srcFolder, dstFolder)
		err := cpCmd.Run()
		if err != nil {
			return err
		}

		if err := os.Remove(srcFolder); err != nil {
			return err
		}
	}

	return nil
}

type ERR_PARSE_EMPTY struct {
}

func (e ERR_PARSE_EMPTY) Error() string {
	return "Parsed line is empty"
}

func loadVideoList(videoList []video) ([]video, error) {
	file, errOpenFile := os.Open(listFileNamePath)
	if errOpenFile != nil {
		return nil, errOpenFile
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		v, err := parseLine(scanner.Text())

		if err == nil {
			videoList = append(videoList, v)
		} else if _, ok := err.(ERR_PARSE_EMPTY); ok == false {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return videoList, nil
}

func parseLine(line string) (v video, err error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return v, ERR_PARSE_EMPTY{}
	}

	var subParts []string

	subParts = strings.Split(line, " ")

	var vURL string
	var typeFlag string
	var folderFlag string

	if len(subParts) == 0 {
		return v, ERR_PARSE_EMPTY{}
	}

	v.parsingLine = line

	if len(subParts) > 0 {
		vURL = subParts[0]

		if strings.HasPrefix(vURL, youtubeURLFull) == false && strings.HasPrefix(vURL, youtubeURL) == false {
			v.setError(fmt.Sprintf("Parsing url failed for video line: %s, expecting url starts as %s or %s", vURL, youtubeURLFull, youtubeURL), nil)
		}

		v.link = vURL
	}

	// DEFAULT
	v.createMp3 = true
	v.keepVideo = true

	if len(subParts) > 1 {
		// type of video process
		typeFlag = strings.TrimSpace(subParts[1])

		if typeFlag == typeBoth || typeFlag == typeBothShort { //both
			v.createMp3 = true
			v.keepVideo = true
		} else if typeFlag == typeMusic || typeFlag == typeMusicShort || typeFlag == typeMusicLong {
			v.createMp3 = true
			v.keepVideo = false
		} else if typeFlag == typeVideo || typeFlag == typeVideoShort || typeFlag == typeVideoLong {
			v.keepVideo = true
			v.createMp3 = false
		} else {
			err := errors.New("unknown type to convert to")
			core.LogError(err, "Wrong type: "+typeFlag)
			return v, err
		}
	}

	if len(subParts) > 2 {
		// folder process
		folderFlag = strings.TrimSpace(subParts[2])

		if folderFlag != "" {
			if val, ok := folders[folderFlag]; ok && val != "" {
				v.moveDir = val
			} else {
				// check folder exists or create folder
				folder, err := checkOrCreateFolder(folderFlag)

				if err != nil {
					core.LogError(errors.New("folder not exist, using root folder"), "folder can not be created: "+folderFlag)
					v.moveDir = folders[root]
				} else {
					folders[folderFlag] = folder
					v.moveDir = folder
				}

			}
		}

	}

	return v, nil
}

func checkOrCreateFolder(folderIn string) (folderOut string, err error) {
	if !strings.HasPrefix(folderIn, "/") {
		folderIn = folders[root] + "/" + folderIn
	}

	if strings.HasSuffix(folderIn, "/") {
		folderIn = strings.TrimSuffix(folderIn, "/")
	}

	exists := core.FolderExists(folderIn)

	if !exists {
		err = os.MkdirAll(folderIn, 0774)
		if err != nil {
			return "", err
		}
	}

	return folderIn, nil
}

func loadSettings() {
	settingsFile, err := os.Open("settings.json")
	if err != nil {
		core.LogError(err, "settings.json file can not be open")
		os.Exit(1)
	}
	defer settingsFile.Close()

	byteValue, _ := ioutil.ReadAll(settingsFile)
	errUn := json.Unmarshal(byteValue, &folders)

	if errUn != nil {
		core.LogError(errUn, "error while unmarshal settings.json file")
	}

	if _, ok := folders[root]; !ok {
		core.LogError(nil, "please add 'root' declaration in file settings.json")
		os.Exit(1)
	}

	for key, val := range folders {
		folders[key], err = checkOrCreateFolder(val)
	}

	// core.PrintE(folders)
}

var allVideosCount int

func main() {

	loadSettings()

	videoList = []video{}

	if core.FileExists(videoHTMLpagePath) {
		var err error
		videoList, err = parseVideoHTML(videoList)

		if err != nil {
			core.LogError(err, "Fail to parse html txt list.")
			os.Exit(1)
		}

		if len(videoList) > 0 {
			fmt.Printf("Html videos loaded, count: %d \n", len(videoList))
			time.Sleep(1 * time.Second)
			err = moveWrapper(videoHTMLpagePath, videoHTMLpagePathLoaded)
			if err != nil {
				core.LogError(err, "can not move file "+videoHTMLpagePath)
				os.Exit(1)
			}
		}
	}

	if core.FileExists(listHTMLpagePath) {
		var err error
		origLength := len(videoList)
		videoList, err = parseListHTML(videoList)

		if err != nil {
			core.LogError(err, "Fail to parse html txt list.")
			os.Exit(1)
		}

		if len(videoList)-origLength > 0 {
			fmt.Printf("Html list videos loaded, count: %d \n", len(videoList)-origLength)
			time.Sleep(1 * time.Second)
			err = moveWrapper(listHTMLpagePath, listHTMLpagePathLoaded)
			if err != nil {
				core.LogError(err, "can not move file "+listHTMLpagePath)
				os.Exit(1)
			}
		}
	}

	videoList, err := loadVideoList(videoList)

	allVideosCount = len(videoList)
	fmt.Printf("All videos loaded, count: %d \n", allVideosCount)

	if err != nil {
		core.LogError(err, "Fail to load list file.")
		os.Exit(1)
	}

	doWork(videoList)

	// TODO: delete list.txt content
}

func processErrors(wg *sync.WaitGroup, errorChannel chan video) {
	defer wg.Done()

	firstErr := true

	for v := range errorChannel {
		if v.hasError == true {
			if firstErr {
				fmt.Println("\n\nErrors:")
				firstErr = false
			}

			fmt.Println(v.err)
			v.printMe()
		}
	}
}

func doWork(videoList []video) {

	videoChannel := make(chan video)
	errorChannel := make(chan video)

	wgProcess := new(sync.WaitGroup)
	wgError := new(sync.WaitGroup)

	wgError.Add(1)
	go processErrors(wgError, errorChannel)

	for i := 0; i < 2; i++ {
		wgProcess.Add(1)
		go processVideoList(wgProcess, videoChannel, errorChannel)
	}

	for index := 0; index < len(videoList); index++ {
		v := loadVideoOptions(videoList[index])
		v = loadVideoNames(v)
		videoChannel <- v
	}

	close(videoChannel)
	wgProcess.Wait()

	close(errorChannel)
	wgError.Wait()

}

func parseVideoHTML(videoList []video) ([]video, error) {

	file, errOpenFile := os.Open(videoHTMLpagePath)

	if errOpenFile != nil {
		return nil, errOpenFile
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 1 && strings.Contains(line, "video-title") {

			if strings.Contains(line, "href") {

				pos := strings.Index(line, "href=\""+listURLPart)

				if pos > 0 {
					// example: href="/watch?v=AspGAZyZzLc">
					length := 27
					substring := line[pos+6 : pos+length] // skip hfref="

					lastChar := substring[len(substring):]

					if lastChar == "\"" || lastChar == "&" {

						unknownURL := errors.New(substring + " is not known url for parsing")
						return nil, unknownURL
					}

					substring = substring[:len(substring)-1]
					videoURL := youtubeURL + substring

					v, err := parseLine(videoURL + " " + typeVideo + " " + listHTMLFolder)

					if err == nil {
						videoList = appendIfMissing(videoList, v)
					} else if _, ok := err.(ERR_PARSE_EMPTY); ok == false {
						return nil, err
					}

				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return videoList, nil
}

func parseListHTML(videoList []video) ([]video, error) {

	file, errOpenFile := os.Open(listHTMLpagePath)

	// test line:     <a class="yt-simple-endpoint style-scope ytd-playlist-video-renderer" href="/watch?v=wVp_VlkWqxI&amp;list=WL&amp;index=552">
	if errOpenFile != nil {
		return nil, errOpenFile
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 1 && strings.Contains(line, "yt-simple-endpoint") {

			if strings.Contains(line, "href") {

				pos := strings.Index(line, "href=\""+listURLPart)
				if pos > 0 {
					// example: href="/watch?v=AspGAZyZzLc">
					length := 27
					substring := line[pos+6 : pos+length] // skip hfref="

					lastChar := substring[len(substring):]

					if lastChar == "\"" || lastChar == "&" {

						unknownURL := errors.New(substring + " is not known url for parsing")
						return nil, unknownURL
					}

					substring = substring[:len(substring)-1]
					videoURL := youtubeURL + substring

					v, err := parseLine(videoURL + " " + typeVideo + " " + listHTMLFolder)

					if err == nil {
						videoList = appendIfMissing(videoList, v)
					} else if _, ok := err.(*ERR_PARSE_EMPTY); ok == false {
						return nil, err
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return videoList, nil
}

func appendIfMissing(list []video, v video) []video {
	// TODO: cyklus odzadu pole bude v tomhle pripade rychlejsi, protoze stejny video bude vzdy na konci
	for _, elInSlice := range list {
		if elInSlice.link == v.link {
			return list
		}
	}

	return append(list, v)
}
