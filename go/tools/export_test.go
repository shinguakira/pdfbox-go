package tools

// What the review round's cases need to reach from outside the package.

// EscapeMarkdownForTest is escapeMarkdown, which is Java's private static
// escape of PDFText2Markdown.
func EscapeMarkdownForTest(chars string) string { return escapeMarkdown(chars) }
