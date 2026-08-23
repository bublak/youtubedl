// Go
//
//  Wrapper for yt-dlp. Can create also mp3 from list of youtube links.
//  Usage: 1.) prepare list of youtube videos with requested output (video/ mp3) 2.) run
//  Format selection is delegated to yt-dlp: video as bestvideo+bestaudio merged to mp4,
//  mp3 extracted from bestaudio via yt-dlp/ffmpeg.
//
//  OS: linux, macosx
//  !keep yt-dlp updated
//  install: brew install yt-dlp
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
//    [empty folder name (.) + author name (if song file name does not have - char, its used only as author name, rest is loaded from youtube title)]
//    https://www.youtube.com/watch?v=eCG7RuI-80M m . The Church
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
//  @requrired yt-dlp, ffmpeg, and for build golang 1.22
//  @author    Pavel Filipcik
//  @year      2017-2024

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

const listHTMLAlbumSeparator = "**startnew**"

var listHTMLpagePath = "/Users/pavelfilipcik/mywork/codes/youtubedl/playlisthtml.txt"
var listHTMLpagePathLoaded = "playlisthtml.txt_d"

const (
	numberOfProcesses = 1
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

type video struct {
	counter            string
	hasError           bool
	err                error
	errMsg             string
	link               string
	createMp3          bool
	keepVideo          bool
	parsingLine        string
	videoName          string
	authorName         string
	videoAlbumPosition int
	videoExtension     string
	moveDir            string
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

func printVideoList(videoList []video) {
	for _, v := range videoList {
		fmt.Println(v.parsingLine, ", ", v.getFullName())
	}
}

func (v *video) getAlbumNamePosition() string {
	if v.videoAlbumPosition == 0 {
		return ""
	}

	//TODO there is not knowledge about count of videos belongs to album, now formating for 2 places
	return fmt.Sprintf("%02d-", v.videoAlbumPosition)
}

func (v *video) getAuthorName() string {
	if v.authorName != "" {
		return v.authorName + "-"
	}

	return ""
}

func (v *video) getFullName() string {
	return v.getAlbumNamePosition() + v.getAuthorName() + v.videoName + "." + v.videoExtension
}

func (v *video) getFullMp3Name() string {
	return v.getAlbumNamePosition() + v.getAuthorName() + v.videoName + "_b.mp3"
}

func (v *video) getMp3() {
	if !v.createMp3 {
		return
	}

	mp3Name := v.getFullMp3Name()
	fmt.Printf("create mp3 %s.| %s+ \n\n", v.counter, mp3Name)

	if core.FileExists(mp3Name) {
		return
	}

	// yt-dlp downloads best audio and converts it to mp3 via ffmpeg by itself;
	// convert into temp name first, so interrupted run can not leave truncated mp3
	// behind, which would be skipped as already existing on the next run
	tempName := strings.TrimSuffix(mp3Name, ".mp3") + ".tmp.mp3"
	outputTemplate := strings.TrimSuffix(tempName, ".mp3") + ".%(ext)s"
	cmd := exec.Command("yt-dlp", "--no-warnings", "-f", "bestaudio/best",
		"-x", "--audio-format", "mp3", "--audio-quality", "160K",
		"-o", outputTemplate, v.link)

	out, errCO := cmd.CombinedOutput()

	if errCO != nil {
		v.setError(fmt.Sprintf("Create of mp3 file failed: yt-dlp audio extract, output: %s ", string(out)), errCO)
		return
	}

	if !core.FileExists(tempName) {
		// should not happen, maybe full disk?
		v.setError(fmt.Sprintf("Mp3 file was not created: %s \n output: %s \n", tempName, string(out)), nil)
		return
	}

	if errRename := os.Rename(tempName, mp3Name); errRename != nil {
		v.setError("Create of mp3 file failed: rename of temp mp3 file.", errRename)
	}
}

func (v *video) downloadVideoIndexesFiles() {
	if !v.keepVideo {
		// mp3-only entries download audio directly in getMp3, no video file needed
		return
	}

	videoFullName := v.getFullName()

	fmt.Printf("\nStarted download of %s.| %s (%s)\n", v.counter, videoFullName, v.link)

	downloadError := v.runExternalDownloadCommand("bv*+ba/b", videoFullName, v.link)

	if downloadError != nil {
		fmt.Printf("\n Error download of video %s.| %s (%s)!\n", v.counter, videoFullName, v.link)
	} else {
		fmt.Printf("\nFinished download of %s.| %s (%s)\n", v.counter, videoFullName, v.link)
	}
}

func (v *video) runExternalDownloadCommand(formatSelector, fullName, link string) error {
	cmd := exec.Command("yt-dlp", "--no-warnings", "--newline", "-f", formatSelector,
		"--merge-output-format", "mp4", "-o", fullName, link)

	// collect stderr, so real error reason from yt-dlp is visible in the report
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	// create a pipe for the output of the script
	cmdReader, err := cmd.StdoutPipe()

	if err != nil {
		v.setError("Command yt-dlp failed with: ", err)
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
		v.setError("Command yt-dlp failed with after Start: ", err)
		return err
	}

	err = cmd.Wait()
	if err != nil {
		// TODO is it helpful to restart download here?
		fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
		fmt.Fprintln(os.Stderr, stderrBuf.String())
		v.setError("Command yt-dlp failed with after Wait: "+stderrBuf.String(), err)
		return err
	}

	return nil
}

func loadVideoNames(v video) video {
	if !v.hasError && v.videoName == "" {
		cmd := exec.Command("yt-dlp", "--no-warnings", "-e", v.link)
		// cmd := exec.Command("yt-dlp", "--no-check-certificate", "-e", v.link)

		out, errCO := cmd.CombinedOutput()

		if errCO != nil {
			core.LogError(errCO, "cmd.Run() for name failed with")
		}

		bufferOutput := string(out)

		fmt.Println(bufferOutput)

		bufferOutput = core.CleanCharactersFromString(bufferOutput)
		v.videoName = bufferOutput
	}

	return v
}

func loadVideoOptions(v video) video {
	// format selection is left to yt-dlp itself (bv*+ba/b merged into mp4),
	// old fixed format indexes 18/22 are no longer offered by youtube
	v.videoExtension = "mp4"

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
		time.Sleep(time.Duration(5 * time.Second))
		processVideosCounter.increment()

		v.counter = processVideosCounter.getValueAsString()

		fmt.Printf("  Processing video %s/%d: %s | %s\n\n", v.counter, allVideosCount, v.videoName, v.link)
		if !v.hasError {
			v.downloadVideoIndexesFiles()

			if !v.hasError {
				v.getMp3()

				if !v.hasError {
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

	var subParts []string = strings.Split(line, " ")

	var vURL string
	var typeFlag string
	var folderFlag string

	if len(subParts) == 0 {
		return v, ERR_PARSE_EMPTY{}
	}

	v.parsingLine = line

	if len(subParts) > 0 {
		vURL = subParts[0]

		if !strings.HasPrefix(vURL, youtubeURLFull) && !strings.HasPrefix(vURL, youtubeURL) {
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

		if folderFlag != "" && folderFlag != "." {
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

		specName = core.CleanCharactersFromString(specName)

		if !strings.Contains(specName, "-") {
			// only author name
			v.authorName = specName
		} else {
			// whole song name with author
			v.videoName = specName
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
			return nil, fmt.Errorf("fail to parse html txt list %s, error: %s", fileName, err.Error())
		}

		if len(videoList)-origLength > 0 {
			fmt.Printf("Html list videos loaded, count: %d \n", len(videoList)-origLength)
			time.Sleep(1 * time.Second)
			err = moveWrapper(listHTMLpagePath, listHTMLpagePathLoaded)
			if err != nil {
				return nil, fmt.Errorf("can not move file %s, error: %s", fileName, err.Error())
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
		if v.hasError {
			if firstErr {
				fmt.Println("\n\nErrors:")
				firstErr = false
			}

			fmt.Printf("%s| %s\n", v.counter, v.errMsg)
			fmt.Printf("%s| %s\n", v.counter, v.err.Error())
			v.printMe()
			fmt.Println()
			errList = append(errList, fmt.Sprintf("  Error:  %s| %s %s\n", v.counter, v.parsingLine, v.getFullName()))
		} else {
			okList = append(okList, fmt.Sprintf("  Success:  %s| %s %s\n", v.counter, v.link, v.getFullName()))
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
		time.Sleep(time.Duration(3 * time.Second))
	}

	close(videoChannel)
	wgProcess.Wait()

	close(statusChannel)
	wgError.Wait()

}

// extractHTMLTitleAttr extracts the value of the title="..." attribute from an HTML line.
func extractHTMLTitleAttr(line string) string {
	const prefix = `title="`
	idx := strings.Index(line, prefix)
	if idx < 0 {
		return ""
	}
	rest := line[idx+len(prefix):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// extractAuthorFromVideoTitle tries to parse an artist/author name from a YouTube video title.
// Handles patterns:
//   - "Author - Song Name"           → returns "Author"
//   - "21. Song by Author | Extra"   → returns "Author"
func extractAuthorFromVideoTitle(title string) string {
	title = strings.ReplaceAll(title, "&amp;", "&")
	title = strings.ReplaceAll(title, "&nbsp;", " ")
	title = strings.TrimSpace(title)

	if title == "" {
		return ""
	}

	// "Author - Song" pattern
	if idx := strings.Index(title, " - "); idx > 0 {
		return strings.TrimSpace(title[:idx])
	}

	// "Song by Author | Extra" pattern
	lowerTitle := strings.ToLower(title)
	if byIdx := strings.Index(lowerTitle, " by "); byIdx >= 0 {
		after := title[byIdx+4:]
		if pipeIdx := strings.Index(after, " | "); pipeIdx >= 0 {
			return strings.TrimSpace(after[:pipeIdx])
		}
		return strings.TrimSpace(after)
	}

	return ""
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
	var authorName string
	var counter int = 0
	var albumPosition int = 1
	var isChange = false
	var lastVideoIdx int = -1
	var lookingForChannelName bool = false

	for scanner.Scan() {
		counter++
		line := scanner.Text()

		if strings.Contains(line, listHTMLAlbumSeparator) {
			counter = 0
			albumPosition = 1
			authorName = ""
			outputFolder = ""
			lastVideoIdx = -1
			lookingForChannelName = false

			continue
		}

		if counter == 1 {
			downloadType = strings.Trim(line, " ")
			continue
		}

		if counter == 2 {
			outputFolder = strings.Trim(line, " ")
			outputFolder = core.CleanCharactersFromString(outputFolder)
			if len(outputFolder) == 0 {
				return nil, fmt.Errorf("missing outputfolder %s", fileName)
			}
			continue
		}

		if counter == 3 {
			authorName = strings.Trim(line, " ")
			authorName = core.CleanCharactersFromString(authorName)
		}

		// Channel name appears a few lines after the video-title line; backfill author if needed.
		if lookingForChannelName && strings.Contains(line, "id=\"text\"") && strings.Contains(line, "ytd-channel-name") {
			channelName := extractHTMLTitleAttr(line)
			// YouTube auto-generated artist channels append " - Topic"; strip it.
			channelName = strings.TrimSuffix(channelName, " - Topic")
			channelName = strings.TrimSpace(channelName)
			if channelName != "" && lastVideoIdx >= 0 {
				videoList[lastVideoIdx].authorName = core.CleanCharactersFromString(channelName)
			}
			lookingForChannelName = false
		}

		if len(line) > 1 && strings.Contains(line, "yt-simple-endpoint") && strings.Contains(line, "id=\"video-title\"") {
			lookingForChannelName = false // reset any pending state from previous entry

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
						if _, ok := err.(*ERR_PARSE_EMPTY); !ok {
							return nil, err
						}
					}

					effectiveAuthor := authorName
					if effectiveAuthor == "" {
						titleAttr := extractHTMLTitleAttr(line)
						effectiveAuthor = core.CleanCharactersFromString(extractAuthorFromVideoTitle(titleAttr))
					}

					v.videoAlbumPosition = albumPosition
					v.authorName = effectiveAuthor
					videoList, isChange = appendIfMissing(videoList, v)

					if isChange {
						albumPosition++
						lastVideoIdx = len(videoList) - 1
						// If author still empty, look for channel name on upcoming lines.
						lookingForChannelName = (effectiveAuthor == "")
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

func appendIfMissing(list []video, v video) ([]video, bool) {
	for _, elInSlice := range list {
		if elInSlice.link == v.link {
			return list, false
		}
	}

	return append(list, v), true
}
