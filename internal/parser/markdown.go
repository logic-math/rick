package parser

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownDocument holds both the AST and source bytes for proper text extraction
type MarkdownDocument struct {
	AST    ast.Node
	Source []byte
}

// ParseMarkdown parses Markdown content and returns the AST
func ParseMarkdown(content string) (ast.Node, error) {
	md := goldmark.New()
	reader := text.NewReader([]byte(content))
	node := md.Parser().Parse(reader)
	return node, nil
}

// ParseMarkdownWithSource parses Markdown and returns both AST and source for text extraction
func ParseMarkdownWithSource(content string) (*MarkdownDocument, error) {
	md := goldmark.New()
	source := []byte(content)
	reader := text.NewReader(source)
	node := md.Parser().Parse(reader)
	return &MarkdownDocument{
		AST:    node,
		Source: source,
	}, nil
}

// ExtractHeading extracts headings of a specific level from the AST
// level: 1 for h1, 2 for h2, etc.
func ExtractHeading(node ast.Node, level int) []string {
	return ExtractHeadingWithSource(node, level, nil)
}

// ExtractHeadingWithSource extracts headings with source bytes for proper text extraction
func ExtractHeadingWithSource(node ast.Node, level int, source []byte) []string {
	var headings []string
	walkNode(node, func(n ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.WalkContinue
		}
		heading, ok := n.(*ast.Heading)
		if ok && heading.Level == level {
			text := extractTextFromNode(heading, source)
			headings = append(headings, text)
		}
		return ast.WalkContinue
	})
	return headings
}

// ExtractListItems extracts all list items from the AST
func ExtractListItems(node ast.Node) []string {
	return ExtractListItemsWithSource(node, nil)
}

// ExtractListItemsWithSource extracts all list items with source bytes
func ExtractListItemsWithSource(node ast.Node, source []byte) []string {
	var items []string
	walkNode(node, func(n ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.WalkContinue
		}
		item, ok := n.(*ast.ListItem)
		if ok {
			text := extractTextFromNode(item, source)
			if text != "" {
				items = append(items, text)
			}
		}
		return ast.WalkContinue
	})
	return items
}

// ExtractParagraph extracts all paragraph text from the AST
func ExtractParagraph(node ast.Node) []string {
	return ExtractParagraphWithSource(node, nil)
}

// ExtractParagraphWithSource extracts all paragraph text with source bytes
func ExtractParagraphWithSource(node ast.Node, source []byte) []string {
	var paragraphs []string
	walkNode(node, func(n ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.WalkContinue
		}
		para, ok := n.(*ast.Paragraph)
		if ok {
			text := extractTextFromNode(para, source)
			if text != "" {
				paragraphs = append(paragraphs, text)
			}
		}
		return ast.WalkContinue
	})
	return paragraphs
}

// ExtractCodeBlock extracts all code blocks from the AST
func ExtractCodeBlock(node ast.Node) []string {
	return ExtractCodeBlockWithSource(node, nil)
}

// ExtractCodeBlockWithSource extracts all code blocks with source bytes
func ExtractCodeBlockWithSource(node ast.Node, source []byte) []string {
	var blocks []string
	walkNode(node, func(n ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.WalkContinue
		}
		codeBlock, ok := n.(*ast.FencedCodeBlock)
		if ok {
			code := extractCodeBlockContent(codeBlock, source)
			if code != "" {
				blocks = append(blocks, code)
			}
		}
		return ast.WalkContinue
	})
	return blocks
}

// Helper function to walk through AST nodes
func walkNode(node ast.Node, fn func(ast.Node, bool) ast.WalkStatus) {
	if node == nil {
		return
	}

	queue := []ast.Node{node}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		status := fn(current, true)
		if status == ast.WalkStop {
			return
		}
		if status == ast.WalkSkipChildren {
			continue
		}

		child := current.FirstChild()
		for child != nil {
			queue = append(queue, child)
			child = child.NextSibling()
		}
	}
}

// Helper function to extract text from a node
func extractTextFromNode(node ast.Node, source []byte) string {
	var buf bytes.Buffer
	extractTextRecursive(node, &buf, source)
	return buf.String()
}

// Helper function to recursively extract text
func extractTextRecursive(node ast.Node, buf *bytes.Buffer, source []byte) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.Text:
		// Extract text from segment
		segment := n.Segment
		if !segment.IsEmpty() && source != nil && segment.Start >= 0 && segment.Stop <= len(source) {
			buf.Write(segment.Value(source))
		}
	case *ast.String:
		buf.Write(n.Value)
	case *ast.CodeSpan:
		// Extract code span content
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			extractTextRecursive(child, buf, source)
		}
	case *ast.Heading, *ast.Paragraph, *ast.ListItem, *ast.Emphasis:
		// Recursively extract from container nodes
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			extractTextRecursive(child, buf, source)
		}
	}
}

// Helper function to extract code block content with source
func extractCodeBlockContent(codeBlock *ast.FencedCodeBlock, source []byte) string {
	var buf bytes.Buffer
	lines := codeBlock.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		if source != nil && line.Start >= 0 && line.Stop <= len(source) {
			buf.Write(line.Value(source))
		}
	}
	return buf.String()
}

// GetCodeBlockLanguage returns the language identifier of a code block
func GetCodeBlockLanguage(codeBlock *ast.FencedCodeBlock) string {
	return GetCodeBlockLanguageWithSource(codeBlock, nil)
}

// GetCodeBlockLanguageWithSource returns the language identifier with source bytes
func GetCodeBlockLanguageWithSource(codeBlock *ast.FencedCodeBlock, source []byte) string {
	info := codeBlock.Info
	if info == nil {
		return ""
	}
	segment := info.Segment
	if source != nil && segment.Start >= 0 && segment.Stop <= len(source) {
		lang := segment.Value(source)
		return string(lang)
	}
	return ""
}

// ParseMarkdownWithMetadata parses Markdown and returns both AST and raw source
func ParseMarkdownWithMetadata(content string) (ast.Node, string, error) {
	node, err := ParseMarkdown(content)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse markdown: %w", err)
	}
	return node, content, nil
}
