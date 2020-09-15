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
//  !keep youtube-dl updated
//
//  usage: create file list.txt and put youtube links inside, one link per line, example:
//  list.txt:
//  	https://www.youtube.com/watch?v=u3m2kQ-tOEk
//  	https://www.youtube.com/watch?v=kkJtHXfRH74&list=WL&index=24&t=0s m
//  	https://www.youtube.com/watch?v=qGyPuey-1Jw v
//
//  @requrired youtube-dl, ffmpeg
//  @author    Pavel Filipcik
//  @year      2017-2020

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"youtube/codes/core"
)

var listFileNamePath string = "list.txt"
var listHTMLpagePath = "yyoutube-playlist-video-html.txt"
var downloadRootDir = "./files"

const (
	youtubeURL     = "https://youtube.com"
	youtubeURLFull = "https://www.youtube.com/"
	listHTMLFolder = "listHtmlFolder"
	typeMusic      = "mp3"
	typeVideo      = "video"
	typeBoth       = "both"
)

var folders map[string]string = map[string]string{
	typeMusic:      "mp3",   // basic type music
	typeVideo:      "video", // basic type video
	"vm":           "movie",
	"i":            "Sadhguru",
	"vh":           "vimhoff",
	"s":            "spoken",
	"now":          "now",
	"mor":          "moravske",
	"hou":          "house",
	"moric":        "moric",
	"lid":          "lidove",
	"tech":         "techno",
	"kouzla":       "kouzla",
	"vtipy":        "vtipy",
	"mh":           "mp3/house",
	listHTMLFolder: "/fromHtml",
}

type ytVideoOptions struct {
	musicIndex string
	videoIndex string
}

type video struct {
	isEmpty        bool
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
	v.isEmpty = true
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
	}
}

var videoList = []video{}

func (v *video) getFullName() string {
	return v.videoName + "." + v.videoExtension
}

func (v *video) getFullMp3Name() string {
	return v.videoName + ".mp3"
}

func (v *video) getMp3() {

	if v.createMp3 == true {
		fmt.Println("get mp3")

		var fullName = v.getFullName()
		//var fullName = "/Users/pf/work/codesgo/youtube/Zwei.mp4"

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

	fmt.Println("stahuju", videoFullName, v.link)

	if v.ytVideoOptions.videoIndex != "" {
		cmd := exec.Command("youtube-dl", "-f", v.ytVideoOptions.videoIndex, "-o", videoFullName, v.link)
		err := cmd.Run()

		if err != nil {
			//fmt.Println("youtube-dl", "-f", v.ytVideoOptions.videoIndex, "-o", videoFullName, v.link)

			errString := fmt.Sprintln("youtube-dl", "-f", v.ytVideoOptions.videoIndex, "-o", videoFullName, v.link)

			v.setError("Command youtube-dl failed with: "+errString, err)
		}

		// fmt.Println(string(out))
	}

	if v.createMp3 && v.ytVideoOptions.videoIndex != v.ytVideoOptions.musicIndex {
		cmd := exec.Command("youtube-dl", "-f", v.ytVideoOptions.musicIndex, "-o", videoFullName, v.link)
		err := cmd.Run()

		if err != nil {
			errString := fmt.Sprint("youtube-dl", "-f", v.ytVideoOptions.musicIndex, "-o", videoFullName, v.link)

			//fmt.Println("youtube-dl", "-f", v.ytVideoOptions.musicIndex, "-o", videoFullName, v.link)
			v.setError("Command youtube-dl failed with: "+errString, err)
		}
	}
}

func (v *video) removeVideo() {
	if v.hasError == false && v.keepVideo == false {
		fullName := v.getFullName()
		fmt.Println("mazu video: ", fullName)
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

func loadVideoNames(list []video) {
	for i := 0; i < len(list); i++ {
		v := &list[i]
		if v.hasError == false {
			cmd := exec.Command("youtube-dl", "-e", v.link)
			out, errCO := cmd.CombinedOutput()

			if errCO != nil {
				core.LogError(errCO, "cmd.Run() failed with")
			}

			bufferOutput := string(out)

			bufferOutput = core.CleanCharactersFromString(bufferOutput)
			v.videoName = bufferOutput
		}
	}
}

func loadVideoOptions(list []video) {
	for i := 0; i < len(list); i++ {
		v := &list[i]
		if v.hasError == false {
			cmd := exec.Command("youtube-dl", "-F", v.link)
			out, errCO := cmd.CombinedOutput()

			if errCO != nil {
				v.setError("get video index failed", errCO)
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
	}
}

func processVideoList(wg *sync.WaitGroup, list []video) {
	defer wg.Done()
	numberOfVideos := len(list)

	// for index, v := range list {
	for index := 0; index < len(list); index++ {
		v := &list[index]
		fmt.Printf("processing %d out of %d \n", index+1, numberOfVideos)

		if v.hasError == false {
			v.downloadVideoIndexesFiles()

			if v.hasError == true {
				continue
			}

			v.getMp3()
			v.removeVideo()
			moveFiles(v)
		}
	}

	fmt.Println("\n\nErrors:")
	// process errors
	for _, v := range list {
		if v.hasError == true {
			fmt.Println(&v.err)
			v.printMe()
		}
	}

}

func moveFiles(v *video) {
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
		errMoveVideo := os.Rename(nameOfFile, downloadRootDir+"/"+moveToDir+"/"+nameOfFile)

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

		errMoveMp3 := os.Rename(nameOfFile, downloadRootDir+"/"+moveToDir+"/"+nameOfFile)

		if errMoveMp3 != nil {
			v.setError("Moving mp3 file failed for file:", errMoveMp3)
		}
	}

}

func loadVideoList(videoList []video) ([]video, error) {
	file, errOpenFile := os.Open(listFileNamePath)
	if errOpenFile != nil {
		return nil, errOpenFile
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		v := parseLine(scanner.Text())

		if v.isEmpty == false {
			videoList = append(videoList, v)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return videoList, nil
}

func parseLine(line string) (v video) {
	if line == "" {
		v.isEmpty = true
		return v
	}
	//fmt.Println("parsovani: ", line)

	var subParts []string

	subParts = strings.Split(line, " ")

	var vURL string
	var videoFlag string

	if len(subParts) == 0 {
		v.isEmpty = true
		return v
	}

	v.parsingLine = line

	if len(subParts) > 0 {
		vURL = subParts[0]

		if strings.HasPrefix(vURL, youtubeURLFull) == false && strings.HasPrefix(vURL, youtubeURL) == false {
			v.setError(fmt.Sprintf("Parsing url failed for video line: %s", vURL), nil)
		}

		v.link = vURL
	}

	// DEFAULT
	v.createMp3 = true
	v.keepVideo = true

	if len(subParts) > 1 {
		videoFlag = strings.TrimSpace(subParts[1])

		if videoFlag == typeBoth { //both
			v.createMp3 = true
			v.keepVideo = true
		}

		if videoFlag == typeMusic {
			v.createMp3 = true
			v.keepVideo = false
		}

		if videoFlag == typeVideo {
			v.keepVideo = true
			v.createMp3 = false
		}
	}

	if len(subParts) > 2 {
		// this second param must be contained in folders mapping
		videoFlag = strings.TrimSpace(subParts[2])

		if videoFlag == "" {

		} else {
			if val, ok := folders[videoFlag]; ok {
				v.moveDir = val
			} else {
				core.LogError(errors.New("missing FOLDER mapping"), "no mapping for folder")
				os.Exit(1)
			}
		}

	}

	return v
}

func main() {
	parser := true

	videoList = []video{}

	if parser {
		var err error
		videoList, err = parseHTML(videoList)

		if err != nil {
			core.LogError(err, "Fail to parse html txt list.")
			os.Exit(1)
		}
	}

	videoList, err := loadVideoList(videoList)

	if err != nil {
		core.LogError(err, "Fail to load list file.")
	}

	loadVideoOptions(videoList)
	loadVideoNames(videoList)

	splitPos := len(videoList) / 2

	videoList1 := videoList[:splitPos]
	videoList2 := videoList[splitPos:]

	var wg sync.WaitGroup

	wg.Add(1)
	go processVideoList(&wg, videoList1)
	wg.Add(1)
	go processVideoList(&wg, videoList2)

	wg.Wait()

	// TODO move files to external disk
	// .      a) udelat kontrolu, jestli je dostupny
	// c)put in git -> gitgnore readme files
	// settings file with folder names
	// constants for everything
	// czech -> english
}

func parseHTML(videoList []video) ([]video, error) {

	file, errOpenFile := os.Open(listHTMLpagePath)

	if errOpenFile != nil {
		return nil, errOpenFile
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 1 && strings.Contains(line, "video-title") {

			if strings.Contains(line, "href") {

				pos := strings.Index(line, "href=\"")
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

					v := parseLine(videoURL + " " + typeMusic + " " + listHTMLFolder)

					videoList = append(videoList, v)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return videoList, nil
}