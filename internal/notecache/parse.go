package notecache

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

type parsedFile struct {
	document    *Document
	occurrences []OccurrenceRecord
	links       []Link
}

type OccurrenceRecord struct {
	ID       string
	File     string
	Position int
}

type heading struct {
	level int
	title string
	id    string
	typ   string
}

var (
	titleRe       = regexp.MustCompile(`(?i)^#\+TITLE:\s*(.*)$`)
	beginBlockRe  = regexp.MustCompile(`(?i)^\s*#\+BEGIN_([[:alnum:]_-]+)\b`)
	endBlockRe    = regexp.MustCompile(`(?i)^\s*#\+END_([[:alnum:]_-]+)\b`)
	headingRe     = regexp.MustCompile(`^(\*+)\s+(.*)$`)
	propertyRe    = regexp.MustCompile(`^\s*:([[:alnum:]_@#%]+):\s*(.*?)\s*$`)
	idLinkRe      = regexp.MustCompile(`\[\[id:([^\]\[]+)\]`)
	plainIDLinkRe = regexp.MustCompile(`\bid:([[:alnum:]][^][[:space:]]*)`)
)

func parseOrg(path, text string, documentCandidate bool, tasksFile bool) parsedFile {
	lines := splitLines(text)
	starts := charLineStarts(lines)
	result := parsedFile{}
	var fileID, fileTitle, fileIDError string
	current := -1
	headings := make([]heading, 0)
	stack := make([]int, 0)
	var block string
	inDrawer := false
	inIgnoredDrawer := false
	drawerOwner := -2
	drawerProps := map[string]string{}
	drawerIDs := []propertyValue{}
	propertyAllowed := true
	inTaskArtifacts := false
	taskArtifactSource := ""
	docRelationDepth := 0
	legacyRelationDepth := 0

	for i, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r\n")
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if block != "" {
			if m := endBlockRe.FindStringSubmatch(line); m != nil && strings.EqualFold(m[1], block) {
				block = ""
			}
			continue
		}
		if inIgnoredDrawer {
			if strings.EqualFold(trimmed, ":END:") {
				inIgnoredDrawer = false
			}
			continue
		}
		if m := beginBlockRe.FindStringSubmatch(line); m != nil {
			name := strings.ToLower(m[1])
			switch name {
			case "src", "example", "quote", "comment", "commentary", "export", "verse", "center":
				block = name
			}
			continue
		}
		if strings.HasPrefix(lower, "# ") || strings.HasPrefix(lower, "#\t") {
			continue
		}
		if inDrawer {
			if strings.EqualFold(trimmed, ":END:") {
				applyDrawer(path, drawerOwner, drawerProps, drawerIDs, &headings, &fileID, &fileIDError, &result.occurrences)
				inDrawer = false
				drawerOwner = -2
				drawerProps = map[string]string{}
				drawerIDs = nil
				propertyAllowed = false
				continue
			}
			if m := propertyRe.FindStringSubmatch(line); m != nil {
				key := strings.ToUpper(m[1])
				value := strings.TrimSpace(m[2])
				position := starts[i] + charIndex(line, m[2])
				if key == "ID" {
					drawerIDs = append(drawerIDs, propertyValue{value: value, position: position})
				} else {
					drawerProps[key] = value
				}
			}
			continue
		}
		if m := titleRe.FindStringSubmatch(line); m != nil && fileTitle == "" {
			fileTitle = strings.TrimSpace(m[1])
			continue
		}
		if m := headingRe.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			title := cleanHeadingTitle(m[2])
			for len(stack) > 0 && headings[stack[len(stack)-1]].level >= level {
				stack = stack[:len(stack)-1]
			}
			headings = append(headings, heading{level: level, title: title})
			current = len(headings) - 1
			stack = append(stack, current)
			propertyAllowed = true
			inTaskArtifacts = false
			taskArtifactSource = ""
			docRelationDepth = 0
			legacyRelationDepth = 0
			if level == 1 && title == "Link" {
				docRelationDepth = level
			}
			if level >= 2 && title == "관련 작업" {
				legacyRelationDepth = level
			}
			continue
		}
		if current >= 0 {
			if docRelationDepth > 0 && headings[current].level <= docRelationDepth && headings[current].title != "Link" {
				docRelationDepth = 0
			}
			if legacyRelationDepth > 0 && headings[current].level <= legacyRelationDepth && headings[current].title != "관련 작업" {
				legacyRelationDepth = 0
			}
		}
		if isDrawerStart(trimmed) {
			if trimmed == ":PROPERTIES:" && propertyAllowed {
				inDrawer = true
				drawerOwner = current
				drawerProps = map[string]string{}
				drawerIDs = nil
			} else {
				inIgnoredDrawer = true
			}
			continue
		}
		if propertyAllowed && isPlanningLine(trimmed) {
			continue
		}
		if current >= 0 && strings.TrimSpace(line) == "산출물:" {
			inTaskArtifacts = true
			taskArtifactSource = nearestHeadingID(headings, stack)
			propertyAllowed = false
			continue
		}
		if inTaskArtifacts {
			if !strings.HasPrefix(strings.TrimLeft(line, " \t"), "- ") {
				inTaskArtifacts = false
				taskArtifactSource = ""
			}
		}
		if inTaskArtifacts && taskArtifactSource != "" {
			result.links = append(result.links, linksInLine(path, line, starts[i], taskArtifactSource, "task-artifact")...)
			continue
		}
		if docRelationDepth > 0 || legacyRelationDepth > 0 {
			source := documentSourceID(fileID, headings)
			kind := "doc-task"
			if legacyRelationDepth > 0 {
				kind = "doc-task-legacy"
			}
			if source != "" && managedLinkLine(line) {
				result.links = append(result.links, linksInLine(path, line, starts[i], source, kind)...)
			}
		}
		if strings.TrimSpace(line) != "" {
			propertyAllowed = false
		}
	}
	if documentCandidate {
		doc := Document{File: path, IdentityError: fileIDError}
		if fileID != "" {
			doc.ID = fileID
			doc.Title = fileTitle
			doc.Kind = "file"
			if doc.Title == "" {
				doc.Title = firstHeadingTitle(headings, filepath.Base(path))
			}
		} else {
			candidates := topLevelTypeCandidates(headings)
			switch len(candidates) {
			case 1:
				doc.ID = candidates[0].id
				doc.Title = candidates[0].title
				doc.Kind = "legacy-root"
			case 0:
				// Match the Emacs catalog contract: project document files are
				// listed even when no document identity can be derived, but
				// title/kind stay empty unless an identity exists or is
				// ambiguous.
			default:
				doc.Kind = "legacy-root"
				doc.IdentityError = "multiple document ID candidates"
			}
		}
		result.document = &doc
	}
	_ = tasksFile
	return result
}

type propertyValue struct {
	value    string
	position int
}

func applyDrawer(path string, owner int, props map[string]string, ids []propertyValue, headings *[]heading, fileID, fileIDError *string, occurrences *[]OccurrenceRecord) {
	for index, id := range ids {
		if id.value == "" {
			continue
		}
		*occurrences = append(*occurrences, OccurrenceRecord{ID: id.value, File: path, Position: id.position})
		if index > 0 {
			continue
		}
		if owner >= 0 && owner < len(*headings) {
			(*headings)[owner].id = id.value
		} else if owner == -1 {
			if *fileID == "" {
				*fileID = id.value
			} else if *fileID != id.value {
				*fileIDError = "duplicate_preamble_id"
			}
		}
	}
	if typ, ok := props["TYPE"]; ok && owner >= 0 && owner < len(*headings) {
		(*headings)[owner].typ = typ
	}
}

func isDrawerStart(trimmed string) bool {
	if len(trimmed) < 3 || !strings.HasPrefix(trimmed, ":") || !strings.HasSuffix(trimmed, ":") {
		return false
	}
	name := strings.Trim(trimmed, ":")
	return name != "" && !strings.ContainsAny(name, " \t")
}

func isPlanningLine(trimmed string) bool {
	if trimmed == "" {
		return true
	}
	return strings.HasPrefix(trimmed, "SCHEDULED:") ||
		strings.HasPrefix(trimmed, "DEADLINE:") ||
		strings.HasPrefix(trimmed, "CLOSED:")
}

func documentSourceID(fileID string, headings []heading) string {
	if fileID != "" {
		return fileID
	}
	candidates := topLevelTypeCandidates(headings)
	if len(candidates) == 1 {
		return candidates[0].id
	}
	return ""
}

func topLevelTypeCandidates(headings []heading) []heading {
	var candidates []heading
	for _, h := range headings {
		if h.level == 1 && h.title != "Link" && h.typ != "" && h.id != "" {
			candidates = append(candidates, h)
		}
	}
	return candidates
}

func nearestHeadingID(headings []heading, stack []int) string {
	for i := len(stack) - 1; i >= 0; i-- {
		if headings[stack[i]].id != "" {
			return headings[stack[i]].id
		}
	}
	return ""
}

func firstHeadingTitle(headings []heading, fallback string) string {
	for _, h := range headings {
		if h.title != "" && h.title != "Link" {
			return h.title
		}
	}
	return strings.TrimSuffix(fallback, filepath.Ext(fallback))
}

func cleanHeadingTitle(title string) string {
	title = strings.TrimSpace(title)
	fields := strings.Fields(title)
	if len(fields) > 1 {
		switch fields[0] {
		case "TODO", "DONE", "NEXT", "WAIT", "CANCELLED":
			title = strings.TrimSpace(strings.TrimPrefix(title, fields[0]))
		}
	}
	if strings.HasSuffix(title, ":") {
		if index := strings.LastIndex(title[:len(title)-1], " :"); index >= 0 {
			title = strings.TrimSpace(title[:index])
		}
	}
	return title
}

func managedLinkLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "[[id:")
}

func linksInLine(path, line string, lineStart int, sourceID, kind string) []Link {
	matches := idLinkRe.FindAllStringSubmatchIndex(line, -1)
	links := make([]Link, 0, len(matches))
	for _, m := range matches {
		id := line[m[2]:m[3]]
		links = append(links, Link{File: path, SourceID: sourceID, TargetID: id, Kind: kind, Position: lineStart + charIndex(line, line[m[2]:])})
	}
	if len(links) == 0 {
		for _, m := range plainIDLinkRe.FindAllStringSubmatchIndex(line, -1) {
			id := strings.TrimRight(line[m[2]:m[3]], ".,;:")
			links = append(links, Link{File: path, SourceID: sourceID, TargetID: id, Kind: kind, Position: lineStart + charIndex(line, line[m[2]:])})
		}
	}
	return links
}

func splitLines(text string) []string {
	lines := strings.SplitAfter(text, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func charLineStarts(lines []string) []int {
	starts := make([]int, len(lines))
	pos := 1
	for i, line := range lines {
		starts[i] = pos
		pos += utf8.RuneCountInString(line)
	}
	return starts
}

func charIndex(line, suffix string) int {
	byteIndex := strings.Index(line, suffix)
	if byteIndex < 0 {
		return 0
	}
	return utf8.RuneCountInString(line[:byteIndex])
}

func contentHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func sortParsed(p *parsedFile) {
	sort.Slice(p.occurrences, func(i, j int) bool {
		if p.occurrences[i].ID == p.occurrences[j].ID {
			return p.occurrences[i].Position < p.occurrences[j].Position
		}
		return p.occurrences[i].ID < p.occurrences[j].ID
	})
	sort.Slice(p.links, func(i, j int) bool {
		if p.links[i].File == p.links[j].File {
			return p.links[i].Position < p.links[j].Position
		}
		return p.links[i].File < p.links[j].File
	})
}
