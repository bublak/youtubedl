// Go
//
//  Wrapper for youtube-dl. Can create also mp3 from list of youtube links.
//  Usage: 1.) prepare list of youtube videos with requested output (video/ mp3) 2.) run
//  Program is trying to obtain suitable quality, and then lowering, if better not existing.
//
//  OS: linux, macosx
//  !keep python3 /usr/local/bin/youtube-dl updated
//
//  Before usage, set in settings.json path, where files will be downloaded:   "root": "/Users/PATH/downloads",
//  usage: create file list.txt and put youtube links inside, one link per line:
//
//  https://www.youtube.com/watch?v=r0hirs3zrDI OPTION FOLDER_NAME OUTPUT FILE NAME
//
//  example list.txt:
//  	https://www.youtube.com/watch?v=u3m2kQ-tOEk
//  	https://www.youtube.com/watch?v=kkJtHXfRH74&list=WL&index=24&t=0s m
//  	https://www.youtube.com/watch?v=qGyPuey-1Jw v
//  	https://www.youtube.com/watch?v=qGyPuey-1Jw v newFolder
//  	https://www.youtube.com/watch?v=qGyPuey-1Jw v FolderKeyDefinedInSettingsFile
//	https://www.youtube.com/watch?v=mKf1x3CALAE m FolderName Horace Silver - Song For My Father
//
//
//   Options:
//				m only mp3
//				v only video
//				no option || b Keep both (video and mp3) (DEFAULT)
//
// == Build & run ==
//  go build; ./youtube
//
//  @requrired python3, /usr/local/bin/youtube-dl, ffmpeg, and for build golang 1.18
//  @author    Pavel Filipcik
//  @year      2017-2022

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
	"youtube/codes/core"
)

var listFileNamePath = "list.txt"
var videoHTMLpagePath = "videolisthtml.txt"
var videoHTMLpagePathLoaded = "videolisthtml.txt_d"

var listHTMLpagePath = "/Volumes/Pavel/work/codes/youtubedl/playlisthtml.txt"
var listHTMLpagePathLoaded = "playlisthtml.txt_d"

const (
	numberOfProcesses = 4
	root              = "root"
	youtubeURL        = "https://youtube.com"
	youtubeURLFull    = "https://www.youtube.com/"
	listHTMLFolder    = "listHTMLFolder"
	typeMusic         = "typeMusic"
	typeVideo         = "typeVideo"
	typeMusicShort    = "m"
	typeMusicLong     = "mp3"
	typeVideoShort    = "v"
	typeVideoLong     = "video"
	typeBoth          = "both"
	typeBothShort     = "b"
	listURLPart       = "/watch?v="
)

var folders map[string]string

var statusChannel chan video

type ytVideoOptions struct {
	musicIndex string
	videoIndex string
}

type video struct {
	counter        string
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
		v.errMsg = msg
	} else {
		v.err = errors.New(msg + err.Error())
		v.errMsg = msg
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

	if v.createMp3 == false {
		return
	}

	var fullName = v.getFullName()
	fmt.Printf("create mp3 %s.| %s+ \n\n", v.counter, fullName)

	if !core.FileExists(fullName) {
		v.setError("Create of mp3 file failed: missing video file to be converted to mp3.", nil)
		return
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
			return
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

	fmt.Printf("\nStarted download of %s.| %s (%s)\n", v.counter, videoFullName, v.link)

	var downloadErrorVideoIndex error
	var downloadErrorMusicIndex error

	if v.ytVideoOptions.videoIndex != "" {
		downloadErrorVideoIndex = v.runExternalDownloadCommand(v.ytVideoOptions.videoIndex, videoFullName, v.link)
	}

	if downloadErrorVideoIndex != nil {
		fmt.Printf("\n Error download of videoIndex %s.| %s (%s)!\n", v.counter, videoFullName, v.link)
	}

	if v.createMp3 && v.ytVideoOptions.videoIndex != v.ytVideoOptions.musicIndex {
		downloadErrorMusicIndex = v.runExternalDownloadCommand(v.ytVideoOptions.musicIndex, videoFullName, v.link)
	}

	if downloadErrorMusicIndex != nil {
		fmt.Printf("\n  Error download of musicIndex %s.| %s (%s)!\n", v.counter, videoFullName, v.link)
	}

	if downloadErrorMusicIndex == nil && downloadErrorVideoIndex == nil {
		fmt.Printf("\nFinished download of %s.| %s (%s)\n", v.counter, videoFullName, v.link)
	}
}

func (v *video) runExternalDownloadCommand(index, fullName, link string) error {
	cmd := exec.Command("python3", "/usr/local/bin/youtube-dl", "--newline", "-f", index, "-o", fullName, link)

	// create a pipe for the output of the script
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		v.setError("Command python3 /usr/local/bin/youtube-dl failed with: ", err)
		fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
		return err
	}

	scanner := bufio.NewScanner(cmdReader)

	go func(vidLink string, position string) {
		var counter int = 0
		fmt.Printf("\n") // this line will be overwritten with output
		for scanner.Scan() {
			counter++
			if counter%10 == 0 {
				fmt.Printf("\033[F \t %s.| %s > %s\n", position, vidLink, scanner.Text())
			}
		}
	}(v.link, v.counter)

	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		v.setError("Command python3 /usr/local/bin/youtube-dl failed with: ", err)
		return err
	}

	err = cmd.Wait()
	if err != nil {
		// TODO is it helpful to restart download here?
		fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
		v.setError("Command python3 /usr/local/bin/youtube-dl failed with: ", err)
		return err
	}

	return nil
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
	if v.hasError == false && v.videoName == "" {
		cmd := exec.Command("python3", "/usr/local/bin/youtube-dl", "-e", v.link)
		out, errCO := cmd.CombinedOutput()

		if errCO != nil {
			core.LogError(errCO, "cmd.Run() for name failed with")
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

func processVideoList(wg *sync.WaitGroup, videoChannel chan video, statusChannel chan video) {
	defer wg.Done()

	for v := range videoChannel {
		processVideosCounter.increment()

		v.counter = processVideosCounter.getValueAsString()

		fmt.Printf("  Processing video %s/%d: %s | %s\n\n", v.counter, allVideosCount, v.videoName, v.link)
		if v.hasError == false {
			v.downloadVideoIndexesFiles()

			if !v.hasError {
				v.getMp3()

				if !v.hasError {
					v.removeVideo()
					v.moveFile()
					fmt.Printf("  Success:  %s/%d: %s | %s\n\n", v.counter, allVideosCount, v.videoName, v.link)
				}
			}

		}

		statusChannel <- v
	}
}

func (v *video) moveFile() {
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

		if err != nil {
			if _, ok := err.(ERR_PARSE_EMPTY); ok {
				continue
			}

			return nil, err
		}

		videoList = append(videoList, v)
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

	if len(subParts) > 3 {
		var specName string
		for i := 3; i < len(subParts); i++ {
			specName = specName + subParts[i] + "_"

		}

		v.videoName = core.CleanCharactersFromString(specName)
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
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Println("Program killed!")
		// small chance, someone will write into status channel when its closed -> panic
		// will not happen, if videos are downloading
		close(statusChannel)

		// TODO active running downloads are not in the list, as they are not failed neither OK

		time.Sleep(10 * time.Millisecond) // get some time, for printing results

		os.Exit(0)
	}()

	loadSettings()

	videoList = []video{}

	videoList, err := loadListFromFiles(videoList)

	if err != nil {
		core.LogError(err, "can not load html list files, "+err.Error())
		os.Exit(1)
	}

	videoList, err = loadVideoList(videoList)

	allVideosCount = len(videoList)
	fmt.Printf("All videos loaded, count: %d \n", allVideosCount)

	if err != nil {
		core.LogError(err, "Fail to load list file.")
		fmt.Println(`example of list.txt: 
[[
https://www.youtube.com/watch?v=0Q8-FSlWHZg m musicFolder
https://www.youtube.com/watch?v=tBjyOENZnmo v videoFolder name of video
]]`)
		os.Exit(1)
	}

	doWork(videoList)

	// TODO: delete list.txt content
}

func loadListFromFiles(videoList []video) ([]video, error) {

	var fileName = listHTMLpagePath
	if core.FileExists(listHTMLpagePath) {
		var err error
		origLength := len(videoList)
		videoList, err = parseListHTML(fileName, videoList)

		if err != nil {
			return nil, fmt.Errorf("Fail to parse html txt list %s, error: %s", fileName, err.Error())
		}

		if len(videoList)-origLength > 0 {
			fmt.Printf("Html list videos loaded, count: %d \n", len(videoList)-origLength)
			time.Sleep(1 * time.Second)
			err = moveWrapper(listHTMLpagePath, listHTMLpagePathLoaded)
			if err != nil {
				return nil, fmt.Errorf("Can not move file %s, error: %s", fileName, err.Error())
			}
		}
	}

	return videoList, nil
}

func processResults(wg *sync.WaitGroup, statusChannel chan video) {
	defer wg.Done()

	firstErr := true

	var errList []string
	var okList []string

	for v := range statusChannel {
		if v.hasError == true {
			if firstErr {
				fmt.Println("\n\nErrors:")
				firstErr = false
			}

			fmt.Printf("%s| %s\n", v.counter, v.errMsg)
			fmt.Printf("%s| %s\n", v.counter, v.err.Error())
			v.printMe()
			fmt.Println()
			errList = append(errList, fmt.Sprintf("  Error:  %s| %s | %s\n", v.counter, v.parsingLine, v.videoName))
		} else {
			okList = append(okList, fmt.Sprintf("  Success:  %s| %s | %s\n", v.counter, v.link, v.videoName))
		}
	}

	for _, v := range okList {
		fmt.Println(v)
	}
	fmt.Println("----------------------")

	for _, v := range errList {
		fmt.Println(v)
	}

}

func doWork(videoList []video) {

	videoChannel := make(chan video)
	statusChannel = make(chan video)

	wgProcess := new(sync.WaitGroup)
	wgError := new(sync.WaitGroup)

	wgError.Add(1)
	go processResults(wgError, statusChannel)

	for i := 0; i < numberOfProcesses; i++ {
		wgProcess.Add(1)
		go processVideoList(wgProcess, videoChannel, statusChannel)
	}

	for index := 0; index < len(videoList); index++ {
		v := loadVideoOptions(videoList[index])
		v = loadVideoNames(v)
		videoChannel <- v
	}

	close(videoChannel)
	wgProcess.Wait()

	close(statusChannel)
	wgError.Wait()

}

func parseListHTML(fileName string, videoList []video) ([]video, error) {
	file, errOpenFile := os.Open(fileName)

	// test line:     <a class="yt-simple-endpoint style-scope ytd-playlist-video-renderer" href="/watch?v=wVp_VlkWqxI&amp;list=WL&amp;index=552">
	if errOpenFile != nil {
		return nil, errOpenFile
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	var maxCapacity = 1024 * 1024
	buf := make([]byte, 0, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	var downloadType string
	var outputFolder string
	var counter int = 0

	for scanner.Scan() {
		counter++
		line := scanner.Text()

		if counter == 1 {
			downloadType = strings.Trim(line, " ")
			continue
		}

		if counter == 2 {
			outputFolder = strings.Trim(line, " ")
			outputFolder = core.CleanCharactersFromString(outputFolder)
			if len(outputFolder) == 0 {
				return nil, fmt.Errorf("Missing outputfolder %s", fileName)
			}
			continue
		}

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

					v, err := parseLine(videoURL + " " + downloadType + " " + outputFolder)

					if err != nil {
						if _, ok := err.(*ERR_PARSE_EMPTY); ok == false {
							return nil, err
						}
					}

					videoList = appendIfMissing(videoList, v)
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
	for _, elInSlice := range list {
		if elInSlice.link == v.link {
			return list
		}
	}

	return append(list, v)
}
