package apiv1

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

// K20: a comment is plain text, never Markdown. The 40 production comments
// holding * or _ and the 3 holding ** are literal characters, so running them
// through the Markdown pipeline would silently rewrite them. Only these three
// token shapes are structure; everything else is text.
var commentTokenRe = regexp.MustCompile(
	`/image/[0-9a-f]{64}(?:_[a-z0-9]+)?` +
		`|\[@[^\]\n]*\]\(kungal-user:[0-9]+\)` +
		`|\[#[0-9]+\]\(kungal-reply:[0-9]+\)`)

var (
	commentMentionRe = regexp.MustCompile(`^\[@[^\]\n]*\]\(kungal-user:([0-9]+)\)$`)
	commentReplyRe   = regexp.MustCompile(`^\[#([0-9]+)\]\(kungal-reply:([0-9]+)\)$`)
)

type commentDocBuilder struct {
	cdn    string
	images map[string]imageclient.ImageMeta
	users  map[int]userclient.User
}

// convertCommentBodies turns stored comment texts into restricted content
// documents, resolving every image hash and mentioned user in one batch each.
func (s *Service) convertCommentBodies(ctx context.Context, sources []string) ([]content.ContentDocument, *problem.Problem) {
	normalized := make([]string, len(sources))
	var hashes []string
	seenHash := map[string]struct{}{}
	var userIDs []int
	seenUser := map[int]struct{}{}
	for i, src := range sources {
		normalized[i] = markdown.NormalizeStoredContent(src)
		for _, tok := range commentTokenRe.FindAllString(normalized[i], -1) {
			if hash, _, ok := markdown.ParseContentImageRef(tok); ok {
				if _, dup := seenHash[hash]; !dup {
					seenHash[hash] = struct{}{}
					hashes = append(hashes, hash)
				}
				continue
			}
			if m := commentMentionRe.FindStringSubmatch(tok); m != nil {
				id, _ := strconv.Atoi(m[1])
				if _, dup := seenUser[id]; id > 0 && !dup {
					seenUser[id] = struct{}{}
					userIDs = append(userIDs, id)
				}
			}
		}
	}

	b := &commentDocBuilder{cdn: s.cdn}
	if len(hashes) > 0 && s.convert != nil && s.convert.Images != nil {
		b.images = s.convert.Images(hashes)
	}
	if len(userIDs) > 0 {
		users, p := s.lookupUsers(ctx, userIDs)
		if p != nil {
			return nil, p
		}
		b.users = users
	}

	out := make([]content.ContentDocument, len(normalized))
	for i, src := range normalized {
		out[i] = b.document(src)
	}
	return out, nil
}

func (b *commentDocBuilder) document(src string) content.ContentDocument {
	inlines := b.inlines(src)
	if len(inlines) == 0 {
		return content.NewDocument(nil)
	}
	return content.NewDocument(content.Blocks{content.NewParagraph(inlines)})
}

func (b *commentDocBuilder) inlines(src string) content.Inlines {
	var out content.Inlines
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	for i, line := range lines {
		if i > 0 {
			out = append(out, content.NewBreak())
		}
		out = append(out, b.line(line)...)
	}
	return out
}

func (b *commentDocBuilder) line(line string) content.Inlines {
	var out content.Inlines
	pos := 0
	for _, span := range commentTokenRe.FindAllStringIndex(line, -1) {
		node, ok := b.token(line[span[0]:span[1]])
		if !ok {
			continue
		}
		if text := line[pos:span[0]]; text != "" {
			out = append(out, content.NewText(text))
		}
		out = append(out, node)
		pos = span[1]
	}
	if text := line[pos:]; text != "" {
		out = append(out, content.NewText(text))
	}
	return out
}

func (b *commentDocBuilder) token(tok string) (content.Inline, bool) {
	if hash, variant, ok := markdown.ParseContentImageRef(tok); ok {
		url := imageclient.MainURL(b.cdn, hash, "webp")
		if variant != "" {
			url = imageclient.VariantURL(b.cdn, hash, variant, "webp")
		}
		return content.NewImage(url, "", repr.NewImage(b.cdn, hash, b.imageMeta(hash))), true
	}
	if m := commentMentionRe.FindStringSubmatch(tok); m != nil {
		id, err := strconv.Atoi(m[1])
		if err != nil || id < 1 {
			return nil, false
		}
		return content.NewMention(b.userRef(id)), true
	}
	if m := commentReplyRe.FindStringSubmatch(tok); m != nil {
		floor, err1 := strconv.Atoi(m[1])
		replyID, err2 := strconv.Atoi(m[2])
		if err1 != nil || err2 != nil || floor < 1 || replyID < 1 {
			return nil, false
		}
		return content.NewReplyReference(replyID, floor), true
	}
	return nil, false
}

func (b *commentDocBuilder) imageMeta(hash string) *imageclient.ImageMeta {
	if b.images == nil {
		return nil
	}
	m, ok := b.images[hash]
	if !ok {
		return nil
	}
	return &m
}

func (b *commentDocBuilder) userRef(id int) repr.UserRef {
	if u, ok := b.users[id]; ok {
		return repr.NewUserRef(b.cdn, u)
	}
	return repr.DeletedUserRef(id)
}
