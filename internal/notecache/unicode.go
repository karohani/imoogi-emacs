package notecache

import "strings"

const (
	hangulSBase  = 0xAC00
	hangulLBase  = 0x1100
	hangulVBase  = 0x1161
	hangulTBase  = 0x11A7
	hangulLCount = 19
	hangulVCount = 21
	hangulTCount = 28
	hangulNCount = hangulVCount * hangulTCount
)

func displayPath(path string) string {
	return composeHangul(path)
}

func composeHangul(text string) string {
	runes := []rune(text)
	var builder strings.Builder
	builder.Grow(len(text))
	for i := 0; i < len(runes); {
		l := int(runes[i]) - hangulLBase
		if l < 0 || l >= hangulLCount || i+1 >= len(runes) {
			builder.WriteRune(runes[i])
			i++
			continue
		}
		v := int(runes[i+1]) - hangulVBase
		if v < 0 || v >= hangulVCount {
			builder.WriteRune(runes[i])
			i++
			continue
		}
		syllable := rune(hangulSBase + (l*hangulVCount+v)*hangulTCount)
		i += 2
		if i < len(runes) {
			t := int(runes[i]) - hangulTBase
			if t > 0 && t < hangulTCount {
				syllable += rune(t)
				i++
			}
		}
		builder.WriteRune(syllable)
	}
	return builder.String()
}
