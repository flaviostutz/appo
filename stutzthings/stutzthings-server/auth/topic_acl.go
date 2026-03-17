package auth

import "strings"

func MatchTopicFilter(filter string, topic string) bool {
	filterParts := splitTopic(filter)
	topicParts := splitTopic(topic)
	if len(filterParts) == 0 || len(topicParts) == 0 {
		return false
	}

	for fi, ti := 0, 0; ; fi, ti = fi+1, ti+1 {
		if fi == len(filterParts) {
			return ti == len(topicParts)
		}
		if ti == len(topicParts) {
			return fi == len(filterParts)-1 && filterParts[fi] == "#"
		}

		switch filterParts[fi] {
		case "#":
			return fi == len(filterParts)-1
		case "+":
			continue
		default:
			if filterParts[fi] != topicParts[ti] {
				return false
			}
		}
	}
}

func AnyMatch(filters []string, topic string) bool {
	for _, filter := range filters {
		if MatchTopicFilter(filter, topic) {
			return true
		}
	}
	return false
}

func FilterCanMatchPrefix(filter string, prefix []string, totalSegments int) bool {
	parts := splitTopic(filter)
	if len(parts) == 0 || len(prefix) == 0 || len(prefix) > totalSegments {
		return false
	}
	return canMatchPrefix(parts, prefix, totalSegments, 0, 0)
}

func AnyFilterCanMatchPrefix(filters []string, prefix []string, totalSegments int) bool {
	for _, filter := range filters {
		if FilterCanMatchPrefix(filter, prefix, totalSegments) {
			return true
		}
	}
	return false
}

func canMatchPrefix(filterParts []string, prefix []string, totalSegments int, fi int, ti int) bool {
	if ti == totalSegments {
		return fi == len(filterParts) || (fi == len(filterParts)-1 && filterParts[fi] == "#")
	}
	if fi == len(filterParts) {
		return false
	}

	segment := filterParts[fi]
	if segment == "#" {
		return fi == len(filterParts)-1
	}
	if ti < len(prefix) {
		if segment != "+" && segment != prefix[ti] {
			return false
		}
	}
	return canMatchPrefix(filterParts, prefix, totalSegments, fi+1, ti+1)
}

func splitTopic(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil
		}
	}
	return parts
}
