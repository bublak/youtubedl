# youtubedl

Wrapper for youtube-dl written in Go. Can create also mp3 from list of youtube links. Program is trying to obtain suitable quality, and then lowering, if better not existing.
OS: linux, macosx
!keep python3 /usr/local/bin/youtube-dl updated

  
#### Usage

* Before usage, set in settings.json path, where files will be downloaded:   "root": "/Users/PATH/downloads",
* create file list.txt and put youtube links inside, one link per line:
* prepare list of youtube videos with requested output (video/ mp3)
* run

##### example list.txt
```
  https://www.youtube.com/watch?v=r0hirs3zrDI OPTION FOLDER_NAME OUTPUT FILE NAME
  https://www.youtube.com/watch?v=u3m2kQ-tOEk
  https://www.youtube.com/watch?v=kkJtHXfRH74&list=WL&index=24&t=0s m
  https://www.youtube.com/watch?v=qGyPuey-1Jw v
  https://www.youtube.com/watch?v=qGyPuey-1Jw v newFolder
  https://www.youtube.com/watch?v=qGyPuey-1Jw v FolderKeyDefinedInSettingsFile
  https://www.youtube.com/watch?v=mKf1x3CALAE m FolderName Horace Silver - Song For My Father
```

##### Download options
```
  m only mp3
  v only video
  no option || b Keep both (video and mp3) (DEFAULT)
```
#### Build & run


* requrired python3, /usr/local/bin/youtube-dl, ffmpeg, and for build golang 1.18
* go build
* ./youtube

@year      2017-2022
