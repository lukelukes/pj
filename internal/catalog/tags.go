package catalog

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

var (
	ErrEmptyTag   = errors.New("tag cannot be empty")
	ErrInvalidTag = errors.New("invalid tag")
	tagPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9_.:/+-]*$`)
)

func NormalizeTag(tag string) (string, error) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag == "" {
		return "", ErrEmptyTag
	}
	if !tagPattern.MatchString(tag) || strings.HasSuffix(tag, ":") {
		return "", fmt.Errorf("%w %q: use letters, digits, _, ., :, /, +, -; start with a letter or digit and do not end with a colon", ErrInvalidTag, tag)
	}
	return tag, nil
}

func NormalizeTags(tags []string) ([]string, error) {
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		value, err := NormalizeTag(tag)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, value)
	}
	return sortedTags(normalized), nil
}

func normalizeTagsLenient(tags []string) []string {
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag != "" {
			normalized = append(normalized, tag)
		}
	}
	return sortedTags(normalized)
}

func sortedTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	slices.Sort(tags)
	return slices.Compact(tags)
}

// SplitTags tokenizes comma- or whitespace-separated input without validating it.
func SplitTags(raw string) []string {
	tags := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
	if len(tags) == 0 {
		return nil
	}
	return tags
}

func ParseTags(raw string) ([]string, error) { return NormalizeTags(SplitTags(raw)) }

func TagSet(projects []Project) []string {
	tags := make(map[string]struct{})
	for _, p := range projects {
		for _, tag := range p.Tags {
			tags[tag] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(tags))
}

func (p Project) WithTags(tags []string) Project {
	p.Tags = slices.Clone(tags)
	return p
}

// AddTags validates additions before changing the project.
func (p *Project) AddTags(tags ...string) error {
	normalized, err := NormalizeTags(append(slices.Clone(p.Tags), tags...))
	if err != nil {
		return err
	}
	p.Tags = normalized
	return nil
}

func (p *Project) RemoveTags(tags ...string) {
	p.Tags = slices.DeleteFunc(slices.Clone(p.Tags), func(tag string) bool { return slices.Contains(tags, tag) })
	if len(p.Tags) == 0 {
		p.Tags = nil
	}
}

func (p Project) HasTag(tag string) bool { return slices.Contains(p.Tags, tag) }
