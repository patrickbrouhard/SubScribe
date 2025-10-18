package yt

import "regexp"

var ytRegex = regexp.MustCompile(`(?i)https?://(www\.)?(youtube\.com/watch\?v=|youtu\.be/)`)

func IsYouTubeURL(s string) bool {
	return ytRegex.MatchString(s)
}
